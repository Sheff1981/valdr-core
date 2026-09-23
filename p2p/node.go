package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	valdrconfig "github.com/Sheff1981/valdr-core/config"
)

const defaultHandshakeTimeout = 5 * time.Second

var (
	ErrInvalidConfig = errors.New("invalid P2P node config")
	ErrAlreadyStarted = errors.New("P2P node already started")
	ErrNotStarted = errors.New("P2P node not started")
	ErrNodeClosed = errors.New("P2P node closed")
)

type HeightProvider func() uint64

type NodeConfig struct {
	NodeID           string
	ListenAddress    string
	ChainID          string
	ProtocolVersion  uint32
	HeightProvider   HeightProvider
	HandshakeTimeout time.Duration
}

type Peer struct {
	NodeID          string `json:"node_id"`
	Address         string `json:"address"`
	Height          uint64 `json:"height"`
	ProtocolVersion uint32 `json:"protocol_version"`
	Inbound         bool   `json:"inbound"`
}

type Node struct {
	nodeID           string
	listenAddress    string
	chainID          string
	protocolVersion  uint32
	heightProvider   HeightProvider
	handshakeTimeout time.Duration

	mu       sync.RWMutex
	listener net.Listener
	peers    map[string]Peer
	conns    map[string]net.Conn
	closed   bool
	wg       sync.WaitGroup
}

func NewNode(cfg NodeConfig) (*Node, error) {
	nodeID := strings.TrimSpace(cfg.NodeID)
	if nodeID == "" || nodeID != cfg.NodeID || len(nodeID) > maxNodeIDLength {
		return nil, fmt.Errorf("%w: node id", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.ListenAddress) == "" {
		return nil, fmt.Errorf("%w: listen address", ErrInvalidConfig)
	}

	chainID := cfg.ChainID
	if chainID == "" {
		chainID = valdrconfig.ChainID
	}
	protocolVersion := cfg.ProtocolVersion
	if protocolVersion == 0 {
		protocolVersion = valdrconfig.P2PProtocolVersion
	}
	heightProvider := cfg.HeightProvider
	if heightProvider == nil {
		heightProvider = func() uint64 { return 0 }
	}
	handshakeTimeout := cfg.HandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = defaultHandshakeTimeout
	}

	return &Node{
		nodeID:           nodeID,
		listenAddress:    cfg.ListenAddress,
		chainID:          chainID,
		protocolVersion:  protocolVersion,
		heightProvider:   heightProvider,
		handshakeTimeout: handshakeTimeout,
		peers:            make(map[string]Peer),
		conns:            make(map[string]net.Conn),
	}, nil
}

func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return ErrNodeClosed
	}
	if n.listener != nil {
		return ErrAlreadyStarted
	}

	listener, err := net.Listen("tcp", n.listenAddress)
	if err != nil {
		return err
	}
	n.listener = listener

	n.wg.Add(1)
	go n.acceptLoop(listener)
	return nil
}

func (n *Node) Address() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.listener != nil {
		return n.listener.Addr().String()
	}
	return n.listenAddress
}

func (n *Node) NodeID() string {
	return n.nodeID
}

func (n *Node) Connect(ctx context.Context, address string) error {
	n.mu.RLock()
	started := n.listener != nil
	closed := n.closed
	n.mu.RUnlock()

	if closed {
		return ErrNodeClosed
	}
	if !started {
		return ErrNotStarted
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	peer, err := n.exchangeHello(conn, false)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := n.registerPeer(peer, conn); err != nil {
		_ = conn.Close()
		return err
	}
	return nil
}

func (n *Node) Peers() []Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]Peer, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].NodeID < peers[j].NodeID
	})
	return peers
}

func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

func (n *Node) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true

	listener := n.listener
	n.listener = nil

	conns := make([]net.Conn, 0, len(n.conns))
	for _, conn := range n.conns {
		conns = append(conns, conn)
	}
	n.peers = make(map[string]Peer)
	n.conns = make(map[string]net.Conn)
	n.mu.Unlock()

	var firstErr error
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			firstErr = err
		}
	}
	for _, conn := range conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	n.wg.Wait()
	return firstErr
}

func (n *Node) acceptLoop(listener net.Listener) {
	defer n.wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			n.mu.RLock()
			closed := n.closed
			n.mu.RUnlock()
			if closed {
				return
			}
			continue
		}

		n.wg.Add(1)
		go n.handleInbound(conn)
	}
}

func (n *Node) handleInbound(conn net.Conn) {
	defer n.wg.Done()

	peer, err := n.exchangeHello(conn, true)
	if err != nil {
		_ = conn.Close()
		return
	}
	if err := n.registerPeer(peer, conn); err != nil {
		_ = conn.Close()
	}
}

func (n *Node) exchangeHello(conn net.Conn, inbound bool) (Peer, error) {
	if err := conn.SetDeadline(time.Now().Add(n.handshakeTimeout)); err != nil {
		return Peer{}, err
	}
	defer func() {
		_ = conn.SetDeadline(time.Time{})
	}()

	local := helloMessage{
		Type:            messageTypeHello,
		ProtocolVersion: n.protocolVersion,
		ChainID:         n.chainID,
		NodeID:          n.nodeID,
		ListenAddress:   n.Address(),
		Height:          n.heightProvider(),
	}
	if err := writeFrame(conn, local); err != nil {
		return Peer{}, err
	}

	var remote helloMessage
	if err := readFrame(conn, &remote); err != nil {
		return Peer{}, err
	}
	if err := n.validateHello(remote); err != nil {
		return Peer{}, err
	}

	return Peer{
		NodeID:          remote.NodeID,
		Address:         remote.ListenAddress,
		Height:          remote.Height,
		ProtocolVersion: remote.ProtocolVersion,
		Inbound:         inbound,
	}, nil
}

func (n *Node) validateHello(remote helloMessage) error {
	if remote.Type != messageTypeHello {
		return fmt.Errorf("%w: message type %q", ErrInvalidHello, remote.Type)
	}
	if remote.ChainID != n.chainID {
		return fmt.Errorf(
			"%w: got %q want %q",
			ErrWrongChainID,
			remote.ChainID,
			n.chainID,
		)
	}
	if remote.ProtocolVersion != n.protocolVersion {
		return fmt.Errorf(
			"%w: got %d want %d",
			ErrProtocolVersion,
			remote.ProtocolVersion,
			n.protocolVersion,
		)
	}

	nodeID := strings.TrimSpace(remote.NodeID)
	if nodeID == "" || nodeID != remote.NodeID || len(nodeID) > maxNodeIDLength {
		return fmt.Errorf("%w: node id", ErrInvalidHello)
	}
	if remote.NodeID == n.nodeID {
		return ErrSelfConnection
	}
	if strings.TrimSpace(remote.ListenAddress) == "" {
		return fmt.Errorf("%w: listen address", ErrInvalidHello)
	}
	if _, _, err := net.SplitHostPort(remote.ListenAddress); err != nil {
		return fmt.Errorf("%w: listen address: %v", ErrInvalidHello, err)
	}
	return nil
}

func (n *Node) registerPeer(peer Peer, conn net.Conn) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return ErrNodeClosed
	}
	if _, exists := n.peers[peer.NodeID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicatePeer, peer.NodeID)
	}
	n.peers[peer.NodeID] = peer
	n.conns[peer.NodeID] = conn
	return nil
}
