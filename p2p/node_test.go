package p2p

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestNodeAConnectsToNodeBAndExchangesHeight(t *testing.T) {
	nodeA, err := NewNode(NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
		HeightProvider: func() uint64 {
			return 7
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer nodeA.Close()

	nodeB, err := NewNode(NodeConfig{
		NodeID:        "node-b",
		ListenAddress: "127.0.0.1:0",
		HeightProvider: func() uint64 {
			return 3
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer nodeB.Close()

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := nodeA.Connect(ctx, nodeB.Address()); err != nil {
		t.Fatalf("Node A -> Node B connect: %v", err)
	}

	waitForPeerCount(t, nodeA, 1)
	waitForPeerCount(t, nodeB, 1)

	aPeers := nodeA.Peers()
	if got := aPeers[0]; got.NodeID != "node-b" ||
		got.Height != 3 ||
		got.Inbound ||
		got.ProtocolVersion != config.P2PProtocolVersion {
		t.Fatalf("Node A peer = %+v", got)
	}

	bPeers := nodeB.Peers()
	if got := bPeers[0]; got.NodeID != "node-a" ||
		got.Height != 7 ||
		!got.Inbound ||
		got.ProtocolVersion != config.P2PProtocolVersion {
		t.Fatalf("Node B peer = %+v", got)
	}

	if aPeers[0].Address != nodeB.Address() {
		t.Fatalf("Node A recorded address = %q, want %q", aPeers[0].Address, nodeB.Address())
	}
	if bPeers[0].Address != nodeA.Address() {
		t.Fatalf("Node B recorded address = %q, want %q", bPeers[0].Address, nodeA.Address())
	}
}

func TestRejectsWrongChainID(t *testing.T) {
	nodeA := mustStartNode(t, NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:        "node-b",
		ListenAddress: "127.0.0.1:0",
		ChainID:       "not-valdr-devnet",
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := nodeA.Connect(ctx, nodeB.Address())
	if !errors.Is(err, ErrWrongChainID) {
		t.Fatalf("Connect error = %v, want ErrWrongChainID", err)
	}
	if nodeA.PeerCount() != 0 {
		t.Fatalf("Node A peer count = %d after rejected handshake", nodeA.PeerCount())
	}
}

func TestRejectsProtocolVersionMismatch(t *testing.T) {
	nodeA := mustStartNode(t, NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:          "node-b",
		ListenAddress:   "127.0.0.1:0",
		ProtocolVersion: config.P2PProtocolVersion + 1,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := nodeA.Connect(ctx, nodeB.Address())
	if !errors.Is(err, ErrProtocolVersion) {
		t.Fatalf("Connect error = %v, want ErrProtocolVersion", err)
	}
}

func TestRejectsSelfConnection(t *testing.T) {
	node := mustStartNode(t, NodeConfig{
		NodeID:        "node-a",
		ListenAddress: "127.0.0.1:0",
	})
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := node.Connect(ctx, node.Address())
	if !errors.Is(err, ErrSelfConnection) {
		t.Fatalf("Connect error = %v, want ErrSelfConnection", err)
	}
}

func mustStartNode(t *testing.T, cfg NodeConfig) *Node {
	t.Helper()
	node, err := NewNode(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Start(); err != nil {
		t.Fatal(err)
	}
	return node
}

func waitForPeerCount(t *testing.T, node *Node, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if node.PeerCount() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("peer count = %d, want %d", node.PeerCount(), want)
}

func TestV2NodesHandshakeAndExchangeDiscovery(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}

	nodeC := mustStartNode(t, NodeConfig{
		NodeID:         "v2-node-c",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer nodeC.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:         "v2-node-b",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer nodeB.Close()

	ctxBC, cancelBC := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelBC()
	if err := nodeB.Connect(ctxBC, nodeC.Address()); err != nil {
		t.Fatalf("v2 Node B -> C connect: %v", err)
	}
	waitForPeerCount(t, nodeB, 1)
	waitForPeerCount(t, nodeC, 1)

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:         "v2-node-a",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer nodeA.Close()

	ctxAB, cancelAB := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelAB()
	if err := nodeA.Connect(ctxAB, nodeB.Address()); err != nil {
		t.Fatalf("v2 Node A -> B connect: %v", err)
	}
	waitForPeerCount(t, nodeA, 1)

	peer := nodeA.Peers()[0]
	if peer.ProtocolVersion != 2 || peer.NodeID != "v2-node-b" {
		t.Fatalf("unexpected v2 peer: %+v", peer)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, discovered := range nodeA.DiscoveredPeers() {
			if discovered.NodeID == "v2-node-c" && discovered.Address == nodeC.Address() {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("v2 node A did not discover node C through node B")
}

func TestV2RejectsWrongNetworkMagic(t *testing.T) {
	devnet, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	testnet, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}

	nodeA := mustStartNode(t, NodeConfig{
		NodeID:         "v2-devnet",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &devnet,
		EnableV2:       true,
	})
	defer nodeA.Close()

	nodeB := mustStartNode(t, NodeConfig{
		NodeID:         "v2-testnet",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &testnet,
		EnableV2:       true,
	})
	defer nodeB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = nodeA.Connect(ctx, nodeB.Address())
	if !errors.Is(err, ErrV2WrongNetwork) {
		t.Fatalf("Connect error=%v want ErrV2WrongNetwork", err)
	}
	if nodeA.PeerCount() != 0 {
		t.Fatalf("devnet peer count=%d after wrong-network reject", nodeA.PeerCount())
	}
}

func TestV2RequiresExplicitNetworkProfile(t *testing.T) {
	_, err := NewNode(NodeConfig{
		NodeID:        "v2-no-profile",
		ListenAddress: "127.0.0.1:0",
		EnableV2:      true,
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewNode error=%v want ErrInvalidConfig", err)
	}
}
