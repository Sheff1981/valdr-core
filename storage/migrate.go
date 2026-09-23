package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
)

var ErrUnsupportedMigrationNetwork = errors.New("unsupported v0.1 migration network")

type MigrationReport struct {
	SourceDataDir     string `json:"source_data_dir"`
	TargetDatabase    string `json:"target_database"`
	Network           string `json:"network"`
	Height            uint64 `json:"height"`
	TipHash           string `json:"tip_hash"`
	ConfirmedTxCount  int    `json:"confirmed_tx_count"`
	UTXOHash          string `json:"utxo_hash"`
	OriginalPreserved bool   `json:"original_preserved"`
}

func MigrateV01(sourceDataDir, network string) (MigrationReport, error) {
	if sourceDataDir == "" {
		return MigrationReport{}, ErrInvalidStorePath
	}
	if network != "valdr-devnet-1" {
		return MigrationReport{}, fmt.Errorf("%w: %q", ErrUnsupportedMigrationNetwork, network)
	}

	legacyPath := filepath.Join(sourceDataDir, "blockchain.json")
	if _, err := os.Stat(legacyPath); err != nil {
		return MigrationReport{}, fmt.Errorf("legacy blockchain.json: %w", err)
	}

	targetPath := BadgerPath(sourceDataDir)
	if entries, err := os.ReadDir(targetPath); err == nil && len(entries) > 0 {
		return MigrationReport{}, fmt.Errorf("%w: %s", ErrMigrationTargetExists, targetPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return MigrationReport{}, err
	}

	legacyStore, err := NewFileStore(sourceDataDir)
	if err != nil {
		return MigrationReport{}, err
	}
	sourceChain, err := blockchain.NewPersistent(legacyStore)
	if err != nil {
		return MigrationReport{}, fmt.Errorf("validate v0.1 source: %w", err)
	}
	sourceTip := sourceChain.Tip()
	if sourceTip == nil {
		return MigrationReport{}, errors.New("v0.1 source has no tip")
	}

	sourceTxIDs := confirmedTxIDs(sourceChain)
	sourceUTXOHash := HashUTXOSet(sourceChain.UTXOSnapshot())

	targetStore, err := NewBadgerStore(sourceDataDir, network, config.GenesisBlockHash)
	if err != nil {
		return MigrationReport{}, err
	}
	defer targetStore.Close()

	targetChain, err := blockchain.NewPersistent(targetStore)
	if err != nil {
		return MigrationReport{}, err
	}
	if err := targetStore.MarkMigration("running", filepath.Clean(sourceDataDir), sourceTip.BlockHash); err != nil {
		return MigrationReport{}, err
	}

	for height := uint64(1); height <= sourceChain.Height(); height++ {
		candidate, ok := sourceChain.BlockAt(height)
		if !ok {
			return MigrationReport{}, fmt.Errorf("source block %d missing", height)
		}
		if err := targetChain.AddBlock(candidate); err != nil {
			return MigrationReport{}, fmt.Errorf("migrate block %d: %w", height, err)
		}
	}

	targetTip := targetChain.Tip()
	if targetTip == nil ||
		targetChain.Height() != sourceChain.Height() ||
		targetTip.BlockHash != sourceTip.BlockHash {
		return MigrationReport{}, errors.New("migration height/tip verification failed")
	}

	targetTxIDs := confirmedTxIDs(targetChain)
	if !equalStrings(sourceTxIDs, targetTxIDs) {
		return MigrationReport{}, errors.New("migration confirmed txid verification failed")
	}
	targetUTXOHash := HashUTXOSet(targetChain.UTXOSnapshot())
	if targetUTXOHash != sourceUTXOHash {
		return MigrationReport{}, errors.New("migration UTXO hash verification failed")
	}
	if err := targetStore.MarkMigration("complete", filepath.Clean(sourceDataDir), sourceTip.BlockHash); err != nil {
		return MigrationReport{}, err
	}

	return MigrationReport{
		SourceDataDir:     filepath.Clean(sourceDataDir),
		TargetDatabase:    targetPath,
		Network:           network,
		Height:            targetChain.Height(),
		TipHash:           targetTip.BlockHash,
		ConfirmedTxCount:  len(targetTxIDs),
		UTXOHash:          targetUTXOHash,
		OriginalPreserved: fileExists(legacyPath),
	}, nil
}

func confirmedTxIDs(chain *blockchain.Blockchain) []string {
	if chain == nil {
		return nil
	}
	var ids []string
	for height := uint64(0); height <= chain.Height(); height++ {
		candidate, ok := chain.BlockAt(height)
		if !ok || candidate == nil {
			continue
		}
		for _, tx := range candidate.Transactions {
			if tx != nil {
				ids = append(ids, tx.TransactionID)
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
