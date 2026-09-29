package consensus

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

const (
	MainnetDAATargetSeconds       = int64(600)
	MainnetDAAWindow              = 30
	MainnetDAAMaxSolveSeconds     = int64(3600)
	MainnetDAAMTPWindow           = 11
	MainnetDAAMaxFutureSeconds    = int64(1800)
)

var (
	ErrMainnetDAAHistory      = errors.New("invalid Mainnet DAA history")
	ErrMainnetDAAParameters   = errors.New("invalid Mainnet DAA parameters")
	ErrMainnetDAATimestamp    = errors.New("invalid Mainnet block timestamp")
)

type MainnetDAAHeader struct {
	Height    uint64
	Timestamp int64
	Target    *big.Int
}

// MainnetNextTargetLWMA implements the v0.3.6 Mainnet LWMA candidate.
// InitialTarget and PowLimit remain external inputs until M5 calibration freezes
// exact values. Missing startup samples use target-time solve and InitialWork.
func MainnetNextTargetLWMA(
	history []MainnetDAAHeader,
	initialTarget *big.Int,
	powLimit *big.Int,
) (*big.Int, error) {
	if initialTarget == nil || powLimit == nil ||
		initialTarget.Sign() <= 0 || powLimit.Sign() <= 0 ||
		initialTarget.BitLen() > 256 || powLimit.BitLen() > 256 ||
		initialTarget.Cmp(powLimit) > 0 {
		return nil, ErrMainnetDAAParameters
	}
	if err := validateMainnetDAAHistory(history); err != nil {
		return nil, err
	}

	initialWork := mainnetWorkForTarget(initialTarget)
	if initialWork.Sign() <= 0 {
		return nil, ErrMainnetDAAParameters
	}

	// We need at most N intervals, therefore at most N+1 accepted headers.
	start := 0
	if len(history) > MainnetDAAWindow+1 {
		start = len(history) - (MainnetDAAWindow + 1)
	}
	h := history[start:]

	type sample struct {
		solve int64
		work  *big.Int
	}
	samples := make([]sample, 0, MainnetDAAWindow)

	if len(h) >= 2 {
		prevEff := h[0].Timestamp
		for i := 1; i < len(h); i++ {
			eff := h[i].Timestamp
			if eff <= prevEff {
				eff = prevEff + 1
			}
			solve := eff - prevEff
			if solve > MainnetDAAMaxSolveSeconds {
				solve = MainnetDAAMaxSolveSeconds
			}
			if solve < 1 {
				solve = 1
			}
			samples = append(samples, sample{
				solve: solve,
				work:  mainnetWorkForTarget(h[i].Target),
			})
			prevEff = eff
		}
	}

	if len(samples) > MainnetDAAWindow {
		samples = samples[len(samples)-MainnetDAAWindow:]
	}
	missing := MainnetDAAWindow - len(samples)

	// Missing oldest startup slots are synthetic T-second samples at InitialWork.
	full := make([]sample, 0, MainnetDAAWindow)
	for i := 0; i < missing; i++ {
		full = append(full, sample{
			solve: MainnetDAATargetSeconds,
			work:  new(big.Int).Set(initialWork),
		})
	}
	full = append(full, samples...)

	if len(full) != MainnetDAAWindow {
		return nil, ErrMainnetDAAHistory
	}

	L := big.NewInt(0)
	sumWork := big.NewInt(0)
	for i, s := range full {
		if s.work == nil || s.work.Sign() <= 0 {
			return nil, ErrMainnetDAAHistory
		}
		weight := int64(i + 1)
		L.Add(L, new(big.Int).Mul(big.NewInt(weight), big.NewInt(s.solve)))
		sumWork.Add(sumWork, s.work)
	}

	minL := big.NewInt(int64(MainnetDAAWindow * MainnetDAAWindow))
	minL.Mul(minL, big.NewInt(MainnetDAATargetSeconds))
	minL.Div(minL, big.NewInt(20))
	if L.Cmp(minL) < 0 {
		L.Set(minL)
	}

	avgWork := new(big.Int).Div(sumWork, big.NewInt(MainnetDAAWindow))
	if avgWork.Sign() <= 0 {
		return nil, ErrMainnetDAAHistory
	}

	// nextWork = floor(avgWork*N*(N+1)*T*99/(200*L))
	nextWork := new(big.Int).Set(avgWork)
	nextWork.Mul(nextWork, big.NewInt(MainnetDAAWindow))
	nextWork.Mul(nextWork, big.NewInt(MainnetDAAWindow+1))
	nextWork.Mul(nextWork, big.NewInt(MainnetDAATargetSeconds))
	nextWork.Mul(nextWork, big.NewInt(99))
	den := new(big.Int).Mul(big.NewInt(200), L)
	nextWork.Div(nextWork, den)
	if nextWork.Sign() <= 0 {
		nextWork.SetInt64(1)
	}

	nextTarget := mainnetTargetForWork(nextWork)
	if nextTarget.Cmp(powLimit) > 0 {
		nextTarget.Set(powLimit)
	}
	if nextTarget.Sign() <= 0 || nextTarget.BitLen() > 256 {
		return nil, ErrInvalidTarget
	}
	return nextTarget, nil
}

func ValidateMainnetTimestampCandidate(
	history []MainnetDAAHeader,
	candidateTimestamp int64,
	localSystemTime int64,
) error {
	if len(history) == 0 {
		return ErrMainnetDAAHistory
	}
	count := MainnetDAAMTPWindow
	if len(history) < count {
		count = len(history)
	}
	values := make([]int64, 0, count)
	for _, h := range history[len(history)-count:] {
		values = append(values, h.Timestamp)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	median := values[len(values)/2]

	if candidateTimestamp <= median {
		return fmt.Errorf("%w: candidate=%d median=%d", ErrMainnetDAATimestamp, candidateTimestamp, median)
	}
	if candidateTimestamp > localSystemTime+MainnetDAAMaxFutureSeconds {
		return fmt.Errorf(
			"%w: candidate=%d max=%d",
			ErrMainnetDAATimestamp,
			candidateTimestamp,
			localSystemTime+MainnetDAAMaxFutureSeconds,
		)
	}
	return nil
}

func validateMainnetDAAHistory(history []MainnetDAAHeader) error {
	for i, h := range history {
		if h.Target == nil || h.Target.Sign() <= 0 || h.Target.BitLen() > 256 {
			return fmt.Errorf("%w at index %d: target", ErrMainnetDAAHistory, i)
		}
		if i > 0 && h.Height != history[i-1].Height+1 {
			return fmt.Errorf("%w at index %d: height", ErrMainnetDAAHistory, i)
		}
	}
	return nil
}

func mainnetWorkForTarget(target *big.Int) *big.Int {
	if target == nil || target.Sign() <= 0 {
		return big.NewInt(0)
	}
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	return two256.Div(two256, new(big.Int).Add(new(big.Int).Set(target), big.NewInt(1)))
}

// Consensus-candidate inverse of work(target)=floor(2^256/(target+1)).
// Exact inverse semantics must be retained if M4 is frozen.
func mainnetTargetForWork(work *big.Int) *big.Int {
	if work == nil || work.Sign() <= 0 {
		return big.NewInt(0)
	}
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	target := two256.Div(two256, work)
	target.Sub(target, big.NewInt(1))
	return target
}
