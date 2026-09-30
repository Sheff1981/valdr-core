package p2p

import (
	"context"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestListenNetwork(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{name: "ipv4 wildcard", address: "0.0.0.0:17333", want: "tcp4"},
		{name: "ipv4 loopback", address: "127.0.0.1:17333", want: "tcp4"},
		{name: "ipv6 wildcard", address: "[::]:17333", want: "tcp6"},
		{name: "ipv6 loopback", address: "[::1]:17333", want: "tcp6"},
		{name: "hostname", address: "node.example.org:17333", want: "tcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenNetwork(tt.address); got != tt.want {
				t.Fatalf("listenNetwork(%q)=%q want %q", tt.address, got, tt.want)
			}
		})
	}
}

func TestBootstrapSkipsOwnAdvertisedSeed(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.DefaultSeeds) == 0 {
		t.Fatal("testnet2 has no default seed")
	}

	node, err := NewNode(NodeConfig{
		NodeID:           "self-seed-test",
		ListenAddress:    "127.0.0.1:0",
		AdvertiseAddress: profile.DefaultSeeds[0],
		NetworkProfile:   &profile,
		EnableV2:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	result := node.Bootstrap(context.Background(), nil)
	if result.Attempted != 0 {
		t.Fatalf("self bootstrap attempted=%d want 0", result.Attempted)
	}
	if result.Connected != 0 {
		t.Fatalf("self bootstrap connected=%d want 0", result.Connected)
	}
}
