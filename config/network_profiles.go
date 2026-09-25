package config

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

const (
	NetworkLegacyV01 = "legacy-v0.1"
	NetworkDevnetV02 = "devnet2"
	NetworkTestnetV02 = "testnet"

	DevnetV02GenesisTimestamp = int64(1790121600)
	DevnetV02GenesisTarget = "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	DevnetV02GenesisNonce = uint64(5229)
	DevnetV02GenesisMessage = "VALDR genesis block | valdr-devnet-2 | 2026-09-23"
	DevnetV02GenesisHash = "0009d92e50db69eae0e654399ab73cbc6baf5161105712466c045286db8a2231"

	TestnetV02GenesisTimestamp = int64(1790208000)
	TestnetV02GenesisTarget = "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	TestnetV02GenesisNonce = uint64(12480)
	TestnetV02GenesisMessage = "VALDR genesis block | valdr-testnet-1 | 2026-09-24"
	TestnetV02GenesisHash = "0009d956448a8caefcd798af1a7957840d0aa7b72f8350362909f241ea100122"
)

var ErrUnknownNetworkProfile = errors.New("unknown VALDR network profile")

type NetworkProfile struct {
	Name                   string
	ChainID                string
	BlockVersion           uint32
	GenesisTimestamp       int64
	GenesisTarget          string
	GenesisNonce           uint64
	GenesisMessage         string
	GenesisHash            string
	ProtocolMin            uint16
	ProtocolMax            uint16
	P2PPort                uint16
	RPCPort                uint16
	AddressPrefix          string
	TargetBlockTimeSeconds int64
	InitialSubsidyVDR      uint64
	PowLimitLeadingZeroBits uint
	RetargetInterval       uint64
	TargetTimespanSeconds  int64
	MinRetargetTimespanSeconds int64
	MaxRetargetTimespanSeconds int64
	MedianTimePastWindow   int
	MaxFutureBlockSeconds  int64
	MinDifficultyAfterSeconds int64
	MinRelayFeePerByte     uint64
	DefaultSeeds           []string
	DNSSeeds               []string
	Public                 bool
}

func (p NetworkProfile) Magic() [4]byte {
	sum := sha256.Sum256([]byte(p.ChainID))
	return [4]byte{sum[0], sum[1], sum[2], sum[3]}
}

func ResolveNetworkProfile(name string) (NetworkProfile, error) {
	switch name {
	case NetworkLegacyV01:
		return NetworkProfile{
			Name:                   NetworkLegacyV01,
			ChainID:                "valdr-devnet-1",
			BlockVersion:           1,
			GenesisTimestamp:       GenesisTimestamp,
			GenesisNonce:           GenesisNonce,
			GenesisMessage:         GenesisMessage,
			GenesisHash:            GenesisBlockHash,
			ProtocolMin:            1,
			ProtocolMax:            1,
			P2PPort:                7333,
			RPCPort:                7332,
			AddressPrefix:          "VDR1",
			TargetBlockTimeSeconds: 60,
			InitialSubsidyVDR:      50,
			PowLimitLeadingZeroBits: PowLimitLeadingZeroBits,
		}, nil
	case NetworkDevnetV02:
		return NetworkProfile{
			Name:                   NetworkDevnetV02,
			ChainID:                "valdr-devnet-2",
			BlockVersion:           2,
			GenesisTimestamp:       DevnetV02GenesisTimestamp,
			GenesisTarget:          DevnetV02GenesisTarget,
			GenesisNonce:           DevnetV02GenesisNonce,
			GenesisMessage:         DevnetV02GenesisMessage,
			GenesisHash:            DevnetV02GenesisHash,
			ProtocolMin:            2,
			ProtocolMax:            2,
			P2PPort:                7333,
			RPCPort:                7332,
			AddressPrefix:          "VDR1",
			TargetBlockTimeSeconds: 60,
			InitialSubsidyVDR:      50,
			PowLimitLeadingZeroBits: PowLimitLeadingZeroBits,
			RetargetInterval:       60,
			TargetTimespanSeconds:  3600,
			MinRetargetTimespanSeconds: 900,
			MaxRetargetTimespanSeconds: 14400,
			MedianTimePastWindow:   11,
			MaxFutureBlockSeconds:  2 * 60 * 60,
		}, nil
	case NetworkTestnetV02:
		return NetworkProfile{
			Name:                   NetworkTestnetV02,
			ChainID:                "valdr-testnet-1",
			BlockVersion:           2,
			GenesisTimestamp:       TestnetV02GenesisTimestamp,
			GenesisTarget:          TestnetV02GenesisTarget,
			GenesisNonce:           TestnetV02GenesisNonce,
			GenesisMessage:         TestnetV02GenesisMessage,
			GenesisHash:            TestnetV02GenesisHash,
			ProtocolMin:            2,
			ProtocolMax:            2,
			P2PPort:                17333,
			RPCPort:                17332,
			AddressPrefix:          "VDR1",
			TargetBlockTimeSeconds: 60,
			InitialSubsidyVDR:      50,
			PowLimitLeadingZeroBits: PowLimitLeadingZeroBits,
			RetargetInterval:       60,
			TargetTimespanSeconds:  3600,
			MinRetargetTimespanSeconds: 900,
			MaxRetargetTimespanSeconds: 14400,
			MedianTimePastWindow:   11,
			MaxFutureBlockSeconds:  2 * 60 * 60,
			MinDifficultyAfterSeconds: 10 * 60,
			MinRelayFeePerByte:     1,
			Public:                 true,
		}, nil
	default:
		return NetworkProfile{}, fmt.Errorf("%w: %q", ErrUnknownNetworkProfile, name)
	}
}
