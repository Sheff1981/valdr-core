package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	valdrconfig "github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
)

const defaultHandshakeTimeout = 5 * time.Second

var (
	ErrInvalidConfig        = errors.New("invalid P2P node config")
	ErrAlreadyStarted       = errors.New("P2P node already started")
	ErrNotStarted           = errors.New("P2P node not started")
	ErrNodeClosed           = errors.New("P2P node closed")
	ErrDataLayerUnavailable = errors.New("P2P data layer unavailable")
	ErrUnknownBlock         = errors.New("block is not in local blockchain")
	ErrForkUnsupported      = errors.New("fork or reorganization is not supported in v0.1 basic sync")
	ErrInvalidPeerAddress   = errors.New("invalid discovered peer address")
)

type HeightProvider func() uint64

type NodeConfig struct {
	NodeID           string
	ListenAddress    string
	ChainID          string
	ProtocolVersion  uint32
	HeightProvider   HeightProvider
	Blockchain       *blockchain.Blockchain
	Mempool          *mempool.Pool
	HandshakeTimeout time.Duration
}

type Peer struct {
	NodeID          string `json:"node_id"`
	Address         string `json:"address"`
	Height          uint64 `json:"height"`
	ProtocolVersion uint32 `json:"protocol_version"`
	Inbound         bool   `json:"inbound"`
}

type DiscoveredPeer struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

type peerConnection struct {
	conn    net.Conn
	writeMu sync.Mutex
}

type Node struct {
	nodeID           string
	listenAddress    string
	chainID          string
	protocolVersion  uint32
	heightProvider   HeightProvider
	blockchain       *blockchain.Blockchain
	mempool          *mempool.Pool
	handshakeTimeout time.Duration

	mu         sync.RWMutex
	listener   net.Listener
	peers      map[string]Peer
	conns      map[string]*peerConnection
	discovered map[string]string
	closed     bool
	wg         sync.WaitGroup
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
	if heightProvider == nil && cfg.Blockchain != nil {
		heightProvider = cfg.Blockchain.Height
	}
	if heightProvider == nil {
		heightProvider = func() uint64 { return 0 }
	}

	pool := cfg.Mempool
	if pool == nil {
		pool = mempool.New()
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
		blockchain:       cfg.Blockchain,
		mempool:          pool,
		handshakeTimeout: handshakeTimeout,
		peers:            make(map[string]Peer),
		conns:            make(map[string]*peerConnection),
		discovered:       make(map[string]string),
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
	pc, err := n.registerPeer(peer, conn)
	if err != nil {
		_ = conn.Close()
		return err
	}

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		n.servePeer(peer.NodeID, pc)
	}()

	n.afterPeerConnected(peer.NodeID, peer.Height)
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

func (n *Node) DiscoveredPeers() []DiscoveredPeer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]DiscoveredPeer, 0, len(n.discovered))
	for nodeID, address := range n.discovered {
		if nodeID == n.nodeID {
			continue
		}
		if _, connected := n.peers[nodeID]; connected {
			continue
		}
		peers = append(peers, DiscoveredPeer{
			NodeID:  nodeID,
			Address: address,
		})
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

func (n *Node) MempoolLen() int {
	return n.mempool.Len()
}

func (n *Node) MempoolTransactions() []*transaction.Transaction {
	return n.mempool.Transactions()
}

func (n *Node) BroadcastTransaction(tx *transaction.Transaction) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := n.blockchain.ValidateTransaction(tx); err != nil {
		return err
	}
	if err := n.mempool.Add(tx); err != nil {
		return err
	}
	return n.broadcastExcept("", transactionMessage{
		Type:        messageTypeTransaction,
		Transaction: tx,
	})
}

func (n *Node) BroadcastBlock(candidate *block.Block) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if candidate == nil {
		return ErrUnknownBlock
	}

	local, ok := n.blockchain.BlockAt(candidate.Height)
	if !ok || local.BlockHash != candidate.BlockHash {
		return ErrUnknownBlock
	}

	n.mempool.RemoveBlockTransactions(candidate.Transactions)
	n.pruneMempool()
	return n.broadcastExcept("", blockMessage{
		Type:  messageTypeBlock,
		Block: candidate,
	})
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
	for _, pc := range n.conns {
		conns = append(conns, pc.conn)
	}
	n.peers = make(map[string]Peer)
	n.conns = make(map[string]*peerConnection)
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
	pc, err := n.registerPeer(peer, conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	n.afterPeerConnected(peer.NodeID, peer.Height)
	n.servePeer(peer.NodeID, pc)
}

func (n *Node) servePeer(peerID string, pc *peerConnection) {
	defer n.dropPeer(peerID, pc)

	for {
		payload, err := readFramePayload(pc.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			return
		}
		if err := n.handlePayload(peerID, payload); err != nil {
			return
		}
	}
}

func (n *Node) handlePayload(peerID string, payload []byte) error {
	header, err := decodeMessageHeader(payload)
	if err != nil {
		return err
	}

	switch header.Type {
	case messageTypeBlock:
		var message blockMessage
		if err := decodePayload(payload, &message); err != nil {
			return err
		}
		return n.handleBlock(peerID, message.Block)

	case messageTypeTransaction:
		var message transactionMessage
		if err := decodePayload(payload, &message); err != nil {
			return err
		}
		return n.handleTransaction(peerID, message.Transaction)

	case messageTypeGetBlock:
		var message getBlockMessage
		if err := decodePayload(payload, &message); err != nil {
			return err
		}
		return n.handleGetBlock(peerID, message.Height)

	case messageTypeGetPeers:
		var message getPeersMessage
		if err := decodePayload(payload, &message); err != nil {
			return err
		}
		return n.handleGetPeers(peerID)

	case messageTypePeers:
		var message peersMessage
		if err := decodePayload(payload, &message); err != nil {
			return err
		}
		return n.handlePeers(message.Peers)

	default:
		return fmt.Errorf("%w: %q", ErrUnknownMessage, header.Type)
	}
}

func (n *Node) handleTransaction(peerID string, tx *transaction.Transaction) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := n.blockchain.ValidateTransaction(tx); err != nil {
		return err
	}
	if err := n.mempool.Add(tx); err != nil {
		if errors.Is(err, mempool.ErrDuplicateTransaction) {
			return nil
		}
		return err
	}

	return n.broadcastExcept(peerID, transactionMessage{
		Type:        messageTypeTransaction,
		Transaction: tx,
	})
}

func (n *Node) handleBlock(peerID string, candidate *block.Block) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if candidate == nil {
		return ErrInvalidFrame
	}

	n.updatePeerHeight(peerID, candidate.Height)
	current := n.blockchain.Height()

	if candidate.Height <= current {
		local, ok := n.blockchain.BlockAt(candidate.Height)
		if ok && local.BlockHash == candidate.BlockHash {
			return nil
		}
		return ErrForkUnsupported
	}

	if candidate.Height > current+1 {
		return n.requestBlock(peerID, current+1)
	}

	if err := n.blockchain.AddBlock(candidate); err != nil {
		return err
	}
	n.mempool.RemoveBlockTransactions(candidate.Transactions)
	n.pruneMempool()

	if err := n.broadcastExcept(peerID, blockMessage{
		Type:  messageTypeBlock,
		Block: candidate,
	}); err != nil {
		return err
	}

	if n.peerHeight(peerID) > n.blockchain.Height() {
		return n.requestBlock(peerID, n.blockchain.Height()+1)
	}
	return nil
}

func (n *Node) handleGetBlock(peerID string, height uint64) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	candidate, ok := n.blockchain.BlockAt(height)
	if !ok {
		return nil
	}
	return n.sendTo(peerID, blockMessage{
		Type:  messageTypeBlock,
		Block: candidate,
	})
}

func (n *Node) handleGetPeers(peerID string) error {
	n.mu.RLock()
	advertisements := make([]peerAdvertisement, 0, len(n.peers)+1)
	advertisements = append(advertisements, peerAdvertisement{
		NodeID:  n.nodeID,
		Address: n.listenerAddressLocked(),
	})
	for _, peer := range n.peers {
		advertisements = append(advertisements, peerAdvertisement{
			NodeID:  peer.NodeID,
			Address: peer.Address,
		})
	}
	n.mu.RUnlock()

	return n.sendTo(peerID, peersMessage{
		Type:  messageTypePeers,
		Peers: advertisements,
	})
}

func (n *Node) handlePeers(peers []peerAdvertisement) error {
	for _, peer := range peers {
		if peer.NodeID == n.nodeID {
			continue
		}

		nodeID := strings.TrimSpace(peer.NodeID)
		if nodeID == "" || nodeID != peer.NodeID || len(nodeID) > maxNodeIDLength {
			return ErrInvalidPeerAddress
		}
		if strings.TrimSpace(peer.Address) == "" {
			return ErrInvalidPeerAddress
		}
		if _, _, err := net.SplitHostPort(peer.Address); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidPeerAddress, err)
		}

		n.mu.Lock()
		if _, connected := n.peers[peer.NodeID]; !connected {
			n.discovered[peer.NodeID] = peer.Address
		}
		n.mu.Unlock()
	}
	return nil
}

func (n *Node) afterPeerConnected(peerID string, peerHeight uint64) {
	_ = n.sendTo(peerID, getPeersMessage{Type: messageTypeGetPeers})

	if n.blockchain != nil && peerHeight > n.blockchain.Height() {
		_ = n.requestBlock(peerID, n.blockchain.Height()+1)
	}
}

func (n *Node) pruneMempool() {
	if n.blockchain == nil {
		return
	}
	for _, tx := range n.mempool.Transactions() {
		if err := n.blockchain.ValidateTransaction(tx); err != nil {
			n.mempool.Remove(tx.TransactionID)
		}
	}
}

func (n *Node) requestBlock(peerID string, height uint64) error {
	if height == 0 {
		height = 1
	}
	return n.sendTo(peerID, getBlockMessage{
		Type:   messageTypeGetBlock,
		Height: height,
	})
}

func (n *Node) broadcastExcept(excludedPeerID string, value any) error {
	n.mu.RLock()
	peerIDs := make([]string, 0, len(n.conns))
	for peerID := range n.conns {
		if peerID != excludedPeerID {
			peerIDs = append(peerIDs, peerID)
		}
	}
	n.mu.RUnlock()
	sort.Strings(peerIDs)

	var firstErr error
	for _, peerID := range peerIDs {
		if err := n.sendTo(peerID, value); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *Node) sendTo(peerID string, value any) error {
	n.mu.RLock()
	pc, exists := n.conns[peerID]
	n.mu.RUnlock()
	if !exists {
		return fmt.Errorf("peer %q not connected", peerID)
	}

	pc.writeMu.Lock()
	defer pc.writeMu.Unlock()
	return writeFrame(pc.conn, value)
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

func (n *Node) registerPeer(peer Peer, conn net.Conn) (*peerConnection, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil, ErrNodeClosed
	}
	if _, exists := n.peers[peer.NodeID]; exists {
		return nil, fmt.Errorf("%w: %s", ErrDuplicatePeer, peer.NodeID)
	}

	pc := &peerConnection{conn: conn}
	n.peers[peer.NodeID] = peer
	n.conns[peer.NodeID] = pc
	delete(n.discovered, peer.NodeID)
	return pc, nil
}

func (n *Node) dropPeer(peerID string, expected *peerConnection) {
	n.mu.Lock()
	current, exists := n.conns[peerID]
	if exists && current == expected {
		delete(n.conns, peerID)
		delete(n.peers, peerID)
	}
	n.mu.Unlock()

	if exists && current == expected {
		_ = current.conn.Close()
	}
}

func (n *Node) updatePeerHeight(peerID string, height uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	peer, exists := n.peers[peerID]
	if !exists || height <= peer.Height {
		return
	}
	peer.Height = height
	n.peers[peerID] = peer
}

func (n *Node) peerHeight(peerID string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.peers[peerID].Height
}

func (n *Node) listenerAddressLocked() string {
	if n.listener != nil {
		return n.listener.Addr().String()
	}
	return n.listenAddress
}
