package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
	"github.com/Sheff1981/valdr-core/wallet"
)

func TestDesktopRPCWireExcludesWalletSecrets(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkTestnetV02)
	if err != nil {
		t.Fatal(err)
	}

	store := wallet.NewStore(filepath.Join(t.TempDir(), "wallets"))
	passphraseText := "VALDR_RPC_SECRET_PASSPHRASE_9c54d42e"
	passphrase := []byte(passphraseText)
	source, err := store.CreateEncrypted("rpc-secret-source", passphrase)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.New("rpc-secret-recipient")
	if err != nil {
		t.Fatal(err)
	}
	privateKey := source.PrivateKey
	if privateKey == "" {
		t.Fatal("source private key unexpectedly empty")
	}

	var mu sync.Mutex
	var wire bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mu.Lock()
		wire.Write(raw)
		wire.WriteByte('\n')
		mu.Unlock()

		var request struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var result any
		switch request.Method {
		case rpc.MethodGetStatus:
			result = rpc.StatusResult{
				Network: profile.Name,
				ChainID: profile.ChainID,
			}
		case rpc.MethodGetUTXOs:
			result = []utxo.UTXO{{
				TransactionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				OutputIndex:   0,
				Amount:        2_000_000,
				Recipient:     source.Address,
			}}
		case rpc.MethodSendTransaction:
			var params rpc.SendTransactionParams
			if err := json.Unmarshal(request.Params, &params); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if params.Transaction == nil {
				t.Error("RPC sendTransaction has nil transaction")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			result = rpc.SendTransactionResult{
				TransactionID: params.Transaction.TransactionID,
			}
		default:
			t.Errorf("unexpected RPC method %q", request.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Result any `json:"result"`
		}{Result: result}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	service, err := NewWalletService(store, rpc.NewClient(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	sent, err := service.Send(
		context.Background(),
		source.Address,
		passphrase,
		recipient.Address,
		1_000_000,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sent.TransactionID == "" {
		t.Fatal("Desktop RPC send returned empty txid")
	}

	mu.Lock()
	rawWire := append([]byte(nil), wire.Bytes()...)
	mu.Unlock()

	for _, forbidden := range []string{
		passphraseText,
		privateKey,
		"private_key",
		"passphrase",
	} {
		if bytes.Contains(rawWire, []byte(forbidden)) {
			t.Fatalf(
				"Desktop RPC wire contains forbidden wallet secret marker %q",
				forbidden,
			)
		}
	}
	for _, required := range []string{
		rpc.MethodSendTransaction,
		`"public_key"`,
		`"signature"`,
		`"transaction_id"`,
		source.PublicKey,
	} {
		if !bytes.Contains(rawWire, []byte(required)) {
			t.Fatalf(
				"Desktop RPC wire missing expected public transaction field %q",
				required,
			)
		}
	}
}
