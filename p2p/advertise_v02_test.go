package p2p

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestV2ExplicitAdvertiseAddressIsUsedInHandshake(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:           "advertise-a",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "node-a.example:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:           "advertise-b",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "node-b.example:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := nodeB.Connect(ctx, nodeA.Address()); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, nodeA, 1)
	waitForPeerCount(t, nodeB, 1)

	peersA := nodeA.Peers()
	peersB := nodeB.Peers()
	if len(peersA) != 1 || peersA[0].Address != "node-b.example:17333" {
		t.Fatalf("node A peer advertisement=%+v", peersA)
	}
	if len(peersB) != 1 || peersB[0].Address != "node-a.example:17333" {
		t.Fatalf("node B peer advertisement=%+v", peersB)
	}
}

func TestPublicV2RejectsUnsafeExplicitAdvertiseAddress(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewNode(NodeConfig{
		NodeID:           "unsafe-advertise",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: "127.0.0.1:17333",
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error=%v want ErrInvalidConfig", err)
	}
}
