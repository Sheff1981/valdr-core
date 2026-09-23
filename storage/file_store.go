package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

const diskFormatVersion = uint32(1)

var (
	ErrInvalidStorePath = errors.New("invalid block store path")
	ErrInvalidDiskChain = errors.New("invalid persisted blockchain")
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

type diskChain struct {
	FormatVersion uint32         `json:"format_version"`
	ChainID       string         `json:"chain_id"`
	Blocks        []*block.Block `json:"blocks"`
}

func NewFileStore(dataDir string) (*FileStore, error) {
	if dataDir == "" {
		return nil, ErrInvalidStorePath
	}
	return &FileStore{
		path: filepath.Join(dataDir, "blockchain.json"),
	}, nil
}

func (s *FileStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *FileStore) Load() ([]*block.Block, error) {
	if s == nil || s.path == "" {
		return nil, ErrInvalidStorePath
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var persisted diskChain
	if err := decoder.Decode(&persisted); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDiskChain, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing data", ErrInvalidDiskChain)
	}
	if persisted.FormatVersion != diskFormatVersion {
		return nil, fmt.Errorf(
			"%w: format version %d",
			ErrInvalidDiskChain,
			persisted.FormatVersion,
		)
	}
	if persisted.ChainID != config.ChainID {
		return nil, fmt.Errorf(
			"%w: chain id %q",
			ErrInvalidDiskChain,
			persisted.ChainID,
		)
	}
	if len(persisted.Blocks) == 0 {
		return nil, fmt.Errorf("%w: empty block list", ErrInvalidDiskChain)
	}
	return persisted.Blocks, nil
}

func (s *FileStore) Save(blocks []*block.Block) error {
	if s == nil || s.path == "" {
		return ErrInvalidStorePath
	}
	if len(blocks) == 0 {
		return fmt.Errorf("%w: empty block list", ErrInvalidDiskChain)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".blockchain-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return err
	}

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(diskChain{
		FormatVersion: diskFormatVersion,
		ChainID:       config.ChainID,
		Blocks:        blocks,
	}); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
