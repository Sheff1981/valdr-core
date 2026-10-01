package main

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
)

func TestCandidateIdentityDoesNotCollideWithExistingNetworks(t *testing.T) {
	if CandidateChainID == "" {
		t.Fatal("candidate Chain ID must not be empty")
	}
	if CandidateP2PPort == 0 || CandidateRPCPort == 0 || CandidateP2PPort == CandidateRPCPort {
		t.Fatalf("invalid candidate ports p2p=%d rpc=%d", CandidateP2PPort, CandidateRPCPort)
	}
	if CandidateProtocol == 0 {
		t.Fatal("candidate protocol version must be non-zero")
	}

	for _, name := range []string{
		config.NetworkLegacyV01,
		config.NetworkDevnetV02,
		config.NetworkTestnetV02,
		config.NetworkTestnetV029,
	} {
		profile, err := config.ResolveNetworkProfile(name)
		if err != nil {
			t.Fatal(err)
		}
		if profile.ChainID == CandidateChainID {
			t.Fatalf("candidate Chain ID collides with %s", name)
		}
		if profile.P2PPort == CandidateP2PPort || profile.RPCPort == CandidateP2PPort {
			t.Fatalf("candidate P2P port %d collides with %s", CandidateP2PPort, name)
		}
		if profile.P2PPort == CandidateRPCPort || profile.RPCPort == CandidateRPCPort {
			t.Fatalf("candidate RPC port %d collides with %s", CandidateRPCPort, name)
		}
		if profile.Magic() == candidateMagic() {
			t.Fatalf("candidate P2P magic collides with %s", name)
		}
	}
}

func TestCandidateMagicGolden(t *testing.T) {
	magic := candidateMagic()
	if got := hex.EncodeToString(magic[:]); got != "12795b10" {
		t.Fatalf("magic=%s want=12795b10", got)
	}
}

func TestMainnetProfileRemainsDisabled(t *testing.T) {
	if _, err := config.ResolveNetworkProfile("mainnet"); !errors.Is(err, config.ErrUnknownNetworkProfile) {
		t.Fatalf("mainnet profile unexpectedly active: %v", err)
	}
}

func TestCandidateDataDirIsSeparatedFromExistingProfileNames(t *testing.T) {
	for _, name := range []string{
		config.NetworkLegacyV01,
		config.NetworkDevnetV02,
		config.NetworkTestnetV02,
		config.NetworkTestnetV029,
	} {
		if CandidateDataDir == name {
			t.Fatalf("candidate data dir %q collides with network profile %q", CandidateDataDir, name)
		}
	}
}
