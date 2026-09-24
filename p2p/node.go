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
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/logging"
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
	NetworkProfile   *valdrconfig.NetworkProfile
	EnableV2         bool
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
	networkProfile   valdrconfig.NetworkProfile
	enableV2         bool
	heightProvider   HeightProvider
	blockchain       *blockchain.Blockchain
	mempool          *mempool.Pool
	handshakeTimeout time.Duration

	mu         sync.RWMutex
	listener   net.Listener
	peers      map[string]Peer
	conns      map[string]*peerConnection
	discovered map[string]string
	syncV2     map[string]*v2SyncState
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

	var networkProfile valdrconfig.NetworkProfile
	if cfg.EnableV2 {
		if cfg.NetworkProfile == nil {
			return nil, fmt.Errorf("%w: v2 network profile is required", ErrInvalidConfig)
		}
		networkProfile = *cfg.NetworkProfile
		chainID = networkProfile.ChainID
		protocolVersion = uint32(networkProfile.ProtocolMax)
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
		poolConfig := mempool.Config{}
		if cfg.EnableV2 && cfg.NetworkProfile != nil {
			poolConfig.MinRelayFeePerByte = cfg.NetworkProfile.MinRelayFeePerByte
		}
		pool = mempool.NewWithConfig(poolConfig)
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
		networkProfile:   networkProfile,
		enableV2:         cfg.EnableV2,
		heightProvider:   heightProvider,
		blockchain:       cfg.Blockchain,
		mempool:          pool,
		handshakeTimeout: handshakeTimeout,
		peers:            make(map[string]Peer),
		conns:            make(map[string]*peerConnection),
		discovered:       make(map[string]string),
		syncV2:           make(map[string]*v2SyncState),
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

func (n *Node) MempoolTransactionsForMining() []*transaction.Transaction {
	return n.mempool.MiningTransactions()
}

func (n *Node) BroadcastTransaction(tx *transaction.Transaction) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	fees, err := n.blockchain.CalculateFees([]*transaction.Transaction{tx})
	if err != nil {
		return err
	}
	if err := n.mempool.AddWithFee(tx, fees); err != nil {
		return err
	}
	logging.Printf(logging.CategoryTX, "accepted local txid=%s", tx.TransactionID)
	logging.Printf(logging.CategoryMempool, "size=%d", n.mempool.Len())
	if n.enableV2 {
		return n.broadcastV2Except("", V2MessageTx, V2TxPayload{
			Transaction: tx,
		})
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
	n.revalidateMempool()
	logging.Printf(
		logging.CategoryBlock,
		"broadcast height=%d hash=%s",
		candidate.Height,
		candidate.BlockHash,
	)
	if n.enableV2 {
		return n.broadcastV2Except("", V2MessageInv, V2InvPayload{
			Items: []V2InventoryItem{{
				Kind: V2InventoryBlock,
				Hash: candidate.BlockHash,
			}},
		})
	}
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
	n.syncV2 = make(map[string]*v2SyncState)
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
		if n.enableV2 {
			frame, err := ReadV2Frame(pc.conn, n.networkProfile)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return
				}
				return
			}
			if err := n.handleV2Frame(peerID, frame); err != nil {
				return
			}
			continue
		}

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
	fees, err := n.blockchain.CalculateFees([]*transaction.Transaction{tx})
	if err != nil {
		return err
	}
	if err := n.mempool.AddWithFee(tx, fees); err != nil {
		if errors.Is(err, mempool.ErrDuplicateTransaction) {
			return nil
		}
		return err
	}
	logging.Printf(logging.CategoryTX, "accepted txid=%s peer=%s", tx.TransactionID, peerID)
	logging.Printf(logging.CategoryMempool, "size=%d", n.mempool.Len())

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
		logging.Printf(
			logging.CategorySync,
			"missing height=%d peer=%s remote_height=%d",
			current+1,
			peerID,
			candidate.Height,
		)
		return n.requestBlock(peerID, current+1)
	}

	update, err := n.blockchain.AddBlockWithUpdate(candidate)
	if err != nil {
		return err
	}
	logging.Printf(
		logging.CategoryBlock,
		"accepted height=%d hash=%s peer=%s",
		candidate.Height,
		candidate.BlockHash,
		peerID,
	)
	n.applyMempoolChainUpdate(update)

	if err := n.broadcastExcept(peerID, blockMessage{
		Type:  messageTypeBlock,
		Block: candidate,
	}); err != nil {
		return err
	}

	if n.peerHeight(peerID) > n.blockchain.Height() {
		next := n.blockchain.Height() + 1
		logging.Printf(logging.CategorySync, "request height=%d peer=%s", next, peerID)
		return n.requestBlock(peerID, next)
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
	if n.enableV2 {
		_ = n.sendV2To(peerID, V2MessageGetPeers, struct{}{})
		if n.blockchain != nil && peerHeight > n.blockchain.Height() {
			_ = n.requestHeadersV2(peerID)
		}
		return
	}

	_ = n.sendTo(peerID, getPeersMessage{Type: messageTypeGetPeers})

	if n.blockchain != nil && peerHeight > n.blockchain.Height() {
		_ = n.requestBlock(peerID, n.blockchain.Height()+1)
	}
}

func (n *Node) revalidateMempool() {
	if n.blockchain == nil {
		return
	}
	removed := n.mempool.Revalidate(func(tx *transaction.Transaction) (uint64, error) {
		return n.blockchain.CalculateFees([]*transaction.Transaction{tx})
	})
	if removed > 0 {
		logging.Printf(
			logging.CategoryMempool,
			"revalidated removed=%d size=%d",
			removed,
			n.mempool.Len(),
		)
	}
}

func (n *Node) applyMempoolChainUpdate(update blockchain.ChainUpdate) {
	if !update.Activated {
		return
	}

	for _, connected := range update.Connected {
		if connected != nil {
			n.mempool.RemoveBlockTransactions(connected.Transactions)
		}
	}

	// Revalidate survivors first against the new active UTXO view so old
	// conflicts do not prevent legitimate disconnected transactions returning.
	n.revalidateMempool()

	for _, disconnected := range update.Disconnected {
		if disconnected == nil {
			continue
		}
		for _, tx := range disconnected.Transactions {
			if tx == nil || tx.IsCoinbase() {
				continue
			}
			_ = n.mempool.Reconsider(
				tx,
				func(candidate *transaction.Transaction) (uint64, error) {
					return n.blockchain.CalculateFees([]*transaction.Transaction{candidate})
				},
			)
		}
	}

	n.revalidateMempool()
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
	if n.enableV2 {
		return n.exchangeHelloV2(conn, inbound)
	}

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

func (n *Node) exchangeHelloV2(conn net.Conn, inbound bool) (Peer, error) {
	if err := conn.SetDeadline(time.Now().Add(n.handshakeTimeout)); err != nil {
		return Peer{}, err
	}
	defer func() {
		_ = conn.SetDeadline(time.Time{})
	}()

	tipHash := ""
	if n.blockchain != nil && n.blockchain.Tip() != nil {
		tipHash = n.blockchain.Tip().BlockHash
	}

	local := V2Hello{
		ChainID:             n.networkProfile.ChainID,
		ProtocolMin:         n.networkProfile.ProtocolMin,
		ProtocolMax:         n.networkProfile.ProtocolMax,
		NodeID:              n.nodeID,
		Services:            []string{"network"},
		ListenAddress:       n.Address(),
		Height:              n.heightProvider(),
		TipHash:             tipHash,
		CumulativeChainwork: "0",
		UserAgent:           "/" + valdrconfig.ProjectName + ":" + valdrconfig.Version + "/",
		Timestamp:           time.Now().UTC().Unix(),
		Nonce:               uint64(time.Now().UnixNano()),
	}
	if err := WriteV2Frame(
		conn,
		n.networkProfile,
		n.networkProfile.ProtocolMax,
		V2MessageHello,
		local,
	); err != nil {
		return Peer{}, err
	}

	frame, err := ReadV2Frame(conn, n.networkProfile)
	if err != nil {
		return Peer{}, err
	}
	if frame.MessageType != V2MessageHello {
		return Peer{}, fmt.Errorf("%w: expected hello", ErrInvalidHello)
	}
	var remote V2Hello
	if err := DecodeV2Payload(frame, &remote); err != nil {
		return Peer{}, err
	}
	selected, err := NegotiateV2Hello(
		n.networkProfile,
		n.networkProfile.ProtocolMin,
		n.networkProfile.ProtocolMax,
		remote,
	)
	if err != nil {
		return Peer{}, err
	}
	if remote.NodeID == n.nodeID {
		return Peer{}, ErrSelfConnection
	}

	if err := WriteV2Frame(
		conn,
		n.networkProfile,
		selected,
		V2MessageHelloAck,
		V2HelloAck{ProtocolVersion: selected},
	); err != nil {
		return Peer{}, err
	}

	ackFrame, err := ReadV2Frame(conn, n.networkProfile)
	if err != nil {
		return Peer{}, err
	}
	if ackFrame.MessageType != V2MessageHelloAck {
		return Peer{}, fmt.Errorf("%w: expected hello_ack", ErrInvalidHello)
	}
	var ack V2HelloAck
	if err := DecodeV2Payload(ackFrame, &ack); err != nil {
		return Peer{}, err
	}
	if ack.ProtocolVersion != selected {
		return Peer{}, ErrV2UnsupportedVersion
	}

	return Peer{
		NodeID:          remote.NodeID,
		Address:         remote.ListenAddress,
		Height:          remote.Height,
		ProtocolVersion: uint32(selected),
		Inbound:         inbound,
	}, nil
}

func (n *Node) handleV2Frame(peerID string, frame V2Frame) error {
	switch frame.MessageType {
	case V2MessagePing:
		var ping struct {
			Nonce uint64 `json:"nonce"`
		}
		if err := DecodeV2Payload(frame, &ping); err != nil {
			return err
		}
		return n.sendV2To(peerID, V2MessagePong, ping)

	case V2MessagePong:
		var pong struct {
			Nonce uint64 `json:"nonce"`
		}
		return DecodeV2Payload(frame, &pong)

	case V2MessageGetHeaders:
		var request V2LocatorRequest
		if err := DecodeV2Payload(frame, &request); err != nil {
			return err
		}
		return n.handleGetHeadersV2(peerID, request)

	case V2MessageHeaders:
		var payload V2HeadersPayload
		if err := DecodeV2Payload(frame, &payload); err != nil {
			return err
		}
		return n.handleHeadersV2(peerID, payload)

	case V2MessageGetData:
		var payload V2GetDataPayload
		if err := DecodeV2Payload(frame, &payload); err != nil {
			return err
		}
		return n.handleGetDataV2(peerID, payload)

	case V2MessageBlock:
		var payload V2BlockPayload
		if err := DecodeV2Payload(frame, &payload); err != nil {
			return err
		}
		return n.handleBlockV2(peerID, payload)

	case V2MessageTx:
		var payload V2TxPayload
		if err := DecodeV2Payload(frame, &payload); err != nil {
			return err
		}
		if payload.Transaction == nil {
			return ErrInvalidFrame
		}
		return n.handleTransaction(peerID, payload.Transaction)

	case V2MessageGetBlocks:
		var request V2LocatorRequest
		if err := DecodeV2Payload(frame, &request); err != nil {
			return err
		}
		return n.handleGetBlocksV2(peerID, request)

	case V2MessageInv:
		var payload V2InvPayload
		if err := DecodeV2Payload(frame, &payload); err != nil {
			return err
		}
		return n.handleInvV2(peerID, payload)

	case V2MessageGetPeers:
		var request struct{}
		if err := DecodeV2Payload(frame, &request); err != nil {
			return err
		}
		return n.handleGetPeersV2(peerID)

	case V2MessagePeers:
		var message struct {
			Peers []peerAdvertisement `json:"peers"`
		}
		if err := DecodeV2Payload(frame, &message); err != nil {
			return err
		}
		return n.handlePeers(message.Peers)

	default:
		return fmt.Errorf(
			"%w: v2 message type %d is reserved for a later v0.2 stage",
			ErrUnknownMessage,
			frame.MessageType,
		)
	}
}

func (n *Node) requestHeadersV2(peerID string) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	locator := n.blockchain.BlockLocator()
	if len(locator) == 0 || len(locator) > V2MaxLocatorEntries {
		return ErrV2InvalidLocator
	}
	return n.sendV2To(peerID, V2MessageGetHeaders, V2LocatorRequest{
		Locator:  locator,
		StopHash: "",
	})
}

func (n *Node) handleGetHeadersV2(peerID string, request V2LocatorRequest) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := ValidateV2LocatorRequest(request); err != nil {
		return err
	}
	headers, _, err := n.blockchain.HeadersAfterLocator(
		request.Locator,
		V2MaxHeaders,
	)
	if err != nil {
		return err
	}
	if request.StopHash != "" {
		for index, header := range headers {
			if header.BlockHash == request.StopHash {
				headers = headers[:index+1]
				break
			}
		}
	}
	return n.sendV2To(peerID, V2MessageHeaders, V2HeadersPayload{
		Headers: headers,
	})
}

func (n *Node) handleHeadersV2(peerID string, payload V2HeadersPayload) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := ValidateV2HeadersPayload(payload); err != nil {
		return err
	}
	if len(payload.Headers) == 0 {
		n.mu.Lock()
		delete(n.syncV2, peerID)
		n.mu.Unlock()
		return nil
	}

	history, ok := n.blockchain.HeaderHistoryByHash(
		payload.Headers[0].PreviousBlockHash,
	)
	if !ok {
		return fmt.Errorf("%w: first header parent is unknown", ErrV2InvalidHeaders)
	}

	state := &v2SyncState{
		pending:     make(map[string]block.Header),
		requested:   make(map[string]struct{}),
		moreHeaders: len(payload.Headers) == V2MaxHeaders,
	}
	for index, header := range payload.Headers {
		special, err := consensus.ValidateHeaderV2(
			header,
			history,
			n.networkProfile,
			time.Now().UTC().Unix(),
		)
		if err != nil {
			return fmt.Errorf(
				"%w: header[%d]: %v",
				ErrV2InvalidHeaders,
				index,
				err,
			)
		}
		target, err := consensus.ParseTargetHexV2(header.Target)
		if err != nil {
			return fmt.Errorf("%w: header[%d] target", ErrV2InvalidHeaders, index)
		}
		history = append(history, consensus.V2DifficultyHeader{
			Height:               header.Height,
			BlockHash:            header.BlockHash,
			Timestamp:            header.Timestamp,
			Target:               target,
			SpecialMinDifficulty: special,
		})
		if n.blockchain.HasBlock(header.BlockHash) {
			continue
		}
		state.order = append(state.order, header.BlockHash)
		state.pending[header.BlockHash] = header
	}

	n.mu.Lock()
	n.syncV2[peerID] = state
	n.mu.Unlock()
	return n.requestNextBodiesV2(peerID)
}

func (n *Node) requestNextBodiesV2(peerID string) error {
	n.mu.Lock()
	state := n.syncV2[peerID]
	if state == nil {
		n.mu.Unlock()
		return nil
	}
	items := make([]V2InventoryItem, 0, V2MaxInventoryItems)
	for _, hash := range state.order {
		if len(items) >= V2MaxInventoryItems {
			break
		}
		if _, pending := state.pending[hash]; !pending {
			continue
		}
		if _, requested := state.requested[hash]; requested {
			continue
		}
		state.requested[hash] = struct{}{}
		items = append(items, V2InventoryItem{
			Kind: V2InventoryBlock,
			Hash: hash,
		})
	}
	moreHeaders := state.moreHeaders
	done := len(state.pending) == 0 && len(state.requested) == 0
	if done {
		delete(n.syncV2, peerID)
	}
	n.mu.Unlock()

	if len(items) > 0 {
		return n.sendV2To(peerID, V2MessageGetData, V2GetDataPayload{
			Items: items,
		})
	}
	if done && moreHeaders {
		return n.requestHeadersV2(peerID)
	}
	return nil
}

func (n *Node) handleGetDataV2(peerID string, payload V2GetDataPayload) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := ValidateV2GetDataPayload(payload); err != nil {
		return err
	}
	for _, item := range payload.Items {
		candidate, ok := n.blockchain.BlockByHash(item.Hash)
		if !ok {
			continue
		}
		if err := n.sendV2To(peerID, V2MessageBlock, V2BlockPayload{
			Block: candidate,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) handleBlockV2(peerID string, payload V2BlockPayload) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if payload.Block == nil {
		return ErrInvalidFrame
	}
	candidate := payload.Block

	n.mu.Lock()
	state := n.syncV2[peerID]
	if state == nil {
		n.mu.Unlock()
		return fmt.Errorf("%w: unsolicited v2 block", ErrUnknownBlock)
	}
	expected, exists := state.pending[candidate.BlockHash]
	_, requested := state.requested[candidate.BlockHash]
	n.mu.Unlock()
	if !exists || !requested || candidate.Header() != expected {
		return fmt.Errorf("%w: block body does not match requested header", ErrUnknownBlock)
	}

	update, err := n.blockchain.AddBlockWithUpdate(candidate)
	if err != nil {
		return err
	}
	n.applyMempoolChainUpdate(update)
	n.updatePeerHeight(peerID, candidate.Height)

	n.mu.Lock()
	if state := n.syncV2[peerID]; state != nil {
		delete(state.pending, candidate.BlockHash)
		delete(state.requested, candidate.BlockHash)
	}
	n.mu.Unlock()

	logging.Printf(
		logging.CategorySync,
		"v2 body accepted height=%d hash=%s peer=%s",
		candidate.Height,
		candidate.BlockHash,
		peerID,
	)
	return n.requestNextBodiesV2(peerID)
}

func (n *Node) handleGetBlocksV2(peerID string, request V2LocatorRequest) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := ValidateV2LocatorRequest(request); err != nil {
		return err
	}
	headers, _, err := n.blockchain.HeadersAfterLocator(
		request.Locator,
		V2MaxInventoryItems,
	)
	if err != nil {
		return err
	}
	items := make([]V2InventoryItem, 0, len(headers))
	for _, header := range headers {
		items = append(items, V2InventoryItem{
			Kind: V2InventoryBlock,
			Hash: header.BlockHash,
		})
		if request.StopHash != "" && header.BlockHash == request.StopHash {
			break
		}
	}
	return n.sendV2To(peerID, V2MessageInv, V2InvPayload{Items: items})
}

func (n *Node) handleInvV2(peerID string, payload V2InvPayload) error {
	if n.blockchain == nil {
		return ErrDataLayerUnavailable
	}
	if err := ValidateV2InvPayload(payload); err != nil {
		return err
	}
	for _, item := range payload.Items {
		if !n.blockchain.HasBlock(item.Hash) {
			// Stage 7 is strictly headers-first: inventory only signals that
			// new data exists. Fetch and validate headers before any body.
			return n.requestHeadersV2(peerID)
		}
	}
	return nil
}

func (n *Node) handleGetPeersV2(peerID string) error {
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

	return n.sendV2To(peerID, V2MessagePeers, struct {
		Peers []peerAdvertisement `json:"peers"`
	}{Peers: advertisements})
}

func (n *Node) broadcastV2Except(
	excludedPeerID string,
	messageType V2MessageType,
	value any,
) error {
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
		if err := n.sendV2To(peerID, messageType, value); err != nil &&
			firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (n *Node) sendV2To(peerID string, messageType V2MessageType, value any) error {
	n.mu.RLock()
	pc, exists := n.conns[peerID]
	peer := n.peers[peerID]
	n.mu.RUnlock()
	if !exists {
		return fmt.Errorf("peer %q not connected", peerID)
	}

	pc.writeMu.Lock()
	defer pc.writeMu.Unlock()
	return WriteV2Frame(
		pc.conn,
		n.networkProfile,
		uint16(peer.ProtocolVersion),
		messageType,
		value,
	)
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
	logging.Printf(
		logging.CategoryP2P,
		"peer connected node=%s address=%s inbound=%t height=%d",
		peer.NodeID,
		peer.Address,
		peer.Inbound,
		peer.Height,
	)
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
		n.mu.Lock()
		delete(n.syncV2, peerID)
		n.mu.Unlock()
		_ = current.conn.Close()
		logging.Printf(logging.CategoryP2P, "peer disconnected node=%s", peerID)
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
