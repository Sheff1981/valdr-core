package p2p

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestDesktopOutboundOnlyV2ConnectsWithoutListener(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	server := mustStartNode(t, NodeConfig{
		NodeID:           "desktop-server",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "server.example:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	defer server.Close()

	desktop, err := NewNode(NodeConfig{
		NodeID:         "desktop-client",
		OutboundOnly:   true,
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := desktop.Start(); err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()

	if desktop.Address() != "" {
		t.Fatalf("outbound-only node unexpectedly has listen address %q", desktop.Address())
	}
	if desktop.AdvertiseAddress() != "" {
		t.Fatalf(
			"outbound-only node unexpectedly advertises %q",
			desktop.AdvertiseAddress(),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := desktop.Connect(ctx, server.Address()); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, desktop, 1)
	waitForPeerCount(t, server, 1)

	time.Sleep(200 * time.Millisecond)
	if desktop.PeerCount() != 1 || server.PeerCount() != 1 {
		t.Fatalf(
			"peer connection did not survive discovery exchange: desktop=%d server=%d",
			desktop.PeerCount(),
			server.PeerCount(),
		)
	}
}

func TestDesktopOutboundOnlyEndpointIsNotGossiped(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	server := mustStartNode(t, NodeConfig{
		NodeID:           "gossip-server",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "server.example:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	defer server.Close()

	desktop, err := NewNode(NodeConfig{
		NodeID:         "hidden-desktop",
		OutboundOnly:   true,
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := desktop.Start(); err != nil {
		t.Fatal(err)
	}
	defer desktop.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := desktop.Connect(ctx, server.Address()); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	waitForPeerCount(t, server, 1)

	observer := mustStartNode(t, NodeConfig{
		NodeID:           "gossip-observer",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "observer.example:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	defer observer.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := observer.Connect(ctx, server.Address()); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, observer, 1)
	time.Sleep(200 * time.Millisecond)

	for _, discovered := range observer.DiscoveredPeers() {
		if discovered.NodeID == "hidden-desktop" {
			t.Fatalf(
				"outbound-only desktop leaked through peer gossip: %+v",
				discovered,
			)
		}
	}
	if observer.PeerCount() != 1 {
		t.Fatalf("observer lost server peer: count=%d", observer.PeerCount())
	}
}

func TestDesktopOutboundOnlyRejectsAdvertisedAddress(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewNode(NodeConfig{
		NodeID:           "invalid-desktop",
		AdvertiseAddress: "desktop.example:17333",
		OutboundOnly:     true,
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error=%v want ErrInvalidConfig", err)
	}
}

func TestOutboundOnlyRequiresV2(t *testing.T) {
	_, err := NewNode(NodeConfig{
		NodeID:       "legacy-outbound-only",
		OutboundOnly: true,
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error=%v want ErrInvalidConfig", err)
	}
}
