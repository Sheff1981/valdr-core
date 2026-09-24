package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/block"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
	"golang.org/x/term"
)

const defaultRPCEndpoint = "http://127.0.0.1:7332"

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

	switch args[0] {
	case "status":
		return statusCommand(args[1:], out, errOut)
	case "block":
		return blockCommand(args[1:], out, errOut)
	case "tx":
		return transactionCommand(args[1:], out, errOut)
	case "balance":
		return balanceCommand(args[1:], out, errOut)
	case "send":
		return sendCommand(args[1:], out, errOut)
	case "peers":
		return peersCommand(args[1:], out, errOut)
	case "mempool":
		return mempoolCommand(args[1:], out, errOut)
	case "mining":
		return miningCommand(args[1:], out, errOut)
	case "wallet":
		return runWallet(args[1:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		return 2
	}
}

func statusCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli status [--node URL]")
		return 2
	}

	var result rpc.StatusResult
	if err := rpcCall(*node, rpc.MethodGetStatus, nil, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func blockCommand(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "get" {
		fmt.Fprintln(errOut, "usage: valdr-cli block get [--node URL] <height>")
		return 2
	}

	fs := flag.NewFlagSet("block get", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: valdr-cli block get [--node URL] <height>")
		return 2
	}

	height, err := strconv.ParseUint(fs.Arg(0), 10, 64)
	if err != nil {
		fmt.Fprintln(errOut, "invalid block height")
		return 2
	}

	var result block.Block
	if err := rpcCall(*node, rpc.MethodGetBlock, rpc.HeightParams{Height: height}, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func transactionCommand(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "get" {
		fmt.Fprintln(errOut, "usage: valdr-cli tx get [--node URL] <txid>")
		return 2
	}

	fs := flag.NewFlagSet("tx get", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: valdr-cli tx get [--node URL] <txid>")
		return 2
	}

	var result rpc.TransactionResult
	if err := rpcCall(
		*node,
		rpc.MethodGetTransaction,
		rpc.TransactionParams{TransactionID: fs.Arg(0)},
		&result,
	); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func balanceCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("balance", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errOut, "usage: valdr-cli balance [--node URL] <address>")
		return 2
	}

	var result rpc.BalanceResult
	if err := rpcCall(
		*node,
		rpc.MethodGetBalance,
		rpc.AddressParams{Address: fs.Arg(0)},
		&result,
	); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	output := struct {
		Address    string `json:"address"`
		BalanceVal uint64 `json:"balance_val"`
		BalanceVDR string `json:"balance_vdr"`
	}{
		Address:    result.Address,
		BalanceVal: result.BalanceVal,
		BalanceVDR: formatVDR(result.BalanceVal),
	}
	return writeJSON(out, output, errOut)
}

func sendCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(errOut)

	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	walletDir := fs.String("wallet-dir", "", "wallet directory")
	from := fs.String("from", "", "wallet name or address")
	to := fs.String("to", "", "recipient VDR address")
	amountText := fs.String("amount", "", "amount in VDR")
	passwordFD := fs.Int("password-fd", -1, "read wallet passphrase from protected file descriptor")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 || *from == "" || *to == "" || *amountText == "" {
		fmt.Fprintln(
			errOut,
			"usage: valdr-cli send [--node URL] [--wallet-dir PATH] [--password-fd FD] --from WALLET --to ADDRESS --amount VDR",
		)
		return 2
	}

	amount, err := parseVDR(*amountText)
	if err != nil || amount == 0 {
		fmt.Fprintln(errOut, "invalid VDR amount")
		return 2
	}

	store, code := walletStore(*walletDir, errOut)
	if code != 0 {
		return code
	}
	passphrase, err := readWalletPassphrase(*passwordFD, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer clearBytes(passphrase)

	source, err := store.Unlock(*from, passphrase)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	var available []utxo.UTXO
	if err := rpcCall(
		*node,
		rpc.MethodGetUTXOs,
		rpc.AddressParams{Address: source.Address},
		&available,
	); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	tx, err := source.CreateTransaction(
		available,
		*to,
		amount,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}

	var result rpc.SendTransactionResult
	if err := rpcCall(
		*node,
		rpc.MethodSendTransaction,
		rpc.SendTransactionParams{Transaction: tx},
		&result,
	); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func peersCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("peers", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli peers [--node URL]")
		return 2
	}

	var result []p2p.Peer
	if err := rpcCall(*node, rpc.MethodGetPeers, nil, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func mempoolCommand(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("mempool", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli mempool [--node URL]")
		return 2
	}

	var result []*transaction.Transaction
	if err := rpcCall(*node, rpc.MethodGetMempool, nil, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func miningCommand(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "info" {
		fmt.Fprintln(errOut, "usage: valdr-cli mining info [--node URL]")
		return 2
	}

	fs := flag.NewFlagSet("mining info", flag.ContinueOnError)
	fs.SetOutput(errOut)
	node := fs.String("node", defaultRPCEndpoint, "VALDR RPC endpoint")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli mining info [--node URL]")
		return 2
	}

	var result rpc.MiningInfoResult
	if err := rpcCall(*node, rpc.MethodGetMiningInfo, nil, &result); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, result, errOut)
}

func runWallet(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: valdr-cli wallet <create|list|export|migrate>")
		return 2
	}

	switch args[0] {
	case "create":
		return walletCreate(args[1:], out, errOut)
	case "list":
		return walletList(args[1:], out, errOut)
	case "export":
		return walletExport(args[1:], out, errOut)
	case "migrate":
		return walletMigrate(args[1:], out, errOut)
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
	passwordFD := fs.Int("password-fd", -1, "read wallet passphrase from protected file descriptor")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	passphrase, err := readWalletPassphrase(*passwordFD, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer clearBytes(passphrase)

	w, err := store.CreateEncrypted(*name, passphrase)
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
	passwordFD := fs.Int("password-fd", -1, "read wallet passphrase from protected file descriptor")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(
			errOut,
			"usage: valdr-cli wallet export [--dir PATH] [--password-fd FD] <name|address>",
		)
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	passphrase, err := readWalletPassphrase(*passwordFD, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer clearBytes(passphrase)

	w, err := store.Export(fs.Arg(0), passphrase)
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

func walletMigrate(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("wallet migrate", flag.ContinueOnError)
	fs.SetOutput(errOut)

	dir := fs.String("dir", "", "wallet directory")
	passwordFD := fs.Int("password-fd", -1, "read new wallet passphrase from protected file descriptor")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(
			errOut,
			"usage: valdr-cli wallet migrate [--dir PATH] [--password-fd FD] <name|address>",
		)
		return 2
	}

	store, code := walletStore(*dir, errOut)
	if code != 0 {
		return code
	}
	passphrase, err := readWalletPassphrase(*passwordFD, errOut)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer clearBytes(passphrase)

	meta, err := store.Migrate(fs.Arg(0), passphrase)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return writeJSON(out, meta, errOut)
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

func readWalletPassphrase(passwordFD int, errOut io.Writer) ([]byte, error) {
	if passwordFD >= 0 {
		file := os.NewFile(uintptr(passwordFD), "valdr-wallet-password")
		if file == nil {
			return nil, errors.New("invalid password file descriptor")
		}
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() && info.Mode().Perm()&0o077 != 0 {
			return nil, errors.New("password file descriptor points to an unprotected file")
		}

		reader := bufio.NewReader(io.LimitReader(file, 4097))
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if len(line) > 4096 {
			return nil, errors.New("wallet passphrase is too long")
		}
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			return nil, wallet.ErrPassphraseRequired
		}
		return []byte(line), nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New(
			"wallet passphrase requires a terminal or --password-fd",
		)
	}
	fmt.Fprint(errOut, "Wallet passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(errOut)
	if err != nil {
		return nil, err
	}
	if len(passphrase) == 0 {
		return nil, wallet.ErrPassphraseRequired
	}
	return passphrase, nil
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func rpcCall(endpoint, method string, params, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return rpc.NewClient(endpoint).Call(ctx, method, params, result)
}

func parseVDR(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, errors.New("invalid amount")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid amount")
	}
	if parts[0] == "" {
		parts[0] = "0"
	}

	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if whole > ^uint64(0)/config.AtomicUnitsPerVDR {
		return 0, errors.New("amount overflow")
	}
	total := whole * config.AtomicUnitsPerVDR

	if len(parts) == 2 {
		if len(parts[1]) > 8 {
			return 0, errors.New("too many decimal places")
		}
		fractionText := parts[1] + strings.Repeat("0", 8-len(parts[1]))
		if fractionText != "" {
			fraction, err := strconv.ParseUint(fractionText, 10, 64)
			if err != nil {
				return 0, err
			}
			if ^uint64(0)-total < fraction {
				return 0, errors.New("amount overflow")
			}
			total += fraction
		}
	}
	return total, nil
}

func formatVDR(value uint64) string {
	whole := value / config.AtomicUnitsPerVDR
	fraction := value % config.AtomicUnitsPerVDR
	if fraction == 0 {
		return strconv.FormatUint(whole, 10)
	}
	text := fmt.Sprintf("%d.%08d", whole, fraction)
	return strings.TrimRight(text, "0")
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
