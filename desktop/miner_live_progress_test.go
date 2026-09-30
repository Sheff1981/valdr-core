package desktop

import (
	"math"
	"testing"

	"github.com/Sheff1981/valdr-core/rpc"
)

func TestMergeLiveMiningProgressShowsWorkBeforeFirstBlock(t *testing.T) {
	status := MinerStatus{Running: true}
	info := rpc.MiningInfoResult{
		MiningActive:      true,
		MiningHashes:      250_000,
		MiningElapsedMS:   1000,
		MiningHashrateHPS: 250_000,
	}

	mergeLiveMiningProgress(&status, info)

	if status.TotalHashes != 250_000 {
		t.Fatalf("total hashes=%d want=250000", status.TotalHashes)
	}
	if math.Abs(status.HashrateHPS-250_000) > 0.001 {
		t.Fatalf("hashrate=%f want=250000", status.HashrateHPS)
	}
	if status.AcceptedBlocks != 0 {
		t.Fatalf("accepted blocks=%d want=0", status.AcceptedBlocks)
	}
}

func TestMergeLiveMiningProgressIncludesCompletedAndInFlightWork(t *testing.T) {
	status := MinerStatus{
		Running:               true,
		TotalHashes:           500_000,
		TotalMiningDurationMS: 2000,
	}
	info := rpc.MiningInfoResult{
		MiningActive:    true,
		MiningHashes:    250_000,
		MiningElapsedMS: 1000,
	}

	mergeLiveMiningProgress(&status, info)

	if status.TotalHashes != 750_000 {
		t.Fatalf("total hashes=%d want=750000", status.TotalHashes)
	}
	if math.Abs(status.HashrateHPS-250_000) > 0.001 {
		t.Fatalf("hashrate=%f want=250000", status.HashrateHPS)
	}
}
