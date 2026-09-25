package block

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestDevnetV02GenesisGolden(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if genesis.Version != VersionV2 ||
		genesis.Height != 0 ||
		genesis.ChainID != profile.ChainID ||
		genesis.Target != config.DevnetV02GenesisTarget ||
		genesis.Nonce != config.DevnetV02GenesisNonce ||
		genesis.BlockHash != config.DevnetV02GenesisHash {
		t.Fatalf("unexpected Devnet2 Genesis: %+v", genesis)
	}
}

func TestTestnetV02GenesisGolden(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if genesis.Version != VersionV2 ||
		genesis.Height != 0 ||
		genesis.ChainID != "valdr-testnet-1" ||
		genesis.Timestamp != config.TestnetV02GenesisTimestamp ||
		genesis.Target != config.TestnetV02GenesisTarget ||
		genesis.Nonce != config.TestnetV02GenesisNonce ||
		genesis.ExtraData != config.TestnetV02GenesisMessage ||
		genesis.BlockHash != config.TestnetV02GenesisHash {
		t.Fatalf("unexpected Testnet Genesis: %+v", genesis)
	}
	targetBytes, err := hex.DecodeString(genesis.Target)
	if err != nil {
		t.Fatal(err)
	}
	hashBytes, err := hex.DecodeString(genesis.BlockHash)
	if err != nil {
		t.Fatal(err)
	}
	if new(big.Int).SetBytes(hashBytes).Cmp(new(big.Int).SetBytes(targetBytes)) > 0 {
		t.Fatalf("frozen Testnet Genesis hash exceeds target")
	}
}

func TestTestnetV029GenesisGolden(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	genesis, err := NewGenesisForProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if genesis.Version != VersionV2 ||
		genesis.Height != 0 ||
		genesis.ChainID != "valdr-testnet-2" ||
		genesis.Timestamp != config.TestnetV029GenesisTimestamp ||
		genesis.Target != config.TestnetV029GenesisTarget ||
		genesis.Nonce != config.TestnetV029GenesisNonce ||
		genesis.ExtraData != config.TestnetV029GenesisMessage ||
		genesis.BlockHash != config.TestnetV029GenesisHash {
		t.Fatalf("unexpected Testnet v0.2.9 Genesis: %+v", genesis)
	}
	targetBytes, err := hex.DecodeString(genesis.Target)
	if err != nil {
		t.Fatal(err)
	}
	hashBytes, err := hex.DecodeString(genesis.BlockHash)
	if err != nil {
		t.Fatal(err)
	}
	if new(big.Int).SetBytes(hashBytes).Cmp(new(big.Int).SetBytes(targetBytes)) > 0 {
		t.Fatalf("frozen Testnet v0.2.9 Genesis hash exceeds target")
	}
}
