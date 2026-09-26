package rpc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/p2p"
)

func FuzzRPCRequestDecoder(f *testing.F) {
	chain := blockchain.New()
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "rpc-fuzz-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       mempool.New(),
	})
	if err != nil {
		f.Fatal(err)
	}
	server, err := NewServer(chain, node)
	if err != nil {
		f.Fatal(err)
	}
	handler := server.Handler()

	f.Add([]byte(`{"method":"getStatus"}`))
	f.Add([]byte(`{"method":"getBlock","params":{"height":0}}`))
	f.Add([]byte(`{"method":"getStatus","unexpected":true}`))
	f.Add([]byte("{"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, body []byte) {
		req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code < 200 || rec.Code > 599 {
			t.Fatalf("invalid HTTP status %d", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type %q", got)
		}
	})
}
