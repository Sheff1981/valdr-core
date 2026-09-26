package p2p

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestTestnet2MalformedPeerIsBannedAndCanReconnectAfterExpiry(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}

	var nowUnix atomic.Int64
	nowUnix.Store(time.Unix(2_000_100_000, 0).Unix())

	node := mustStartNode(t, NodeConfig{
		NodeID:         "testnet2-protected-node",
		ListenAddress:  "127.0.0.1:0",
		NetworkProfile: &profile,
		EnableV2:       true,
		Protection: ProtectionConfig{
			MalformedThreshold: 2,
			BanDuration:        time.Minute,
			Now: func() time.Time {
				return time.Unix(nowUnix.Load(), 0)
			},
		},
	})
	defer node.Close()

	for attempt := 0; attempt < 2; attempt++ {
		conn := dialAndHandshakeTestnet2Peer(
			t,
			node.Address(),
			profile,
			"malformed-peer-"+string(rune('a'+attempt)),
		)

		raw := malformedChecksumFrame(t, profile)
		if _, err := conn.Write(raw); err != nil {
			_ = conn.Close()
			t.Fatal(err)
		}
		_ = conn.Close()

		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			if node.PeerCount() == 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if node.PeerCount() != 0 {
			t.Fatalf("attempt %d malicious peer was not disconnected", attempt+1)
		}
	}

	if !node.isIPBanned("127.0.0.1") {
		t.Fatal("malformed Testnet2 peer IP was not banned at threshold")
	}

	conn, err := net.DialTimeout("tcp", node.Address(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("banned peer unexpectedly received handshake data")
	}

	nowUnix.Add(int64(time.Minute/time.Second) + 1)
	if node.isIPBanned("127.0.0.1") {
		t.Fatal("temporary Testnet2 peer ban did not expire")
	}

	recovered := dialAndHandshakeTestnet2Peer(
		t,
		node.Address(),
		profile,
		"recovered-peer",
	)
	defer recovered.Close()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if node.PeerCount() == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("valid Testnet2 peer did not reconnect after ban expiry; peers=%d", node.PeerCount())
}

func dialAndHandshakeTestnet2Peer(
	t *testing.T,
	address string,
	profile config.NetworkProfile,
	nodeID string,
) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetDeadline(time.Now().Add(time.Second))

	serverHelloFrame, err := ReadV2Frame(conn, profile)
	if err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if serverHelloFrame.MessageType != V2MessageHello {
		_ = conn.Close()
		t.Fatalf("server message type=%d want hello", serverHelloFrame.MessageType)
	}

	var serverHello V2Hello
	if err := DecodeV2Payload(serverHelloFrame, &serverHello); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}

	clientHello := V2Hello{
		ChainID:             profile.ChainID,
		ProtocolMin:         profile.ProtocolMin,
		ProtocolMax:         profile.ProtocolMax,
		NodeID:              nodeID,
		Services:            []string{"network"},
		ListenAddress:       "127.0.0.1:1",
		Height:              0,
		TipHash:             profile.GenesisHash,
		CumulativeChainwork: "0",
		UserAgent:           "/VALDR:testnet2-protection-test/",
		Timestamp:           time.Now().UTC().Unix(),
		Nonce:               1,
	}
	if err := WriteV2Frame(
		conn,
		profile,
		profile.ProtocolMax,
		V2MessageHello,
		clientHello,
	); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}

	ackFrame, err := ReadV2Frame(conn, profile)
	if err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if ackFrame.MessageType != V2MessageHelloAck {
		_ = conn.Close()
		t.Fatalf("server message type=%d want hello_ack", ackFrame.MessageType)
	}

	var ack V2HelloAck
	if err := DecodeV2Payload(ackFrame, &ack); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if ack.ProtocolVersion != profile.ProtocolMax {
		_ = conn.Close()
		t.Fatalf("ack protocol=%d want=%d", ack.ProtocolVersion, profile.ProtocolMax)
	}
	if err := WriteV2Frame(
		conn,
		profile,
		ack.ProtocolVersion,
		V2MessageHelloAck,
		V2HelloAck{ProtocolVersion: ack.ProtocolVersion},
	); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}

	_ = conn.SetDeadline(time.Time{})
	return conn
}

func malformedChecksumFrame(
	t *testing.T,
	profile config.NetworkProfile,
) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := WriteV2Frame(
		&buf,
		profile,
		profile.ProtocolMax,
		V2MessagePing,
		struct {
			Nonce uint64 `json:"nonce"`
		}{Nonce: 99},
	); err != nil {
		t.Fatal(err)
	}

	raw := append([]byte(nil), buf.Bytes()...)
	if len(raw) <= v2FrameHeaderSize {
		t.Fatal("unexpected short frame")
	}
	raw[v2FrameHeaderSize] ^= 0x01

	if _, err := ReadV2Frame(bytes.NewReader(raw), profile); !errors.Is(err, ErrV2Checksum) {
		t.Fatalf("fixture error=%v want ErrV2Checksum", err)
	}

	size := binary.BigEndian.Uint32(raw[8:12])
	if int(size) != len(raw)-v2FrameHeaderSize {
		t.Fatal("malformed frame fixture changed payload length")
	}
	return raw
}
