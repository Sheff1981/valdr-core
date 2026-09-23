// Package logging provides the structured category prefixes required by
// the VALDR v0.1 master specification.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const (
	CategoryNode    = "NODE"
	CategoryP2P     = "P2P"
	CategoryBlock   = "BLOCK"
	CategoryTX      = "TX"
	CategoryMiner   = "MINER"
	CategoryMempool = "MEMPOOL"
	CategorySync    = "SYNC"
	CategoryError   = "ERROR"
)

var (
	mu      sync.RWMutex
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags|log.LUTC)
)

func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	if w == nil {
		w = io.Discard
	}
	defaultLogger = log.New(w, "", log.LstdFlags|log.LUTC)
}

func Printf(category, format string, args ...any) {
	mu.RLock()
	logger := defaultLogger
	mu.RUnlock()
	logger.Printf("[%s] %s", category, fmt.Sprintf(format, args...))
}
