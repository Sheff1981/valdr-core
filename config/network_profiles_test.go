package config

import (
	"encoding/hex"
	"errors"
	"testing"
)

func TestV02NetworkProfiles(t *testing.T) {
	tests := []struct {
		name       string
		chainID    string
		protocol   uint16
		p2pPort    uint16
		rpcPort    uint16
		public     bool
		magicHex   string
	}{
		{NetworkLegacyV01, "valdr-devnet-1", 1, 7333, 7332, false, "db9ef71e"},
		{NetworkDevnetV02, "valdr-devnet-2", 2, 7333, 7332, false, "4b6638e3"},
		{NetworkTestnetV02, "valdr-testnet-1", 2, 17333, 17332, true, "614ac40e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := ResolveNetworkProfile(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if profile.ChainID != tt.chainID ||
				profile.ProtocolMin != tt.protocol ||
				profile.ProtocolMax != tt.protocol ||
				profile.P2PPort != tt.p2pPort ||
				profile.RPCPort != tt.rpcPort ||
				profile.Public != tt.public ||
				profile.AddressPrefix != "VDR1" ||
				profile.TargetBlockTimeSeconds != 60 ||
				profile.InitialSubsidyVDR != 50 {
				t.Fatalf("unexpected profile: %+v", profile)
			}
			magic := profile.Magic()
			if got := hex.EncodeToString(magic[:]); got != tt.magicHex {
				t.Fatalf("magic=%s want=%s", got, tt.magicHex)
			}
		})
	}
}

func TestUnknownNetworkProfileRejected(t *testing.T) {
	_, err := ResolveNetworkProfile("mainnet")
	if !errors.Is(err, ErrUnknownNetworkProfile) {
		t.Fatalf("error=%v want ErrUnknownNetworkProfile", err)
	}
}
