package main

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
)

func main() {
	hashrate := flag.String("hashrate-hs", "", "RandomX hashrate in H/s, integer or decimal")
	seconds := flag.Int64("seconds", 600, "target block interval in seconds")
	safetyNum := flag.Int64("safety-num", 1, "explicit conservative multiplier numerator")
	safetyDen := flag.Int64("safety-den", 1, "explicit conservative multiplier denominator")
	flag.Parse()

	target, work, err := calibrationTarget(*hashrate, *seconds, *safetyNum, *safetyDen)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	fmt.Printf("hashrate_hs=%s\n", *hashrate)
	fmt.Printf("target_seconds=%d\n", *seconds)
	fmt.Printf("safety_multiplier=%d/%d\n", *safetyNum, *safetyDen)
	fmt.Printf("initial_work=%s\n", work.String())
	fmt.Printf("initial_target=%064x\n", target)
	fmt.Println("status=CALIBRATION_CANDIDATE_ONLY")
	fmt.Println("note=PowLimit remains unresolved and Mainnet remains disabled")
}

func calibrationTarget(hashrate string, seconds, safetyNum, safetyDen int64) (*big.Int, *big.Int, error) {
	if seconds <= 0 || safetyNum <= 0 || safetyDen <= 0 {
		return nil, nil, errors.New("seconds and safety multiplier must be positive")
	}
	rate, ok := new(big.Rat).SetString(strings.TrimSpace(hashrate))
	if !ok || rate.Sign() <= 0 {
		return nil, nil, errors.New("hashrate-hs must be a positive integer or decimal")
	}

	workRat := new(big.Rat).Mul(rate, new(big.Rat).SetInt64(seconds))
	workRat.Mul(workRat, new(big.Rat).SetFrac(big.NewInt(safetyNum), big.NewInt(safetyDen)))

	work := new(big.Int).Quo(workRat.Num(), workRat.Denom())
	if work.Sign() <= 0 {
		return nil, nil, errors.New("derived work is zero")
	}

	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	target := new(big.Int).Div(two256, work)
	target.Sub(target, big.NewInt(1))
	if target.Sign() <= 0 || target.BitLen() > 256 {
		return nil, nil, errors.New("derived target is outside uint256 range")
	}
	return target, work, nil
}
