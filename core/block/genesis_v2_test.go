package block

import (
	"errors"
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

func TestTestnetGenesisRemainsUnfrozen(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewGenesisForProfile(profile); !errors.Is(err, ErrGenesisNotFrozen) {
		t.Fatalf("Testnet Genesis error=%v want ErrGenesisNotFrozen", err)
	}
}
