package p2p

import (
	"context"
	"net"
	"sort"
	"strings"
	"time"
)

type BootstrapFailure struct {
	Address string
	Error   string
}

type BootstrapResult struct {
	Attempted int
	Connected int
	Failures  []BootstrapFailure
}

// Bootstrap connects to compiled seeds plus operator overrides until the
// outbound target is reached. Individual seed failures are non-fatal.
func (n *Node) Bootstrap(ctx context.Context, overrides []string) BootstrapResult {
	result := BootstrapResult{}
	seeds := n.seedCandidates(overrides)
	for _, address := range seeds {
		if n.outboundCount() >= n.protection.OutboundTarget {
			break
		}
		if strings.TrimSpace(address) == "" {
			continue
		}
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
	}
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

	if n.enableV2 {
		for _, address := range n.networkProfile.DefaultSeeds {
			appendUnique(address)
		}
	}
	for _, address := range overrides {
		appendUnique(address)
	}
	return seeds
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
	}
	return result
}

func (n *Node) BootstrapAndMaintain(ctx context.Context, overrides []string) BootstrapResult {
	if len(n.seedCandidates(overrides)) == 0 {
		return BootstrapResult{}
	}
	result := n.Bootstrap(ctx, overrides)

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
		result.Failures = append(result.Failures, next.Failures...)
		if next.Attempted == 0 {
			break
		}
	}
	return result
}
