package p2p

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dnsSeedOutboundConnectionThreshold = 2
	dnsSeedRetryInterval                = 10 * time.Minute
)

type BootstrapFailure struct {
	Address string
	Error   string
}

type BootstrapResult struct {
	Attempted          int
	Connected          int
	ConnectedAddresses []string
	DNSLookups         int
	Failures           []BootstrapFailure
}

type BootstrapDiagnostics struct {
	LastAttemptUTC int64  `json:"last_attempt_utc,omitempty"`
	LastDNSSeedUTC int64  `json:"last_dns_seed_utc,omitempty"`
	LastFailure    string `json:"last_failure,omitempty"`
}

// Bootstrap connects to compiled fixed seeds plus operator/cache overrides.
// If the outbound target is still not met, optional DNS seeds are resolved.
// Individual bootstrap failures are non-fatal.
func (n *Node) BootstrapDiagnostics() BootstrapDiagnostics {
	n.mu.RLock()
	defer n.mu.RUnlock()

	result := BootstrapDiagnostics{
		LastFailure: n.lastBootstrapFailure,
	}
	if !n.lastBootstrapAttempt.IsZero() {
		result.LastAttemptUTC = n.lastBootstrapAttempt.UTC().Unix()
	}
	if !n.lastDNSSeedLookup.IsZero() {
		result.LastDNSSeedUTC = n.lastDNSSeedLookup.UTC().Unix()
	}
	return result
}

func (n *Node) recordBootstrapDiagnostics(result BootstrapResult) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.lastBootstrapAttempt = n.protection.Now().UTC()
	if len(result.Failures) == 0 {
		n.lastBootstrapFailure = ""
		return
	}
	last := result.Failures[len(result.Failures)-1]
	if strings.TrimSpace(last.Address) == "" {
		n.lastBootstrapFailure = last.Error
		return
	}
	n.lastBootstrapFailure = last.Address + ": " + last.Error
}

func (n *Node) Bootstrap(ctx context.Context, overrides []string) BootstrapResult {
	result := BootstrapResult{}
	seen := make(map[string]struct{})

	selfAdvertise := strings.TrimSpace(n.AdvertiseAddress())

	connectCandidates := func(addresses []string) {
		for _, address := range addresses {
			if n.outboundCount() >= n.protection.OutboundTarget {
				return
			}
			address = strings.TrimSpace(address)
			if address == "" {
				continue
			}
			if selfAdvertise != "" && address == selfAdvertise {
				continue
			}
			if _, exists := seen[address]; exists {
				continue
			}
			seen[address] = struct{}{}
			if _, _, err := net.SplitHostPort(address); err != nil {
				result.Attempted++
				result.Failures = append(result.Failures, BootstrapFailure{
					Address: address,
					Error:   ErrInvalidPeerAddress.Error(),
				})
				continue
			}

			result.Attempted++
			connectCtx, cancel := context.WithTimeout(ctx, n.handshakeTimeout)
			err := n.Connect(connectCtx, address)
			cancel()
			if err != nil {
				result.Failures = append(result.Failures, BootstrapFailure{
					Address: address,
					Error:   err.Error(),
				})
				continue
			}
			result.Connected++
			result.ConnectedAddresses = append(result.ConnectedAddresses, address)
		}
	}

	connectCandidates(n.seedCandidates(overrides))
	if n.outboundCount() >= n.protection.OutboundTarget {
		return result
	}
	if !n.shouldQueryDNSSeeds(n.protection.Now()) {
		return result
	}

	dnsCandidates, dnsFailures, dnsLookups := n.resolveDNSSeeds(ctx)
	result.DNSLookups += dnsLookups
	result.Failures = append(result.Failures, dnsFailures...)
	connectCandidates(dnsCandidates)
	return result
}

func (n *Node) seedCandidates(overrides []string) []string {
	seen := make(map[string]struct{})
	seeds := make([]string, 0, len(overrides)+8)
	appendUnique := func(address string) {
		address = strings.TrimSpace(address)
		if address == "" {
			return
		}
		if _, exists := seen[address]; exists {
			return
		}
		seen[address] = struct{}{}
		seeds = append(seeds, address)
	}

	// Bitcoin-Core-derived priority: try remembered/operator peers before
	// compiled fixed seeds. A healthy returning node should reconnect to the
	// live graph without needlessly hammering bootstrap infrastructure.
	for _, address := range overrides {
		appendUnique(address)
	}
	if n.enableV2 {
		for _, address := range n.networkProfile.DefaultSeeds {
			appendUnique(address)
		}
	}
	return seeds
}

func (n *Node) shouldQueryDNSSeeds(now time.Time) bool {
	if !n.enableV2 || len(n.networkProfile.DNSSeeds) == 0 {
		return false
	}

	threshold := dnsSeedOutboundConnectionThreshold
	if n.protection.OutboundTarget < threshold {
		threshold = n.protection.OutboundTarget
	}
	if threshold < 1 {
		threshold = 1
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	outbound := 0
	for _, peer := range n.peers {
		if !peer.Inbound {
			outbound++
		}
	}
	if outbound >= threshold {
		return false
	}
	if !n.lastDNSSeedLookup.IsZero() && now.Sub(n.lastDNSSeedLookup) < dnsSeedRetryInterval {
		return false
	}
	n.lastDNSSeedLookup = now
	return true
}

func (n *Node) resolveDNSSeeds(ctx context.Context) ([]string, []BootstrapFailure, int) {
	if !n.enableV2 || len(n.networkProfile.DNSSeeds) == 0 {
		return nil, nil, 0
	}

	seen := make(map[string]struct{})
	addresses := make([]string, 0)
	failures := make([]BootstrapFailure, 0)
	lookups := 0
	defaultPort := strconv.Itoa(int(n.networkProfile.P2PPort))

	for _, rawSeed := range n.networkProfile.DNSSeeds {
		rawSeed = strings.TrimSpace(rawSeed)
		if rawSeed == "" {
			continue
		}
		lookups++

		host := strings.TrimSuffix(rawSeed, ".")
		port := defaultPort
		if parsedHost, parsedPort, err := net.SplitHostPort(rawSeed); err == nil {
			host = strings.TrimSuffix(parsedHost, ".")
			port = parsedPort
		}
		if host == "" {
			failures = append(failures, BootstrapFailure{
				Address: rawSeed,
				Error:   ErrInvalidPeerAddress.Error(),
			})
			continue
		}

		resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			failures = append(failures, BootstrapFailure{
				Address: rawSeed,
				Error:   err.Error(),
			})
			continue
		}
		sort.Slice(resolved, func(i, j int) bool {
			return resolved[i].IP.String() < resolved[j].IP.String()
		})
		for _, ip := range resolved {
			address := net.JoinHostPort(ip.IP.String(), port)
			if err := validateDiscoveredAddress(address, n.isPublicDiscovery()); err != nil {
				continue
			}
			if _, exists := seen[address]; exists {
				continue
			}
			seen[address] = struct{}{}
			addresses = append(addresses, address)
		}
	}
	sort.Strings(addresses)
	return addresses, failures, lookups
}

// MaintainOutbound attempts discovered peers in stable order until the target
// outbound count is reached. It is best-effort and safe to call repeatedly.
func (n *Node) MaintainOutbound(ctx context.Context) BootstrapResult {
	result := BootstrapResult{}
	peers := n.DiscoveredPeers()
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].NodeID == peers[j].NodeID {
			return peers[i].Address < peers[j].Address
		}
		return peers[i].NodeID < peers[j].NodeID
	})
	for _, peer := range peers {
		if n.outboundCount() >= n.protection.OutboundTarget {
			break
		}
		result.Attempted++
		connectCtx, cancel := context.WithTimeout(ctx, n.handshakeTimeout)
		err := n.Connect(connectCtx, peer.Address)
		cancel()
		if err != nil {
			result.Failures = append(result.Failures, BootstrapFailure{
				Address: peer.Address,
				Error:   err.Error(),
			})
			continue
		}
		result.Connected++
		result.ConnectedAddresses = append(result.ConnectedAddresses, peer.Address)
	}
	return result
}

func (n *Node) BootstrapAndMaintain(ctx context.Context, overrides []string) (result BootstrapResult) {
	defer func() {
		n.recordBootstrapDiagnostics(result)
	}()
	// Frozen v0.1 keeps its historical manual topology: without explicit seed
	// candidates it must not expand via discovered peers. V2 networks use the
	// persistent/discovered peer model introduced by v0.2.6.
	if !n.enableV2 && len(n.seedCandidates(overrides)) == 0 {
		return BootstrapResult{}
	}

	result = n.Bootstrap(ctx, overrides)

	// Discovery can expand after each new outbound peer. Keep the bootstrap
	// work bounded while allowing several waves toward the outbound target.
	for round := 0; round < 4 && n.outboundCount() < n.protection.OutboundTarget; round++ {
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return result
		case <-timer.C:
		}

		next := n.MaintainOutbound(ctx)
		result.Attempted += next.Attempted
		result.Connected += next.Connected
		result.ConnectedAddresses = append(result.ConnectedAddresses, next.ConnectedAddresses...)
		result.Failures = append(result.Failures, next.Failures...)
		if next.Attempted == 0 {
			break
		}
	}
	return result
}
