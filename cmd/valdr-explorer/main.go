package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/explorer"
	"github.com/Sheff1981/valdr-core/rpc"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "Explorer HTTP listen address")
	node := flag.String("node", "http://127.0.0.1:7332", "VALDR read-only RPC endpoint")
	indexFile := flag.String("index-file", defaultIndexFile(), "persistent Explorer index file")
	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("%s %s valdr-explorer %s\n", config.ProjectName, config.Ticker, config.Version)
		return
	}

	server, err := explorer.NewWithIndex(rpc.NewClient(*node), *indexFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	httpServer := &http.Server{
		Addr: *listen, Handler: server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	fmt.Fprintf(os.Stderr, "VALDR Explorer listening on http://%s using node %s\n", *listen, *node)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func defaultIndexFile() string {
	if override := os.Getenv("VALDR_EXPLORER_INDEX_FILE"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil { return "" }
	return filepath.Join(home, ".valdr", "explorer", "index.json")
}
