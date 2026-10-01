package p2p

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestListenNetwork(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{name: "dual-stack wildcard", address: ":17333", want: "tcp"},
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

func TestDualStackWildcardAcceptsIPv4AndIPv6WhenAvailable(t *testing.T) {
	ipv6Probe, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable on this host: %v", err)
	}
	_ = ipv6Probe.Close()

	listener, err := net.Listen(listenNetwork(":0"), ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	accepted := make(chan error, 2)
	go func() {
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				accepted <- err
				return
			}
			_ = conn.Close()
			accepted <- nil
		}
	}()

	for _, tc := range []struct {
		network string
		host    string
	}{
		{network: "tcp4", host: "127.0.0.1"},
		{network: "tcp6", host: "::1"},
	} {
		conn, err := net.DialTimeout(
			tc.network,
			net.JoinHostPort(tc.host, port),
			time.Second,
		)
		if err != nil {
			t.Fatalf("dual-stack wildcard did not accept %s: %v", tc.network, err)
		}
		_ = conn.Close()
	}

	for i := 0; i < 2; i++ {
		select {
		case err := <-accepted:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for dual-stack accept")
		}
	}
}

func TestBootstrapSkipsOwnAdvertisedSeed(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	profile.DefaultSeeds = []string{"8.8.8.8:17333"}

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

func TestWildcardListenerIsNotAdvertisedBeforeMapping(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	node, err := NewNode(NodeConfig{
		NodeID:         "wildcard-listener-test",
		ListenAddress:  "0.0.0.0:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	if got := node.AdvertiseAddress(); got != "" {
		t.Fatalf("wildcard listener advertised as %q", got)
	}
	if got := node.helloAddress(); got != outboundOnlyHelloAddress {
		t.Fatalf("hello address=%q want=%q", got, outboundOnlyHelloAddress)
	}
	if err := node.SetAdvertiseAddress("8.8.8.8:17333"); err != nil {
		t.Fatal(err)
	}
	if got := node.AdvertiseAddress(); got != "8.8.8.8:17333" {
		t.Fatalf("mapped advertise address=%q", got)
	}
	node.ClearAdvertiseAddress()
	if got := node.AdvertiseAddress(); got != "" {
		t.Fatalf("cleared advertise address=%q", got)
	}
	if got := node.helloAddress(); got != outboundOnlyHelloAddress {
		t.Fatalf("cleared hello address=%q want=%q", got, outboundOnlyHelloAddress)
	}
}
