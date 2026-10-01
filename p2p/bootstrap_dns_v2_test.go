package p2p

import (
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestDNSSeedQueryIsThrottledBetweenRecoveryAttempts(t *testing.T) {
	now := time.Unix(2_100_000_000, 0).UTC()
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	profile.DNSSeeds = []string{"seed.example.org"}

	node, err := NewNode(NodeConfig{
		NodeID:         "dns-cadence",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			Now: func() time.Time { return now },
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !node.shouldQueryDNSSeeds(now) {
		t.Fatal("first cold-start DNS seed query was suppressed")
	}
	if node.shouldQueryDNSSeeds(now) {
		t.Fatal("immediate repeated DNS seed query was allowed")
	}

	now = now.Add(dnsSeedRetryInterval - time.Second)
	if node.shouldQueryDNSSeeds(now) {
		t.Fatal("DNS seed query was allowed before retry interval")
	}

	now = now.Add(time.Second)
	if !node.shouldQueryDNSSeeds(now) {
		t.Fatal("DNS seed query was not re-enabled for recovery")
	}
}

func TestDNSSeedQuerySuppressedWithHealthyOutboundSet(t *testing.T) {
	now := time.Unix(2_100_000_100, 0).UTC()
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	profile.DNSSeeds = []string{"seed.example.org"}

	node, err := NewNode(NodeConfig{
		NodeID:         "dns-healthy",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			OutboundTarget: 8,
			Now:            func() time.Time { return now },
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	node.mu.Lock()
	node.peers["peer-a"] = Peer{NodeID: "peer-a", Address: "127.0.0.1:10001", Inbound: false}
	node.peers["peer-b"] = Peer{NodeID: "peer-b", Address: "127.0.0.1:10002", Inbound: false}
	node.mu.Unlock()

	if node.shouldQueryDNSSeeds(now) {
		t.Fatal("healthy outbound set unexpectedly triggered DNS seeds")
	}
}

func TestDNSSeedQueryStillRunsWithOnlyOneOutboundPeer(t *testing.T) {
	now := time.Unix(2_100_000_200, 0).UTC()
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	profile.DNSSeeds = []string{"seed.example.org"}

	node, err := NewNode(NodeConfig{
		NodeID:         "dns-low-outbound",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			OutboundTarget: 8,
			Now:            func() time.Time { return now },
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	node.mu.Lock()
	node.peers["peer-a"] = Peer{NodeID: "peer-a", Address: "127.0.0.1:10001", Inbound: false}
	node.mu.Unlock()

	if !node.shouldQueryDNSSeeds(now) {
		t.Fatal("single outbound peer suppressed recovery DNS lookup")
	}
}

func TestDNSSeedQueryDisabledWhenProfileHasNoDNSSeeds(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	node, err := NewNode(NodeConfig{
		NodeID:         "dns-none",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if node.shouldQueryDNSSeeds(time.Now()) {
		t.Fatal("DNS query enabled without configured DNS seeds")
	}
}
