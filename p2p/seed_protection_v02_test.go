package p2p

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestSeedBootstrapContinuesPastOfflineSeed(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	seedA := mustStartNode(t, NodeConfig{
		NodeID:         "seed-a",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer seedA.Close()

	seedB := mustStartNode(t, NodeConfig{
		NodeID:         "seed-b",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer seedB.Close()

	offline := reserveClosedAddress(t)
	targetProfile := profile
	targetProfile.DefaultSeeds = []string{
		offline,
		seedA.Address(),
	}
	target := mustStartNode(t, NodeConfig{
		NodeID:         "bootstrap-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &targetProfile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			OutboundTarget: 2,
		},
	})
	defer target.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := target.Bootstrap(ctx, []string{seedB.Address()})
	if result.Attempted != 3 {
		t.Fatalf("attempted=%d want=3 result=%+v", result.Attempted, result)
	}
	if result.Connected != 2 {
		t.Fatalf("connected=%d want=2 result=%+v", result.Connected, result)
	}
	if len(result.Failures) != 1 || result.Failures[0].Address != offline {
		t.Fatalf("failures=%+v want only offline seed", result.Failures)
	}
	if target.PeerCount() != 2 {
		t.Fatalf("peer count=%d want=2", target.PeerCount())
	}
}

func TestInboundReservationLimits(t *testing.T) {
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	node, err := NewNode(NodeConfig{
		NodeID:         "limit-node",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			MaxInbound:      2,
			MaxInboundPerIP: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := node.reserveInbound("192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	if err := node.reserveInbound("192.0.2.1"); !errors.Is(err, ErrInboundLimit) {
		t.Fatalf("second same-IP reservation error=%v want ErrInboundLimit", err)
	}
	if err := node.reserveInbound("192.0.2.2"); err != nil {
		t.Fatal(err)
	}
	if err := node.reserveInbound("192.0.2.3"); !errors.Is(err, ErrInboundLimit) {
		t.Fatalf("global reservation error=%v want ErrInboundLimit", err)
	}
	node.releaseInboundReservation("192.0.2.1")
	node.releaseInboundReservation("192.0.2.2")
}

func TestTemporaryBanAfterMalformedThreshold(t *testing.T) {
	now := time.Unix(2_000_000_000, 0)
	profile, _ := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	node, err := NewNode(NodeConfig{
		NodeID:         "ban-node",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			MalformedThreshold: 3,
			BanDuration:        time.Hour,
			Now:                func() time.Time { return now },
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ip := "198.51.100.25"
	if node.recordIPViolation(ip, 1) {
		t.Fatal("banned too early after score 1")
	}
	if node.recordIPViolation(ip, 1) {
		t.Fatal("banned too early after score 2")
	}
	if !node.recordIPViolation(ip, 1) {
		t.Fatal("not banned at threshold")
	}
	if !node.isIPBanned(ip) {
		t.Fatal("ban not active")
	}
	now = now.Add(time.Hour + time.Second)
	if node.isIPBanned(ip) {
		t.Fatal("ban did not expire")
	}
}

func TestPublicDiscoveryRejectsUnsafeNumericAddresses(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1:17333",
		"10.0.0.1:17333",
		"0.0.0.0:17333",
		"[::1]:17333",
		"224.0.0.1:17333",
	} {
		if err := validateDiscoveredAddress(address, true); !errors.Is(err, ErrInvalidPeerAddress) {
			t.Fatalf("address %q error=%v want ErrInvalidPeerAddress", address, err)
		}
	}

	if err := validateDiscoveredAddress("203.0.113.10:17333", true); err != nil {
		t.Fatalf("public documentation address rejected: %v", err)
	}
	if err := validateDiscoveredAddress("seed.example.org:17333", true); err != nil {
		t.Fatalf("valid DNS seed rejected: %v", err)
	}
	for _, address := range []string{
		"node1:17333",
		"node-a.example:17333",
		"node-b.example:17333",
	} {
		if err := validateDiscoveredAddress(address, true); err != nil {
			t.Fatalf("valid advertised DNS %q rejected: %v", address, err)
		}
	}
	if err := validateDiscoveredAddress("127.0.0.1:7333", false); err != nil {
		t.Fatalf("private devnet discovery rejected: %v", err)
	}
}

func TestTokenBucketRejectsBurstAboveConfiguredLimit(t *testing.T) {
	now := time.Unix(2_000_000_100, 0)
	cfg := normalizeProtectionConfig(ProtectionConfig{
		MessagesPerSecond: 1,
		MessageBurst:      2,
		BytesPerSecond:    100,
		ByteBurst:         100,
		Now:               func() time.Time { return now },
	})
	state := newPeerTrafficState(cfg)
	if !state.allow(cfg, 10) || !state.allow(cfg, 10) {
		t.Fatal("initial burst unexpectedly rejected")
	}
	if state.allow(cfg, 10) {
		t.Fatal("third immediate message should be rate limited")
	}
	now = now.Add(time.Second)
	if !state.allow(cfg, 10) {
		t.Fatal("rate limit did not replenish")
	}
}

func TestBoundedDuplicateCacheEvictsOldest(t *testing.T) {
	cache := newBoundedStringSet(2)
	if !cache.Add("a") || !cache.Add("b") {
		t.Fatal("new cache entries rejected")
	}
	if cache.Add("a") {
		t.Fatal("duplicate entry reported as new")
	}
	if !cache.Add("c") {
		t.Fatal("third entry rejected")
	}
	if !cache.Add("a") {
		t.Fatal("oldest entry was not evicted")
	}
}

func reserveClosedAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}


func TestIdleWatchdogPingsBeforeDisconnect(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	protection := ProtectionConfig{
		IdleTimeout: 20 * time.Millisecond,
		PingTimeout: 40 * time.Millisecond,
	}

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:         "idle-a",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection:     protection,
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:         "idle-b",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection:     protection,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := nodeA.Connect(ctx, nodeB.Address()); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, nodeA, 1)
	waitForPeerCount(t, nodeB, 1)

	time.Sleep(180 * time.Millisecond)
	if nodeA.PeerCount() != 1 || nodeB.PeerCount() != 1 {
		t.Fatalf(
			"idle keepalive failed: A=%d B=%d",
			nodeA.PeerCount(),
			nodeB.PeerCount(),
		)
	}
}

func TestDiscoveryContinuesBeyondSeedNodes(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	peer := mustStartNode(t, NodeConfig{
		NodeID:         "bootstrap-peer",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer peer.Close()

	seed := mustStartNode(t, NodeConfig{
		NodeID:         "bootstrap-seed",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer seed.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := seed.Connect(ctx, peer.Address()); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	waitForPeerCount(t, seed, 1)

	targetProfile := profile
	targetProfile.DefaultSeeds = []string{seed.Address()}
	target := mustStartNode(t, NodeConfig{
		NodeID:         "bootstrap-discovery-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &targetProfile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			OutboundTarget: 2,
		},
	})
	defer target.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	first := target.Bootstrap(ctx, nil)
	if first.Connected != 1 {
		t.Fatalf("seed connected=%d want=1 result=%+v", first.Connected, first)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, discovered := range target.DiscoveredPeers() {
			if discovered.NodeID == peer.NodeID() {
				second := target.MaintainOutbound(ctx)
				if second.Connected != 1 {
					t.Fatalf("discovered connected=%d want=1 result=%+v", second.Connected, second)
				}
				if target.PeerCount() != 2 {
					t.Fatalf("target peer count=%d want=2", target.PeerCount())
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("target did not discover non-seed peer")
}

func TestProtectionDefaultValues(t *testing.T) {
	cfg := normalizeProtectionConfig(ProtectionConfig{})
	if cfg.IdleTimeout != 5*time.Minute ||
		cfg.MaxInbound != 64 ||
		cfg.OutboundTarget != 8 ||
		cfg.MaxInboundPerIP != 4 ||
		cfg.BanDuration != time.Hour {
		t.Fatalf("unexpected protection defaults: %+v", cfg)
	}
}


func TestLegacyBootstrapAndMaintainDoesNothingWithoutSeeds(t *testing.T) {
	node := mustStartNode(t, NodeConfig{
		NodeID:        "legacy-no-seed-target",
		ListenAddress: "127.0.0.1:0",
	})
	defer node.Close()

	node.mu.Lock()
	node.discovered["legacy-would-connect"] = "127.0.0.1:65534"
	node.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result := node.BootstrapAndMaintain(ctx, nil)
	if result.Attempted != 0 || result.Connected != 0 || len(result.Failures) != 0 {
		t.Fatalf("legacy bootstrap changed frozen topology without seeds: %+v", result)
	}
	if node.PeerCount() != 0 {
		t.Fatalf("peer count=%d want=0", node.PeerCount())
	}
}

func TestBootstrapAndMaintainUsesDiscoveredPeersWithoutSeeds(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	peer := mustStartNode(t, NodeConfig{
		NodeID:         "cached-peer",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer peer.Close()

	node := mustStartNode(t, NodeConfig{
		NodeID:         "no-seed-target",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			OutboundTarget: 1,
		},
	})
	defer node.Close()

	node.mu.Lock()
	node.discovered[peer.NodeID()] = peer.Address()
	node.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := node.BootstrapAndMaintain(ctx, nil)
	if result.Connected != 1 {
		t.Fatalf("discovered connected=%d want=1 result=%+v", result.Connected, result)
	}
	if node.PeerCount() != 1 {
		t.Fatalf("peer count=%d want=1", node.PeerCount())
	}
}
