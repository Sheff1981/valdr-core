package mining

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sort"

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

type transactionCandidate struct {
	tx   *transaction.Transaction
	fee  uint64
	size int
}

// MineBlock creates the required coinbase transaction, selects valid normal
// transactions by fee-rate, respects the consensus block-size limit, mines PoW
// and commits the block after full blockchain/UTXO validation.
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

	candidates := make([]transactionCandidate, 0, len(transactions))
	for i, tx := range transactions {
		if tx == nil {
			return nil, fmt.Errorf("nil transaction at index %d", i)
		}
		if tx.HasCoinbaseMarker() {
			return nil, fmt.Errorf("%w at transaction %d", ErrCoinbaseProvided, i)
		}
		fee, err := chain.CalculateFees([]*transaction.Transaction{tx})
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, transactionCandidate{
			tx:   tx,
			fee:  fee,
			size: tx.SerializedSize(),
		})
	}

	selected, selectedFees, err := selectTransactions(
		chain,
		tip,
		height,
		minerAddress,
		timestamp,
		reward,
		candidates,
		block.MaxSerializedSize,
	)
	if err != nil {
		return nil, err
	}

	if math.MaxUint64-reward < selectedFees {
		return nil, fmt.Errorf(
			"coinbase reward overflow: subsidy=%d fees=%d",
			reward,
			selectedFees,
		)
	}
	coinbase, err := transaction.NewCoinbase(
		height,
		minerAddress,
		reward+selectedFees,
		timestamp,
	)
	if err != nil {
		return nil, err
	}

	blockTransactions := make([]*transaction.Transaction, 0, len(selected)+1)
	blockTransactions = append(blockTransactions, coinbase)
	blockTransactions = append(blockTransactions, selected...)

	return chain.Append(timestamp, blockTransactions)
}

func selectTransactions(
	chain *blockchain.Blockchain,
	tip *block.Block,
	height uint64,
	minerAddress string,
	timestamp int64,
	reward uint64,
	candidates []transactionCandidate,
	maxBlockBytes int,
) ([]*transaction.Transaction, uint64, error) {
	ordered := append([]transactionCandidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		cmp := compareFeeRate(ordered[i], ordered[j])
		if cmp != 0 {
			return cmp > 0
		}
		return ordered[i].tx.TransactionID < ordered[j].tx.TransactionID
	})

	selected := make([]*transaction.Transaction, 0, len(ordered))
	var selectedFees uint64
	for _, candidate := range ordered {
		prospective := append(
			append([]*transaction.Transaction(nil), selected...),
			candidate.tx,
		)
		fees, err := chain.CalculateFees(prospective)
		if err != nil {
			continue
		}
		if math.MaxUint64-reward < fees {
			return nil, 0, fmt.Errorf(
				"coinbase reward overflow: subsidy=%d fees=%d",
				reward,
				fees,
			)
		}

		coinbase, err := transaction.NewCoinbase(
			height,
			minerAddress,
			reward+fees,
			timestamp,
		)
		if err != nil {
			return nil, 0, err
		}
		blockTransactions := make(
			[]*transaction.Transaction,
			0,
			len(prospective)+1,
		)
		blockTransactions = append(blockTransactions, coinbase)
		blockTransactions = append(blockTransactions, prospective...)
		sizeProbe := block.New(
			height,
			tip.BlockHash,
			timestamp,
			tip.Difficulty,
			0,
			blockTransactions,
			"",
		)
		if sizeProbe.SerializedSize() > maxBlockBytes {
			continue
		}

		selected = prospective
		selectedFees = fees
	}
	return selected, selectedFees, nil
}

func compareFeeRate(a, b transactionCandidate) int {
	aHi, aLo := bits.Mul64(a.fee, uint64(b.size))
	bHi, bLo := bits.Mul64(b.fee, uint64(a.size))
	if aHi < bHi {
		return -1
	}
	if aHi > bHi {
		return 1
	}
	if aLo < bLo {
		return -1
	}
	if aLo > bLo {
		return 1
	}
	return 0
}
