package rpc

import (
	"encoding/json"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/p2p"
)

const (
	MethodGetStatus       = "getStatus"
	MethodGetBlock        = "getBlock"
	MethodGetBlockByHash  = "getBlockByHash"
	MethodGetTransaction  = "getTransaction"
	MethodGetBalance      = "getBalance"
	MethodGetMempool      = "getMempool"
	MethodSendTransaction = "sendTransaction"
	MethodGetPeers        = "getPeers"
	MethodGetMiningInfo   = "getMiningInfo"
	MethodMineBlock       = "mineBlock"

	// MethodGetUTXOs is a Day 11 helper used by valdr-cli send so transaction
	// selection/signing stays local to the wallet and private keys never enter RPC.
	MethodGetUTXOs = "getUTXOs"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Result any       `json:"result,omitempty"`
	Error  *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HeightParams struct {
	Height uint64 `json:"height"`
}

type HashParams struct {
	Hash string `json:"hash"`
}

type TransactionParams struct {
	TransactionID string `json:"txid"`
}

type AddressParams struct {
	Address string `json:"address"`
}

type SendTransactionParams struct {
	Transaction *transaction.Transaction `json:"transaction"`
}

type MineBlockParams struct {
	RewardAddress string `json:"reward_address"`
}

type StatusResult struct {
	Project                string `json:"project"`
	Ticker                 string `json:"ticker"`
	Version                string `json:"version"`
	SourceCommit           string `json:"source_commit,omitempty"`
	Network                string `json:"network"`
	ChainID                string `json:"chain_id"`
	ProtocolVersion        uint16 `json:"protocol_version"`
	UptimeSeconds          uint64 `json:"uptime_seconds"`
	Height                 uint64  `json:"height"`
	BestKnownHeight        uint64  `json:"best_known_height"`
	SyncProgress           float64 `json:"sync_progress"`
	TipHash                string  `json:"tip_hash"`
	Chainwork              string `json:"chainwork"`
	BlockVersion           uint32 `json:"block_version"`
	Target                 string `json:"target,omitempty"`
	PeerCount              int    `json:"peer_count"`
	MempoolCount           int    `json:"mempool_count"`
	MempoolSizeBytes       uint64 `json:"mempool_size_bytes"`
	TargetBlockTimeSeconds int64  `json:"target_block_time_seconds"`
	LastBlockTime          int64  `json:"last_block_time"`
}

type BalanceResult struct {
	Address    string `json:"address"`
	BalanceVal uint64 `json:"balance_val"`
}

type TransactionResult struct {
	Transaction *transaction.Transaction `json:"transaction"`
	Confirmed   bool                     `json:"confirmed"`
	BlockHeight uint64                   `json:"block_height,omitempty"`
	BlockHash   string                   `json:"block_hash,omitempty"`
}

type SendTransactionResult struct {
	TransactionID string `json:"transaction_id"`
}

type MiningInfoResult struct {
	Height                 uint64 `json:"height"`
	NextHeight             uint64 `json:"next_height"`
	CurrentDifficulty      uint64 `json:"current_difficulty"`
	CurrentTarget          string `json:"current_target,omitempty"`
	BlockRewardVal         uint64 `json:"block_reward_val"`
	TargetBlockTimeSeconds int64  `json:"target_block_time_seconds"`
	RetargetInterval       uint64 `json:"retarget_interval"`
	BlocksUntilRetarget    uint64 `json:"blocks_until_retarget"`
}

type MineBlockResult struct {
	Height           uint64  `json:"height"`
	BlockHash        string  `json:"block_hash"`
	RewardAddress    string  `json:"reward_address"`
	RewardVal        uint64  `json:"reward_val"`
	TransactionCount int     `json:"transaction_count"`
	Nonce            uint64  `json:"nonce"`
	HashesTried      uint64  `json:"hashes_tried"`
	MiningDurationMS float64 `json:"mining_duration_ms"`
	HashrateHPS      float64 `json:"hashrate_hps"`
}

type BlockResult = block.Block
type PeerResult = p2p.Peer
type UTXOResult = utxo.UTXO
