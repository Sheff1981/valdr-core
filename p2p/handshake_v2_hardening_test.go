package p2p

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

func TestV2HandshakeRejectsUnexpectedFirstMessageWithoutRegisteringPeer(t *testing.T) {
	profile := mustTestnet2Profile(t)

	addr, done := startV2HandshakeServer(t, profile, func(conn net.Conn) error {
		if _, err := ReadV2Frame(conn, profile); err != nil {
			return err
		}
		return WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessagePing,
			struct {
				Nonce uint64 `json:"nonce"`
			}{Nonce: 1},
		)
	})

	node := mustStartNode(t, NodeConfig{
		NodeID:         "handshake-client-first-message",
		OutboundOnly:   true,
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := node.Connect(ctx, addr)
	if !errors.Is(err, ErrInvalidHello) {
		t.Fatalf("Connect error=%v want ErrInvalidHello", err)
	}
	if node.PeerCount() != 0 {
		t.Fatalf("peer count=%d after rejected handshake", node.PeerCount())
	}
	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestV2HandshakeRejectsDuplicateHelloInsteadOfAck(t *testing.T) {
	profile := mustTestnet2Profile(t)

	addr, done := startV2HandshakeServer(t, profile, func(conn net.Conn) error {
		if _, err := ReadV2Frame(conn, profile); err != nil {
			return err
		}
		remote := validTestnet2Hello(profile, "handshake-server-duplicate-hello")
		if err := WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessageHello,
			remote,
		); err != nil {
			return err
		}
		ack, err := ReadV2Frame(conn, profile)
		if err != nil {
			return err
		}
		if ack.MessageType != V2MessageHelloAck {
			return errors.New("client did not send hello_ack")
		}
		return WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessageHello,
			remote,
		)
	})

	node := mustStartNode(t, NodeConfig{
		NodeID:         "handshake-client-duplicate-hello",
		OutboundOnly:   true,
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := node.Connect(ctx, addr)
	if !errors.Is(err, ErrInvalidHello) {
		t.Fatalf("Connect error=%v want ErrInvalidHello", err)
	}
	if node.PeerCount() != 0 {
		t.Fatalf("peer count=%d after rejected duplicate hello", node.PeerCount())
	}
	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestV2HandshakeRejectsMismatchedHelloAckWithoutRegisteringPeer(t *testing.T) {
	profile := mustTestnet2Profile(t)

	addr, done := startV2HandshakeServer(t, profile, func(conn net.Conn) error {
		if _, err := ReadV2Frame(conn, profile); err != nil {
			return err
		}
		if err := WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessageHello,
			validTestnet2Hello(profile, "handshake-server-bad-ack"),
		); err != nil {
			return err
		}
		ack, err := ReadV2Frame(conn, profile)
		if err != nil {
			return err
		}
		if ack.MessageType != V2MessageHelloAck {
			return errors.New("client did not send hello_ack")
		}
		return WriteV2Frame(
			conn,
			profile,
			profile.ProtocolMax,
			V2MessageHelloAck,
			V2HelloAck{ProtocolVersion: profile.ProtocolMin - 1},
		)
	})

	node := mustStartNode(t, NodeConfig{
		NodeID:         "handshake-client-bad-ack",
		OutboundOnly:   true,
		NetworkProfile: &profile,
		EnableV2:       true,
	})
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := node.Connect(ctx, addr)
	if !errors.Is(err, ErrV2UnsupportedVersion) {
		t.Fatalf("Connect error=%v want ErrV2UnsupportedVersion", err)
	}
	if node.PeerCount() != 0 {
		t.Fatalf("peer count=%d after rejected hello_ack", node.PeerCount())
	}
	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func mustTestnet2Profile(t *testing.T) config.NetworkProfile {
	t.Helper()
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func validTestnet2Hello(profile config.NetworkProfile, nodeID string) V2Hello {
	return V2Hello{
		ChainID:             profile.ChainID,
		ProtocolMin:         profile.ProtocolMin,
		ProtocolMax:         profile.ProtocolMax,
		NodeID:              nodeID,
		Services:            []string{"network"},
		ListenAddress:       "127.0.0.1:17333",
		Height:              0,
		TipHash:             "",
		CumulativeChainwork: "0",
		UserAgent:           "/VALDR:test/",
		Timestamp:           time.Now().UTC().Unix(),
		Nonce:               1,
	}
}

func startV2HandshakeServer(
	t *testing.T,
	profile config.NetworkProfile,
	handler func(net.Conn) error,
) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		done <- handler(conn)
	}()

	return listener.Addr().String(), done
}
