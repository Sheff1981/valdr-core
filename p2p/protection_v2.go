package p2p

import (
	"errors"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	DefaultIdleTimeout          = 5 * time.Minute
	DefaultPingTimeout          = 30 * time.Second
	DefaultMaxInbound           = 64
	DefaultOutboundTarget       = 8
	DefaultMaxInboundPerIP      = 4
	DefaultBanDuration          = time.Hour
	DefaultMalformedThreshold   = 10
	DefaultDuplicateCacheSize   = 4096
	DefaultMessagesPerSecond    = 64
	DefaultMessageBurst         = 128
	DefaultBytesPerSecond       = 2 * 1024 * 1024
	DefaultByteBurst            = 4 * 1024 * 1024
)

var (
	ErrInboundLimit    = errors.New("P2P inbound connection limit reached")
	ErrPeerBanned      = errors.New("P2P peer IP is temporarily banned")
	ErrPeerRateLimited = errors.New("P2P peer rate limit exceeded")
)

type ProtectionConfig struct {
	IdleTimeout        time.Duration
	PingTimeout        time.Duration
	MaxInbound         int
	OutboundTarget     int
	MaxInboundPerIP    int
	BanDuration        time.Duration
	MalformedThreshold int
	DuplicateCacheSize int
	MessagesPerSecond  float64
	MessageBurst       float64
	BytesPerSecond     float64
	ByteBurst          float64
	Now                func() time.Time
}

func normalizeProtectionConfig(cfg ProtectionConfig) ProtectionConfig {
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = DefaultPingTimeout
	}
	if cfg.MaxInbound <= 0 {
		cfg.MaxInbound = DefaultMaxInbound
	}
	if cfg.OutboundTarget <= 0 {
		cfg.OutboundTarget = DefaultOutboundTarget
	}
	if cfg.MaxInboundPerIP <= 0 {
		cfg.MaxInboundPerIP = DefaultMaxInboundPerIP
	}
	if cfg.BanDuration <= 0 {
		cfg.BanDuration = DefaultBanDuration
	}
	if cfg.MalformedThreshold <= 0 {
		cfg.MalformedThreshold = DefaultMalformedThreshold
	}
	if cfg.DuplicateCacheSize <= 0 {
		cfg.DuplicateCacheSize = DefaultDuplicateCacheSize
	}
	if cfg.MessagesPerSecond <= 0 {
		cfg.MessagesPerSecond = DefaultMessagesPerSecond
	}
	if cfg.MessageBurst <= 0 {
		cfg.MessageBurst = DefaultMessageBurst
	}
	if cfg.BytesPerSecond <= 0 {
		cfg.BytesPerSecond = DefaultBytesPerSecond
	}
	if cfg.ByteBurst <= 0 {
		cfg.ByteBurst = DefaultByteBurst
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return cfg
}

type tokenBucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(rate, burst float64, now time.Time) tokenBucket {
	return tokenBucket{
		rate: rate, burst: burst, tokens: burst, last: now,
	}
}

func (b *tokenBucket) allow(amount float64, now time.Time) bool {
	if amount <= 0 {
		return true
	}
	if now.Before(b.last) {
		b.last = now
	}
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(b.burst, b.tokens+elapsed*b.rate)
		b.last = now
	}
	if b.tokens < amount {
		return false
	}
	b.tokens -= amount
	return true
}

type peerTrafficState struct {
	mu           sync.Mutex
	messages     tokenBucket
	bytes        tokenBucket
	lastActivity time.Time
	awaitingPong bool
	lastPing     uint64
	pingSentAt   time.Time
}

func newPeerTrafficState(cfg ProtectionConfig) *peerTrafficState {
	now := cfg.Now()
	return &peerTrafficState{
		messages:     newTokenBucket(cfg.MessagesPerSecond, cfg.MessageBurst, now),
		bytes:        newTokenBucket(cfg.BytesPerSecond, cfg.ByteBurst, now),
		lastActivity: now,
	}
}

func (s *peerTrafficState) allow(cfg ProtectionConfig, payloadBytes int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := cfg.Now()
	return s.messages.allow(1, now) &&
		s.bytes.allow(float64(payloadBytes), now)
}

func (s *peerTrafficState) markActivity(now time.Time) {
	s.mu.Lock()
	s.lastActivity = now
	s.awaitingPong = false
	s.lastPing = 0
	s.pingSentAt = time.Time{}
	s.mu.Unlock()
}

func (s *peerTrafficState) setAwaitingPong(
	nonce uint64,
	now time.Time,
) {
	s.mu.Lock()
	s.awaitingPong = true
	s.lastPing = nonce
	s.pingSentAt = now
	s.mu.Unlock()
}

func (s *peerTrafficState) idleState() (
	lastActivity time.Time,
	awaiting bool,
	pingSentAt time.Time,
	lastPing uint64,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastActivity, s.awaitingPong, s.pingSentAt, s.lastPing
}

type boundedStringSet struct {
	mu    sync.Mutex
	max   int
	order []string
	set   map[string]struct{}
}

func newBoundedStringSet(max int) *boundedStringSet {
	return &boundedStringSet{
		max: max,
		set: make(map[string]struct{}, max),
	}
}

// Add returns true when value was not already present.
func (c *boundedStringSet) Add(value string) bool {
	if c == nil || value == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.set[value]; exists {
		return false
	}
	c.set[value] = struct{}{}
	c.order = append(c.order, value)
	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.set, oldest)
	}
	return true
}

func remoteIP(address net.Addr) string {
	if address == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(address.String())
	if err != nil {
		return ""
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	return ip.String()
}

func validateDiscoveredAddress(address string, public bool) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil || host == "" || port == "" {
		return ErrInvalidPeerAddress
	}
	ip := net.ParseIP(host)
	if ip == nil {
		if !validDNSName(host) {
			return ErrInvalidPeerAddress
		}
		return nil
	}
	if !public {
		return nil
	}
	if ip.IsUnspecified() ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() {
		return ErrInvalidPeerAddress
	}
	return nil
}

func validDNSName(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	host = strings.TrimSuffix(host, ".")
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 ||
			label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') ||
				(r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') ||
				r == '-' {
				continue
			}
			return false
		}
	}
	return true
}


func (n *Node) reserveInbound(ip string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := n.protection.Now()
	if until, exists := n.bannedUntil[ip]; exists {
		if now.Before(until) {
			return ErrPeerBanned
		}
		delete(n.bannedUntil, ip)
		delete(n.violations, ip)
	}

	active := 0
	activeForIP := 0
	for peerID, peer := range n.peers {
		if !peer.Inbound {
			continue
		}
		active++
		if pc := n.conns[peerID]; pc != nil && pc.remoteIP == ip {
			activeForIP++
		}
	}
	if active+n.inboundPending >= n.protection.MaxInbound {
		return ErrInboundLimit
	}
	if activeForIP+n.inboundPendingByIP[ip] >= n.protection.MaxInboundPerIP {
		return ErrInboundLimit
	}

	n.inboundPending++
	n.inboundPendingByIP[ip]++
	return nil
}

func (n *Node) releaseInboundReservation(ip string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.inboundPending > 0 {
		n.inboundPending--
	}
	if n.inboundPendingByIP[ip] <= 1 {
		delete(n.inboundPendingByIP, ip)
	} else {
		n.inboundPendingByIP[ip]--
	}
}

func (n *Node) isIPBanned(ip string) bool {
	if ip == "" {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	until, exists := n.bannedUntil[ip]
	if !exists {
		return false
	}
	if !n.protection.Now().Before(until) {
		delete(n.bannedUntil, ip)
		delete(n.violations, ip)
		return false
	}
	return true
}

func (n *Node) recordIPViolation(ip string, weight int) bool {
	if ip == "" || weight <= 0 {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	now := n.protection.Now()
	if until, exists := n.bannedUntil[ip]; exists && now.Before(until) {
		return true
	}
	n.violations[ip] += weight
	if n.violations[ip] < n.protection.MalformedThreshold {
		return false
	}
	n.bannedUntil[ip] = now.Add(n.protection.BanDuration)
	n.violations[ip] = 0
	return true
}

func (n *Node) recordPeerViolation(peerID string, weight int) bool {
	n.mu.RLock()
	pc := n.conns[peerID]
	n.mu.RUnlock()
	if pc == nil {
		return false
	}
	return n.recordIPViolation(pc.remoteIP, weight)
}

func (n *Node) allowPeerTraffic(peerID string, payloadBytes int) bool {
	n.mu.RLock()
	pc := n.conns[peerID]
	n.mu.RUnlock()
	if pc == nil || pc.traffic == nil {
		return false
	}
	return pc.traffic.allow(n.protection, payloadBytes)
}

func (n *Node) outboundCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	count := 0
	for _, peer := range n.peers {
		if !peer.Inbound {
			count++
		}
	}
	return count
}

func (n *Node) isPublicDiscovery() bool {
	return n.enableV2 && n.networkProfile.Public
}
