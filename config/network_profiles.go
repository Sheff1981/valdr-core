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
)

var ErrUnknownNetworkProfile = errors.New("unknown VALDR network profile")

type NetworkProfile struct {
	Name                   string
	ChainID                string
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
