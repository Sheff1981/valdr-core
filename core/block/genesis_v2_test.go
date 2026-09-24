package block

import (
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/consensus"
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
	target, err := consensus.ParseTargetHexV2(genesis.Target)
	if err != nil {
		t.Fatal(err)
	}
	if err := consensus.ValidatePoWTarget(genesis.BlockHash, target); err != nil {
		t.Fatalf("frozen Testnet Genesis PoW invalid: %v", err)
	}
}
