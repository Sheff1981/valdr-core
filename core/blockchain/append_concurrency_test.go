package blockchain

import (
	"errors"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/core/block"
)

func TestAppendDoesNotHoldChainLockWhileMining(t *testing.T) {
	chain := New()
	tip := chain.Tip()
	if tip == nil {
		t.Fatal("missing genesis tip")
	}

	miningStarted := make(chan struct{})
	releaseMining := make(chan struct{})
	injectedErr := errors.New("stop injected mining")
	done := make(chan error, 1)

	go func() {
		_, err := chain.appendWithMiners(
			tip.Timestamp+1,
			nil,
			nil,
			func(_ *block.Block) error {
				close(miningStarted)
				<-releaseMining
				return injectedErr
			},
		)
		done <- err
	}()

	select {
	case <-miningStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("injected miner did not start")
	}

	readDone := make(chan uint64, 1)
	go func() {
		readDone <- chain.Height()
	}()

	select {
	case height := <-readDone:
		if height != tip.Height {
			t.Fatalf("height=%d want=%d", height, tip.Height)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("chain read blocked while proof-of-work was running")
	}

	close(releaseMining)
	select {
	case err := <-done:
		if !errors.Is(err, injectedErr) {
			t.Fatalf("append error=%v want=%v", err, injectedErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("append did not finish after injected miner release")
	}
}
