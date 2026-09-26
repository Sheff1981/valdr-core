package p2p

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/core/transaction"
	"github.com/Sheff1981/valdr-core/logging"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestP2PWireAndTransactionLogsExcludeWalletSecrets(t *testing.T) {
	passphraseText := "VALDR_P2P_SECRET_PASSPHRASE_62d8953a"
	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	source, err := store.CreateEncrypted(
		"p2p-secret-source",
		[]byte(passphraseText),
	)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("p2p-secret-recipient")
	if err != nil {
		t.Fatal(err)
	}
	privateKey := source.PrivateKey
	if privateKey == "" {
		t.Fatal("source private key unexpectedly empty")
	}

	chain := blockchain.New()
	block, err := mining.MineBlock(
		chain,
		source.Address,
		config.GenesisTimestamp+60,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if block.Height != 1 {
		t.Fatalf("mined height=%d want=1", block.Height)
	}
	available, err := chain.UTXOs(source.Address)
	if err != nil {
		t.Fatal(err)
	}

	legacyTx, err := source.CreateTransaction(
		available,
		recipient.Address,
		config.AtomicUnitsPerVDR,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}

	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV029)
	if err != nil {
		t.Fatal(err)
	}
	v2Tx, _, err := source.CreateTransactionForChain(
		profile.ChainID,
		available,
		recipient.Address,
		config.AtomicUnitsPerVDR,
		0,
		config.GenesisTimestamp+90,
	)
	if err != nil {
		t.Fatal(err)
	}
	if v2Tx.Version != transaction.VersionV2 {
		t.Fatalf("P2P secret-boundary transaction version=%d", v2Tx.Version)
	}

	var frame bytes.Buffer
	if err := WriteV2Frame(
		&frame,
		profile,
		uint16(profile.ProtocolMax),
		V2MessageTx,
		V2TxPayload{Transaction: v2Tx},
	); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadV2Frame(bytes.NewReader(frame.Bytes()), profile)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.MessageType != V2MessageTx {
		t.Fatalf("P2P frame type=%d want=%d", decoded.MessageType, V2MessageTx)
	}

	for _, forbidden := range []string{
		passphraseText,
		privateKey,
		"private_key",
		"passphrase",
	} {
		if bytes.Contains(frame.Bytes(), []byte(forbidden)) {
			t.Fatalf("P2P wire contains forbidden wallet secret marker %q", forbidden)
		}
	}
	for _, required := range []string{
		`"transaction"`,
		`"public_key"`,
		`"signature"`,
		`"transaction_id"`,
		v2Tx.PublicKey,
		v2Tx.Signature,
		v2Tx.TransactionID,
	} {
		if !bytes.Contains(decoded.Payload, []byte(required)) {
			t.Fatalf("P2P tx payload missing expected public field %q", required)
		}
	}

	node, err := NewNode(NodeConfig{
		NodeID:        "secret-log-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       mempool.New(),
	})
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	logging.SetOutput(&logs)
	t.Cleanup(func() { logging.SetOutput(os.Stderr) })

	if err := node.BroadcastTransaction(legacyTx); err != nil {
		t.Fatal(err)
	}
	logText := logs.String()
	if !strings.Contains(logText, legacyTx.TransactionID) {
		t.Fatalf("transaction log does not contain expected txid: %q", logText)
	}
	for _, forbidden := range []string{
		passphraseText,
		privateKey,
		source.PublicKey,
		legacyTx.Signature,
	} {
		if strings.Contains(logText, forbidden) {
			t.Fatalf(
				"transaction log contains forbidden secret/payload marker %q",
				forbidden,
			)
		}
	}
}
