package mining

import (
	"errors"
	"fmt"
	"math"

	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/consensus"
	"github.com/Sheff1981/valdr-core/core/transaction"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
)

var (
	ErrNilBlockchain    = errors.New("blockchain is nil")
	ErrInvalidMiner     = errors.New("invalid miner address")
	ErrCoinbaseProvided = errors.New("caller must not provide coinbase transaction")
)

// MineBlock creates the required coinbase transaction, mines PoW and commits
// the block to the local chain after full blockchain/UTXO validation.
func MineBlock(
	chain *blockchain.Blockchain,
	minerAddress string,
	timestamp int64,
	transactions []*transaction.Transaction,
) (*block.Block, error) {
	if chain == nil {
		return nil, ErrNilBlockchain
	}
	if !valdrcrypto.ValidateAddress(minerAddress) {
		return nil, ErrInvalidMiner
	}

	tip := chain.Tip()
	if tip == nil {
		return nil, ErrNilBlockchain
	}
	height := tip.Height + 1
	reward := consensus.BlockReward(height)
	if reward == 0 {
		return nil, fmt.Errorf("no block reward configured for height %d", height)
	}

	for i, tx := range transactions {
		if tx != nil && tx.HasCoinbaseMarker() {
			return nil, fmt.Errorf("%w at transaction %d", ErrCoinbaseProvided, i)
		}
	}

	fees, err := chain.CalculateFees(transactions)
	if err != nil {
		return nil, err
	}
	if math.MaxUint64-reward < fees {
		return nil, fmt.Errorf("coinbase reward overflow: subsidy=%d fees=%d", reward, fees)
	}
	claim := reward + fees

	coinbase, err := transaction.NewCoinbase(
		height,
		minerAddress,
		claim,
		timestamp,
	)
	if err != nil {
		return nil, err
	}

	blockTransactions := make([]*transaction.Transaction, 0, len(transactions)+1)
	blockTransactions = append(blockTransactions, coinbase)
	blockTransactions = append(blockTransactions, transactions...)

	return chain.Append(timestamp, blockTransactions)
}
