package consensus

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/block"
)

func lowIntegratedPoWHash() []byte {
	out := make([]byte, 32)
	out[31] = 1
	return out
}

func integratedHistory(count int, interval int64, target *big.Int) []MainnetDAAHeader {
	h := make([]MainnetDAAHeader, 0, count)
	ts := int64(1800000000)
	for i := 0; i < count; i++ {
		h = append(h, MainnetDAAHeader{
			Height:    uint64(i),
			Timestamp: ts,
			Target:    new(big.Int).Set(target),
		})
		ts += interval
	}
	return h
}

func buildIntegratedCandidate(
	t *testing.T,
	height uint64,
	parent string,
	timestamp int64,
	target *big.Int,
) *block.Block {
	t.Helper()
	targetHex, err := TargetHexV2(target)
	if err != nil {
		t.Fatal(err)
	}
	b, err := block.NewV2(
		height,
		parent,
		timestamp,
		targetHex,
		7,
		nil,
		MainnetCandidateChainID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	b.BlockHash = b.CalculateHash()
	return b
}

func TestValidateMainnetConsensusCandidateM3M4Integration(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := integratedHistory(31, 600, initial)

	expected, err := MainnetNextTargetLWMA(history, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	parent := strings.Repeat("11", 32)
	candidateTimestamp := history[len(history)-1].Timestamp + 600
	candidate := buildIntegratedCandidate(
		t,
		history[len(history)-1].Height+1,
		parent,
		candidateTimestamp,
		expected,
	)

	h := &fixedRandomXHasher{hash: lowIntegratedPoWHash()}
	got, err := ValidateMainnetConsensusCandidate(
		candidate,
		history,
		parent,
		initial,
		powLimit,
		candidateTimestamp,
		nil,
		h,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("pow hash length=%d want=32", len(got))
	}
}

func TestValidateMainnetConsensusCandidateRejectsWrongTarget(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := integratedHistory(31, 600, initial)
	parent := strings.Repeat("22", 32)
	candidateTimestamp := history[len(history)-1].Timestamp + 600
	candidate := buildIntegratedCandidate(
		t,
		history[len(history)-1].Height+1,
		parent,
		candidateTimestamp,
		initial,
	)

	h := &fixedRandomXHasher{hash: lowIntegratedPoWHash()}
	_, err := ValidateMainnetConsensusCandidate(
		candidate,
		history,
		parent,
		initial,
		powLimit,
		candidateTimestamp,
		nil,
		h,
	)
	if !errors.Is(err, ErrMainnetCandidateTarget) {
		t.Fatalf("error=%v want ErrMainnetCandidateTarget", err)
	}
}

func TestValidateMainnetConsensusCandidateRejectsMTPAndFuture(t *testing.T) {
	initial := daaTarget(t, "0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	powLimit := daaTarget(t, "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	history := integratedHistory(11, 600, initial)
	expected, err := MainnetNextTargetLWMA(history, initial, powLimit)
	if err != nil {
		t.Fatal(err)
	}
	parent := strings.Repeat("33", 32)
	local := history[len(history)-1].Timestamp

	mtpCandidate := buildIntegratedCandidate(
		t,
		history[len(history)-1].Height+1,
		parent,
		history[5].Timestamp,
		expected,
	)
	h := &fixedRandomXHasher{hash: lowIntegratedPoWHash()}
	if _, err := ValidateMainnetConsensusCandidate(
		mtpCandidate, history, parent, initial, powLimit, local, nil, h,
	); !errors.Is(err, ErrMainnetDAATimestamp) {
		t.Fatalf("MTP error=%v", err)
	}

	futureCandidate := buildIntegratedCandidate(
		t,
		history[len(history)-1].Height+1,
		parent,
		local+MainnetDAAMaxFutureSeconds+1,
		expected,
	)
	if _, err := ValidateMainnetConsensusCandidate(
		futureCandidate, history, parent, initial, powLimit, local, nil, h,
	); !errors.Is(err, ErrMainnetDAATimestamp) {
		t.Fatalf("future error=%v", err)
	}
}
