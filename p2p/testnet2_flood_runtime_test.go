package p2p

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestTestnet2OversizeFrameDisconnectsPeerBeforePayloadRead(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	node := mustStartNode(t, NodeConfig{
		NodeID:         "testnet2-oversize-node",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer node.Close()

	conn := dialAndHandshakeTestnet2Peer(
		t,
		node.Address(),
		profile,
		"oversize-peer",
	)
	defer conn.Close()

	header := make([]byte, v2FrameHeaderSize)
	magic := profile.Magic()
	copy(header[0:4], magic[:])
	binary.BigEndian.PutUint16(header[4:6], profile.ProtocolMax)
	binary.BigEndian.PutUint16(header[6:8], uint16(V2MessagePing))
	binary.BigEndian.PutUint32(
		header[8:12],
		uint32(v2MessagePayloadLimit(V2MessagePing)+1),
	)

	if _, err := conn.Write(header); err != nil {
		t.Fatal(err)
	}

	waitPeerDropped(t, node)

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("oversize peer connection remained readable after rejection")
	}
}

func TestTestnet2FloodRateLimitDisconnectsAndBansPeer(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	node := mustStartNode(t, NodeConfig{
		NodeID:         "testnet2-flood-node",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			MessagesPerSecond: 0.01,
			MessageBurst:      2,
			BytesPerSecond:    1024 * 1024,
			ByteBurst:         1024 * 1024,
			MalformedThreshold: 3,
			BanDuration:        time.Minute,
		},
	})
	defer node.Close()

	conn := dialAndHandshakeTestnet2Peer(
		t,
		node.Address(),
		profile,
		"flood-peer",
	)
	defer conn.Close()

	for i := uint64(1); i <= 3; i++ {
		if err := WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessagePing,
			struct {
				Nonce uint64 `json:"nonce"`
			}{Nonce: i},
		); err != nil {
			t.Fatal(err)
		}
	}

	waitPeerDropped(t, node)
	if !node.isIPBanned("127.0.0.1") {
		t.Fatal("rate-limited Testnet2 peer IP was not temporarily banned")
	}

	probe, err := net.DialTimeout("tcp", node.Address(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()

	_ = probe.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	var one [1]byte
	if _, err := probe.Read(one[:]); err == nil {
		t.Fatal("banned flood peer unexpectedly received handshake data")
	}
}

func waitPeerDropped(t *testing.T, node *Node) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if node.PeerCount() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("peer was not dropped; peers=%d", node.PeerCount())
}
