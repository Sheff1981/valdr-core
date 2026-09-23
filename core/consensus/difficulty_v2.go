package consensus

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
)

var (
	ErrInvalidDifficultyV2Config = errors.New("invalid difficulty v2 network configuration")
	ErrInvalidTarget             = errors.New("invalid proof-of-work target")
	ErrTimestampMedianPast       = errors.New("block timestamp is not greater than median-time-past")
	ErrTimestampTooFarFuture     = errors.New("block timestamp is too far in the future")
	ErrInvalidDifficultyHistory  = errors.New("invalid difficulty history")
)

type V2DifficultyHeader struct {
	Height               uint64
	BlockHash            string
	Timestamp            int64
	Target               *big.Int
	SpecialMinDifficulty bool
}

func PowLimitForProfile(profile config.NetworkProfile) (*big.Int, error) {
	if profile.PowLimitLeadingZeroBits == 0 || profile.PowLimitLeadingZeroBits >= 256 {
		return nil, ErrInvalidDifficultyV2Config
	}
	limit := new(big.Int).Lsh(
		big.NewInt(1),
		uint(256)-profile.PowLimitLeadingZeroBits,
	)
	return limit.Sub(limit, big.NewInt(1)), nil
}

func TargetHexV2(target *big.Int) (string, error) {
	if target == nil || target.Sign() <= 0 || target.BitLen() > 256 {
		return "", ErrInvalidTarget
	}
	return fmt.Sprintf("%064x", target), nil
}

func ParseTargetHexV2(value string) (*big.Int, error) {
	raw, err := block.DecodeTarget(value)
	if err != nil {
		return nil, ErrInvalidTarget
	}
	target := new(big.Int).SetBytes(raw)
	if target.Sign() <= 0 || target.BitLen() > 256 {
		return nil, ErrInvalidTarget
	}
	return target, nil
}

// ValidateHeaderV2 validates a non-Genesis header using only already accepted
// header history. No transaction/body data is required.
func ValidateHeaderV2(
	header block.Header,
	history []V2DifficultyHeader,
	profile config.NetworkProfile,
	localSystemTime int64,
) (bool, error) {
	if header.Version != block.VersionV2 || header.Difficulty != 0 {
		return false, fmt.Errorf("%w: block version/difficulty", ErrInvalidDifficultyHistory)
	}
	if header.ChainID != profile.ChainID {
		return false, fmt.Errorf(
			"%w: got %q want %q",
			ErrWrongChainID,
			header.ChainID,
			profile.ChainID,
		)
	}
	if len(history) == 0 {
		return false, ErrInvalidDifficultyHistory
	}
	parent := history[len(history)-1]
	if header.Height != parent.Height+1 ||
		parent.BlockHash == "" ||
		header.PreviousBlockHash != parent.BlockHash {
		return false, fmt.Errorf("%w: header linkage", ErrInvalidDifficultyHistory)
	}

	target, err := ParseTargetHexV2(header.Target)
	if err != nil {
		return false, err
	}
	powLimit, err := PowLimitForProfile(profile)
	if err != nil {
		return false, err
	}
	if target.Cmp(powLimit) > 0 {
		return false, ErrInvalidTarget
	}
	if err := ValidateTimestampV2(
		history,
		header.Timestamp,
		localSystemTime,
		profile,
	); err != nil {
		return false, err
	}
	expected, special, err := NextTargetV2(
		history,
		header.Timestamp,
		profile,
	)
	if err != nil {
		return false, err
	}
	if target.Cmp(expected) != 0 {
		return false, fmt.Errorf(
			"%w: got target=%064x want=%064x",
			ErrInvalidDifficultyHistory,
			target,
			expected,
		)
	}

	expectedHash := header.CalculateHash()
	if expectedHash == "" || header.BlockHash != expectedHash {
		return false, ErrInvalidHash
	}
	if err := ValidatePoWTarget(header.BlockHash, target); err != nil {
		return false, err
	}
	return special, nil
}

func RetargetV2(
	oldTarget *big.Int,
	actualTimespan int64,
	profile config.NetworkProfile,
) (*big.Int, error) {
	if err := validateDifficultyV2Profile(profile); err != nil {
		return nil, err
	}
	if oldTarget == nil || oldTarget.Sign() <= 0 || oldTarget.BitLen() > 256 {
		return nil, ErrInvalidTarget
	}

	if actualTimespan < profile.MinRetargetTimespanSeconds {
		actualTimespan = profile.MinRetargetTimespanSeconds
	}
	if actualTimespan > profile.MaxRetargetTimespanSeconds {
		actualTimespan = profile.MaxRetargetTimespanSeconds
	}

	next := new(big.Int).Mul(
		new(big.Int).Set(oldTarget),
		big.NewInt(actualTimespan),
	)
	next.Div(next, big.NewInt(profile.TargetTimespanSeconds))

	powLimit, err := PowLimitForProfile(profile)
	if err != nil {
		return nil, err
	}
	if next.Cmp(powLimit) > 0 {
		next.Set(powLimit)
	}
	if next.Sign() <= 0 {
		return nil, ErrInvalidTarget
	}
	return next, nil
}

// NextTargetV2 applies the frozen v0.2 retarget rules.
//
// A retarget boundary is the first block whose height is divisible by the
// network RetargetInterval. The measured timespan is the difference between
// the first and last timestamps in the preceding interval of block headers.
// This keeps the next target a pure function of already accepted history.
func NextTargetV2(
	history []V2DifficultyHeader,
	candidateTimestamp int64,
	profile config.NetworkProfile,
) (*big.Int, bool, error) {
	if err := validateDifficultyV2Profile(profile); err != nil {
		return nil, false, err
	}
	if len(history) == 0 {
		return nil, false, ErrInvalidDifficultyHistory
	}
	if err := validateDifficultyHistory(history); err != nil {
		return nil, false, err
	}

	parent := history[len(history)-1]
	candidateHeight := parent.Height + 1

	if candidateHeight%profile.RetargetInterval == 0 {
		if uint64(len(history)) < profile.RetargetInterval {
			return nil, false, ErrInvalidDifficultyHistory
		}
		first := history[len(history)-int(profile.RetargetInterval)]
		oldTarget := lastNonSpecialTarget(history)
		if oldTarget == nil {
			return nil, false, ErrInvalidDifficultyHistory
		}
		next, err := RetargetV2(
			oldTarget,
			parent.Timestamp-first.Timestamp,
			profile,
		)
		return next, false, err
	}

	if profile.MinDifficultyAfterSeconds > 0 &&
		candidateTimestamp > parent.Timestamp+profile.MinDifficultyAfterSeconds {
		powLimit, err := PowLimitForProfile(profile)
		return powLimit, true, err
	}

	if parent.SpecialMinDifficulty {
		target := lastNonSpecialTarget(history)
		if target == nil {
			return nil, false, ErrInvalidDifficultyHistory
		}
		return new(big.Int).Set(target), false, nil
	}

	return new(big.Int).Set(parent.Target), false, nil
}

func MedianTimePastV2(
	history []V2DifficultyHeader,
	window int,
) (int64, error) {
	if len(history) == 0 || window <= 0 {
		return 0, ErrInvalidDifficultyHistory
	}
	count := window
	if len(history) < count {
		count = len(history)
	}
	timestamps := make([]int64, 0, count)
	for _, header := range history[len(history)-count:] {
		timestamps = append(timestamps, header.Timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})
	return timestamps[len(timestamps)/2], nil
}

func ValidateTimestampV2(
	history []V2DifficultyHeader,
	candidateTimestamp int64,
	localSystemTime int64,
	profile config.NetworkProfile,
) error {
	if profile.MedianTimePastWindow <= 0 || profile.MaxFutureBlockSeconds <= 0 {
		return ErrInvalidDifficultyV2Config
	}
	median, err := MedianTimePastV2(history, profile.MedianTimePastWindow)
	if err != nil {
		return err
	}
	if candidateTimestamp <= median {
		return fmt.Errorf(
			"%w: candidate=%d median=%d",
			ErrTimestampMedianPast,
			candidateTimestamp,
			median,
		)
	}
	if candidateTimestamp > localSystemTime+profile.MaxFutureBlockSeconds {
		return fmt.Errorf(
			"%w: candidate=%d max=%d",
			ErrTimestampTooFarFuture,
			candidateTimestamp,
			localSystemTime+profile.MaxFutureBlockSeconds,
		)
	}
	return nil
}

func ValidatePoWTarget(blockHash string, target *big.Int) error {
	if target == nil || target.Sign() <= 0 || target.BitLen() > 256 {
		return ErrInvalidTarget
	}
	raw, err := hex.DecodeString(blockHash)
	if err != nil || len(raw) != 32 {
		return ErrInvalidHash
	}
	value := new(big.Int).SetBytes(raw)
	if value.Cmp(target) > 0 {
		return fmt.Errorf(
			"%w: hash=%s target=%064x",
			ErrInvalidPoW,
			blockHash,
			target,
		)
	}
	return nil
}

func MineTarget(candidate *block.Block, target *big.Int) error {
	if candidate == nil {
		return ErrInvalidHash
	}
	if target == nil || target.Sign() <= 0 || target.BitLen() > 256 {
		return ErrInvalidTarget
	}
	for nonce := uint64(0); ; nonce++ {
		candidate.Nonce = nonce
		candidate.BlockHash = candidate.CalculateHash()
		if err := ValidatePoWTarget(candidate.BlockHash, target); err == nil {
			return nil
		}
		if nonce == math.MaxUint64 {
			return ErrNonceExhausted
		}
	}
}

func validateDifficultyV2Profile(profile config.NetworkProfile) error {
	if profile.RetargetInterval == 0 ||
		profile.TargetTimespanSeconds <= 0 ||
		profile.MinRetargetTimespanSeconds <= 0 ||
		profile.MaxRetargetTimespanSeconds < profile.MinRetargetTimespanSeconds ||
		profile.MinRetargetTimespanSeconds > profile.TargetTimespanSeconds ||
		profile.MaxRetargetTimespanSeconds < profile.TargetTimespanSeconds {
		return ErrInvalidDifficultyV2Config
	}
	_, err := PowLimitForProfile(profile)
	return err
}

func validateDifficultyHistory(history []V2DifficultyHeader) error {
	for index, header := range history {
		if header.Target == nil ||
			header.Target.Sign() <= 0 ||
			header.Target.BitLen() > 256 {
			return fmt.Errorf("%w at index %d: target", ErrInvalidDifficultyHistory, index)
		}
		if index == 0 {
			continue
		}
		if header.Height != history[index-1].Height+1 {
			return fmt.Errorf("%w at index %d: height", ErrInvalidDifficultyHistory, index)
		}
	}
	return nil
}

func lastNonSpecialTarget(history []V2DifficultyHeader) *big.Int {
	for index := len(history) - 1; index >= 0; index-- {
		if !history[index].SpecialMinDifficulty && history[index].Target != nil {
			return history[index].Target
		}
	}
	return nil
}
