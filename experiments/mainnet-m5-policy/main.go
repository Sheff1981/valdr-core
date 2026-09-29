package main

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
)

type Candidate struct {
	ReferenceHashrate *big.Rat
	TargetSeconds     int64
	SafetyNum         int64
	SafetyDen         int64
	PowLimitFactor    int64
	InitialWork       *big.Int
	InitialTarget     *big.Int
	PowLimitWork      *big.Int
	PowLimitTarget    *big.Int
}

func main() {
	h := flag.String("hashrate-hs", "418.723", "reference measured all-thread RandomX H/s")
	seconds := flag.Int64("seconds", 600, "target block interval")
	sn := flag.Int64("safety-num", 1, "InitialWork multiplier numerator")
	sd := flag.Int64("safety-den", 1, "InitialWork multiplier denominator")
	pf := flag.Int64("powlimit-factor", 4, "PowLimit is this factor easier than InitialTarget in work terms")
	flag.Parse()

	c, err := buildCandidate(*h, *seconds, *sn, *sd, *pf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	fmt.Printf("reference_hashrate_hs=%s\n", c.ReferenceHashrate.FloatString(3))
	fmt.Printf("target_seconds=%d\n", c.TargetSeconds)
	fmt.Printf("safety_multiplier=%d/%d\n", c.SafetyNum, c.SafetyDen)
	fmt.Printf("powlimit_easier_factor=%d\n", c.PowLimitFactor)
	fmt.Printf("initial_work=%s\n", c.InitialWork.String())
	fmt.Printf("initial_target=%064x\n", c.InitialTarget)
	fmt.Printf("powlimit_work=%s\n", c.PowLimitWork.String())
	fmt.Printf("powlimit_target=%064x\n", c.PowLimitTarget)
	fmt.Printf("expected_seconds_at_reference_initial=%s\n", expectedSeconds(c.InitialWork, c.ReferenceHashrate).FloatString(3))
	fmt.Printf("expected_seconds_at_reference_powlimit=%s\n", expectedSeconds(c.PowLimitWork, c.ReferenceHashrate).FloatString(3))
	fmt.Println("status=M5_POLICY_CANDIDATE_ONLY")
	fmt.Println("note=No value is frozen until DAA simulations and independent reproduction pass.")
}

func buildCandidate(hashrate string, seconds, safetyNum, safetyDen, powLimitFactor int64) (*Candidate, error) {
	if seconds <= 0 || safetyNum <= 0 || safetyDen <= 0 || powLimitFactor < 1 {
		return nil, errors.New("seconds, safety ratio and powlimit factor must be positive")
	}
	rate, ok := new(big.Rat).SetString(strings.TrimSpace(hashrate))
	if !ok || rate.Sign() <= 0 {
		return nil, errors.New("hashrate-hs must be a positive decimal or integer")
	}

	workRat := new(big.Rat).Mul(rate, new(big.Rat).SetInt64(seconds))
	workRat.Mul(workRat, new(big.Rat).SetFrac(big.NewInt(safetyNum), big.NewInt(safetyDen)))
	initialWork := new(big.Int).Quo(workRat.Num(), workRat.Denom())
	if initialWork.Sign() <= 0 {
		return nil, errors.New("derived InitialWork is zero")
	}

	powWork := new(big.Int).Quo(new(big.Int).Set(initialWork), big.NewInt(powLimitFactor))
	if powWork.Sign() <= 0 {
		return nil, errors.New("derived PowLimit work is zero")
	}

	initialTarget, err := inverseWork(initialWork)
	if err != nil {
		return nil, err
	}
	powTarget, err := inverseWork(powWork)
	if err != nil {
		return nil, err
	}
	if powTarget.Cmp(initialTarget) < 0 {
		return nil, errors.New("PowLimit target must not be harder than InitialTarget")
	}

	return &Candidate{
		ReferenceHashrate: new(big.Rat).Set(rate),
		TargetSeconds: seconds,
		SafetyNum: safetyNum,
		SafetyDen: safetyDen,
		PowLimitFactor: powLimitFactor,
		InitialWork: initialWork,
		InitialTarget: initialTarget,
		PowLimitWork: powWork,
		PowLimitTarget: powTarget,
	}, nil
}

func inverseWork(work *big.Int) (*big.Int, error) {
	if work == nil || work.Sign() <= 0 {
		return nil, errors.New("work must be positive")
	}
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	target := new(big.Int).Div(two256, work)
	target.Sub(target, big.NewInt(1))
	if target.Sign() <= 0 || target.BitLen() > 256 {
		return nil, errors.New("target outside uint256 range")
	}
	return target, nil
}

func expectedSeconds(work *big.Int, hashrate *big.Rat) *big.Rat {
	return new(big.Rat).Quo(new(big.Rat).SetInt(work), hashrate)
}
