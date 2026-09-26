package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/logging"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/p2p"
)

const maxRPCRequestBytes = 4 * 1024 * 1024

var (
	ErrInvalidServerConfig = errors.New("invalid RPC server config")
	ErrMethodNotFound      = errors.New("RPC method not found")
	ErrInvalidParams       = errors.New("invalid RPC params")
	ErrNotFound            = errors.New("RPC object not found")
)

type Server struct {
	chain     *blockchain.Blockchain
	node      *p2p.Node
	startedAt time.Time
	mineMu    sync.Mutex
}

func NewServer(chain *blockchain.Blockchain, node *p2p.Node) (*Server, error) {
	if chain == nil || node == nil {
		return nil, ErrInvalidServerConfig
	}
	return &Server{chain: chain, node: node, startedAt: time.Now().UTC()}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", s.handleRPC)
	return mux
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(Response{
			Error: &RPCError{Code: -32600, Message: "POST required"},
		})
		return
	}

	// Privileged RPC is not a browser API. Reject browser-originated requests
	// and require application/json so a cross-origin page cannot use a CORS-
	// safelisted content type to trigger state-changing localhost methods.
	if strings.TrimSpace(r.Header.Get("Origin")) != "" {
		s.writeError(w, http.StatusForbidden, -32600, errors.New("browser-origin RPC requests are not allowed"))
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		s.writeError(w, http.StatusUnsupportedMediaType, -32600, errors.New("application/json required"))
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxRPCRequestBytes)
	defer body.Close()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	var request Request
	if err := decoder.Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, -32700, err)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		s.writeError(w, http.StatusBadRequest, -32700, errors.New("multiple JSON values"))
		return
	}

	result, err := s.call(request.Method, request.Params)
	if err != nil {
		status, code := http.StatusBadRequest, -32602
		switch {
		case errors.Is(err, ErrMethodNotFound):
			status, code = http.StatusNotFound, -32601
		case errors.Is(err, ErrNotFound):
			status, code = http.StatusNotFound, -32004
		}
		s.writeError(w, status, code, err)
		return
	}

	_ = json.NewEncoder(w).Encode(Response{Result: result})
}

func (s *Server) call(method string, raw json.RawMessage) (any, error) {
	switch method {
	case MethodGetStatus:
		if err := requireNoParams(raw); err != nil {
			return nil, err
		}
		tip := s.chain.Tip()
		profile := s.chain.Profile()
		tipHash := ""
		target := ""
		lastBlockTime := int64(0)
		blockVersion := profile.BlockVersion
		if tip != nil {
			tipHash = tip.BlockHash
			target = tip.Target
			lastBlockTime = tip.Timestamp
			blockVersion = tip.Version
		}
		height := s.chain.Height()
		bestKnownHeight, syncProgress := calculateSyncProgress(
			height,
			s.node.Peers(),
		)
		var mempoolSizeBytes uint64
		for _, tx := range s.node.MempoolTransactions() {
			if tx != nil {
				mempoolSizeBytes += uint64(tx.SerializedSize())
			}
		}
		uptimeSeconds := uint64(0)
		if !s.startedAt.IsZero() {
			elapsed := time.Since(s.startedAt)
			if elapsed > 0 {
				uptimeSeconds = uint64(elapsed / time.Second)
			}
		}
		return StatusResult{
			Project:                config.ProjectName,
			Ticker:                 config.Ticker,
			Version:                config.Version,
			SourceCommit:           buildSourceCommit(),
			Network:                profile.Name,
			ChainID:                profile.ChainID,
			ProtocolVersion:        profile.ProtocolMax,
			UptimeSeconds:          uptimeSeconds,
			Height:                 height,
			BestKnownHeight:        bestKnownHeight,
			SyncProgress:           syncProgress,
			TipHash:                tipHash,
			Chainwork:              s.chain.Chainwork(),
			BlockVersion:           blockVersion,
			Target:                 target,
			PeerCount:              s.node.PeerCount(),
			MempoolCount:           s.node.MempoolLen(),
			MempoolSizeBytes:       mempoolSizeBytes,
			TargetBlockTimeSeconds: profile.TargetBlockTimeSeconds,
			LastBlockTime:          lastBlockTime,
		}, nil

	case MethodGetBlock:
		var params HeightParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		candidate, ok := s.chain.BlockAt(params.Height)
		if !ok {
			return nil, fmt.Errorf("%w: block height %d", ErrNotFound, params.Height)
		}
		return candidate, nil

	case MethodGetBlockByHash:
		var params HashParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if strings.TrimSpace(params.Hash) == "" {
			return nil, fmt.Errorf("%w: hash", ErrInvalidParams)
		}
		candidate, ok := s.chain.BlockByHash(params.Hash)
		if !ok {
			return nil, fmt.Errorf("%w: block hash %s", ErrNotFound, params.Hash)
		}
		return candidate, nil

	case MethodGetTransaction:
		var params TransactionParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if strings.TrimSpace(params.TransactionID) == "" {
			return nil, fmt.Errorf("%w: txid", ErrInvalidParams)
		}
		if location, ok := s.chain.TransactionByID(params.TransactionID); ok {
			return TransactionResult{
				Transaction: location.Transaction,
				Confirmed:   true,
				BlockHeight: location.BlockHeight,
				BlockHash:   location.BlockHash,
			}, nil
		}
		for _, tx := range s.node.MempoolTransactions() {
			if tx != nil && tx.TransactionID == params.TransactionID {
				return TransactionResult{
					Transaction: tx,
					Confirmed:   false,
				}, nil
			}
		}
		return nil, fmt.Errorf("%w: transaction %s", ErrNotFound, params.TransactionID)

	case MethodGetBalance:
		var params AddressParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		balance, err := s.chain.Balance(params.Address)
		if err != nil {
			return nil, err
		}
		return BalanceResult{
			Address:    params.Address,
			BalanceVal: balance,
		}, nil

	case MethodGetMempool:
		if err := requireNoParams(raw); err != nil {
			return nil, err
		}
		return s.node.MempoolTransactions(), nil

	case MethodSendTransaction:
		var params SendTransactionParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if params.Transaction == nil {
			return nil, fmt.Errorf("%w: transaction", ErrInvalidParams)
		}
		if err := s.node.BroadcastTransaction(params.Transaction); err != nil {
			return nil, err
		}
		return SendTransactionResult{
			TransactionID: params.Transaction.TransactionID,
		}, nil

	case MethodGetPeers:
		if err := requireNoParams(raw); err != nil {
			return nil, err
		}
		return s.node.Peers(), nil

	case MethodGetMiningInfo:
		if err := requireNoParams(raw); err != nil {
			return nil, err
		}
		tip := s.chain.Tip()
		if tip == nil {
			return nil, ErrNotFound
		}
		nextHeight := tip.Height + 1
		profile := s.chain.Profile()
		return MiningInfoResult{
			Height:                 tip.Height,
			NextHeight:             nextHeight,
			CurrentDifficulty:      tip.Difficulty,
			CurrentTarget:          tip.Target,
			BlockRewardVal:         consensus.BlockRewardForProfile(profile, nextHeight),
			TargetBlockTimeSeconds: profile.TargetBlockTimeSeconds,
			RetargetInterval:       profile.RetargetInterval,
			BlocksUntilRetarget:    blocksUntilRetarget(nextHeight, profile.RetargetInterval),
		}, nil

	case MethodGetUTXOs:
		var params AddressParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return s.chain.UTXOs(params.Address)

	case MethodMineBlock:
		var params MineBlockParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return s.mineBlock(params)

	default:
		return nil, fmt.Errorf("%w: %s", ErrMethodNotFound, method)
	}
}

func (s *Server) mineBlock(params MineBlockParams) (MineBlockResult, error) {
	s.mineMu.Lock()
	defer s.mineMu.Unlock()

	tip := s.chain.Tip()
	if tip == nil {
		return MineBlockResult{}, ErrNotFound
	}

	timestamp := time.Now().UTC().Unix()
	if timestamp <= tip.Timestamp {
		timestamp = tip.Timestamp + 1
	}

	transactions := s.node.MempoolTransactionsForMining()
	miningStarted := time.Now()
	candidate, err := mining.MineBlock(
		s.chain,
		params.RewardAddress,
		timestamp,
		transactions,
	)
	miningDuration := time.Since(miningStarted)
	if err != nil {
		logging.Printf(logging.CategoryError, "mining failed error=%v", err)
		return MineBlockResult{}, err
	}

	if err := s.node.BroadcastBlock(candidate); err != nil {
		logging.Printf(
			logging.CategoryError,
			"block mined but broadcast failed height=%d hash=%s error=%v",
			candidate.Height,
			candidate.BlockHash,
			err,
		)
		return MineBlockResult{}, err
	}

	reward := consensus.BlockRewardForProfile(s.chain.Profile(), candidate.Height)
	if len(candidate.Transactions) > 0 &&
		candidate.Transactions[0] != nil &&
		len(candidate.Transactions[0].Outputs) > 0 {
		reward = candidate.Transactions[0].Outputs[0].Amount
	}
	hashesTried := candidate.Nonce + 1
	miningDurationMS := float64(miningDuration) / float64(time.Millisecond)
	hashrateHPS := 0.0
	if miningDuration > 0 {
		hashrateHPS = float64(hashesTried) / miningDuration.Seconds()
	}

	logging.Printf(
		logging.CategoryMiner,
		"block found height=%d hash=%s reward_address=%s txs=%d nonce=%d hashes=%d mining_ms=%.3f hashrate_hps=%.2f",
		candidate.Height,
		candidate.BlockHash,
		params.RewardAddress,
		len(candidate.Transactions),
		candidate.Nonce,
		hashesTried,
		miningDurationMS,
		hashrateHPS,
	)

	return MineBlockResult{
		Height:           candidate.Height,
		BlockHash:        candidate.BlockHash,
		RewardAddress:    params.RewardAddress,
		RewardVal:        reward,
		TransactionCount: len(candidate.Transactions),
		Nonce:            candidate.Nonce,
		HashesTried:      hashesTried,
		MiningDurationMS: miningDurationMS,
		HashrateHPS:      hashrateHPS,
	}, nil
}

func buildSourceCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("%w: params required", ErrInvalidParams)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: multiple JSON values", ErrInvalidParams)
	}
	return nil
}

func requireNoParams(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil
	}
	return fmt.Errorf("%w: method takes no params", ErrInvalidParams)
}

func (s *Server) writeError(w http.ResponseWriter, status, code int, err error) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Error: &RPCError{
			Code:    code,
			Message: err.Error(),
		},
	})
}

func blocksUntilRetarget(nextHeight, interval uint64) uint64 {
	if interval == 0 {
		return 0
	}
	remainder := nextHeight % interval
	if remainder == 0 {
		return 0
	}
	return interval - remainder
}

func calculateSyncProgress(
	height uint64,
	peers []p2p.Peer,
) (uint64, float64) {
	bestKnownHeight := height
	for _, peer := range peers {
		if peer.Height > bestKnownHeight {
			bestKnownHeight = peer.Height
		}
	}
	if bestKnownHeight == 0 {
		return 0, 1
	}
	progress := float64(height) / float64(bestKnownHeight)
	if progress > 1 {
		progress = 1
	}
	return bestKnownHeight, progress
}

// Compile-time reference keeps transaction in the RPC surface explicit.
var _ *transaction.Transaction
