package main

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/Sheff1981/valdr-core/core/consensus"
)

const (
	initialTargetHex = "000042c78dcd2f09f6673e99417b7b7d0dbb0ddcf732f964dc73fce1c6b50768"
	referenceRateMilli = int64(418723) // 418.723 H/s
)

type scenario struct {
	name string
	rateNum int64
	rateDen int64
}

type result struct {
	powFactor int64
	scenario string
	firstSolve int64
	last30Avg *big.Rat
	maxSolve int64
	powLimitHits int
	finalWork *big.Int
	finalTarget *big.Int
}

func main() {
	factors := []int64{2, 4, 8}
	scenarios := []scenario{
		{"stable", 1, 1},
		{"2x_up", 2, 1},
		{"10x_up", 10, 1},
		{"100x_up", 100, 1},
		{"2x_down", 1, 2},
		{"10x_down", 1, 10},
		{"100x_down", 1, 100},
	}
	fmt.Println("VALDR_M5_DAA_SIM_V1")
	fmt.Println("model=deterministic_expected_solvetime_rounded_to_nearest_second")
	fmt.Println("blocks=120")
	fmt.Println("reference_hashrate_hs=418.723")
	fmt.Println("initial_target=" + initialTargetHex)
	fmt.Println()
	for _, factor := range factors {
		for _, s := range scenarios {
			r, err := simulate(factor, s, 120)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(2)
			}
			fmt.Printf(
				"powlimit=%dx scenario=%s first_s=%d last30_avg_s=%s max_s=%d cap_hits=%d final_work=%s final_target=%064x\n",
				factor, s.name, r.firstSolve, r.last30Avg.FloatString(3), r.maxSolve,
				r.powLimitHits, r.finalWork.String(), r.finalTarget,
			)
		}
	}
	fmt.Println()
	fmt.Println("candidate_rule=minimal PowLimit easing that preserves DAA headroom after a 2x hashrate decrease")
	fmt.Println("candidate_powlimit_factor=4")
	fmt.Println("status=M5_DAA_SIM_CANDIDATE_ONLY")
}

func simulate(powFactor int64, s scenario, blocks int) (*result, error) {
	initialTarget, ok := new(big.Int).SetString(initialTargetHex, 16)
	if !ok {
		return nil, fmt.Errorf("bad initial target")
	}
	initialWork := workForTarget(initialTarget)
	if initialWork.Sign() <= 0 {
		return nil, fmt.Errorf("bad initial work")
	}
	if powFactor < 1 {
		return nil, fmt.Errorf("bad pow factor")
	}
	powWork := new(big.Int).Quo(new(big.Int).Set(initialWork), big.NewInt(powFactor))
	if powWork.Sign() <= 0 {
		return nil, fmt.Errorf("pow work zero")
	}
	powLimit := targetForWork(powWork)

	rate := new(big.Rat).SetFrac(
		new(big.Int).Mul(big.NewInt(referenceRateMilli), big.NewInt(s.rateNum)),
		new(big.Int).Mul(big.NewInt(1000), big.NewInt(s.rateDen)),
	)
	if rate.Sign() <= 0 {
		return nil, fmt.Errorf("bad rate")
	}

	history := []consensus.MainnetDAAHeader{{
		Height: 0, Timestamp: 1800000000, Target: new(big.Int).Set(initialTarget),
	}}
	current := new(big.Int).Set(initialTarget)
	solves := make([]int64, 0, blocks)
	capHits := 0

	for height := 1; height <= blocks; height++ {
		work := workForTarget(current)
		solve := roundedSeconds(work, rate)
		if solve < 1 {
			solve = 1
		}
		solves = append(solves, solve)
		ts := history[len(history)-1].Timestamp + solve
		history = append(history, consensus.MainnetDAAHeader{
			Height: uint64(height),
			Timestamp: ts,
			Target: new(big.Int).Set(current),
		})
		next, err := consensus.MainnetNextTargetLWMA(history, initialTarget, powLimit)
		if err != nil {
			return nil, fmt.Errorf("height %d: %w", height, err)
		}
		current = next
		if current.Cmp(powLimit) == 0 {
			capHits++
		}
	}

	last := solves
	if len(last) > 30 {
		last = last[len(last)-30:]
	}
	sum := int64(0)
	maxSolve := int64(0)
	for _, v := range solves {
		if v > maxSolve {
			maxSolve = v
		}
	}
	for _, v := range last {
		sum += v
	}
	avg := new(big.Rat).SetFrac(big.NewInt(sum), big.NewInt(int64(len(last))))

	return &result{
		powFactor: powFactor,
		scenario: s.name,
		firstSolve: solves[0],
		last30Avg: avg,
		maxSolve: maxSolve,
		powLimitHits: capHits,
		finalWork: workForTarget(current),
		finalTarget: current,
	}, nil
}

func roundedSeconds(work *big.Int, rate *big.Rat) int64 {
	x := new(big.Rat).Quo(new(big.Rat).SetInt(work), rate)
	n := new(big.Int).Mul(x.Num(), big.NewInt(2))
	n.Add(n, x.Denom())
	d := new(big.Int).Mul(x.Denom(), big.NewInt(2))
	n.Quo(n, d)
	if !n.IsInt64() {
		panic("solve seconds overflow")
	}
	return n.Int64()
}

func workForTarget(target *big.Int) *big.Int {
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	return two256.Div(two256, new(big.Int).Add(new(big.Int).Set(target), big.NewInt(1)))
}

func targetForWork(work *big.Int) *big.Int {
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	out := two256.Div(two256, work)
	return out.Sub(out, big.NewInt(1))
}

func mustInt(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil { panic(err) }
	return v
}
