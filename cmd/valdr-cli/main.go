package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/wallet"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "version" {
		fmt.Fprintf(
			out,
			"%s %s valdr-cli %s\n",
			config.ProjectName,
			config.Ticker,
			config.Version,
		)
		return 0
	}
	if args[0] != "wallet" {
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		return 2
	}
	return runWallet(args[1:], out, errOut)
}

func runWallet(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli wallet <create|list|export|balance>")
		return 2
	}

	switch args[0] {
	case "create":
		return walletCreate(args[1:], out, errOut)
	case "list":
		return walletList(args[1:], out, errOut)
	case "export":
		return walletExport(args[1:], out, errOut)
	case "balance":
		fmt.Fprintln(
			errOut,
			"wallet balance requires the UTXO Engine scheduled for Day 6",
		)
		return 2
	default:
		fmt.Fprintf(errOut, "unknown wallet command %q\n", args[0])
		return 2
	}
}

func walletCreate(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("wallet create", flag.ContinueOnError)
	fs.SetOutput(errOut)

	dir := fs.String("dir", "", "wallet directory")
	name := fs.String("name", "", "wallet name")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	w, err := store.Create(*name)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, w.Metadata(), errOut)
}

func walletList(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("wallet list", flag.ContinueOnError)
	fs.SetOutput(errOut)

	dir := fs.String("dir", "", "wallet directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	items, err := store.List()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, items, errOut)
}

func walletExport(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("wallet export", flag.ContinueOnError)
	fs.SetOutput(errOut)

	dir := fs.String("dir", "", "wallet directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(
			errOut,
			"usage: valdr-cli wallet export [--dir PATH] <name|address>",
		)
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	w, err := store.Export(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	fmt.Fprintln(
		errOut,
		"WARNING: export reveals the private key; keep it offline and secret",
	)
	return writeJSON(out, w, errOut)
}

func walletStore(dir string, errOut io.Writer) (*wallet.Store, int) {
	if dir == "" {
		resolved, err := wallet.DefaultDir()
		if err != nil {
			fmt.Fprintln(errOut, err)
			return nil, 1
		}
		dir = resolved
	}
	return wallet.NewStore(dir), 0
}

func writeJSON(out io.Writer, value any, errOut io.Writer) int {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}
