package rpc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	"github.com/Sheff1981/valdr-core/p2p"
)

func newRPCSecurityTestServer(t *testing.T) *Server {
	t.Helper()
	chain := blockchain.New()
	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:        "rpc-security-node",
		ListenAddress: "127.0.0.1:0",
		Blockchain:    chain,
		Mempool:       mempool.New(),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(chain, node)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestRPCRejectsBrowserOriginRequests(t *testing.T) {
	server := newRPCSecurityTestServer(t)
	body := []byte(`{"method":"getStatus"}`)
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://attacker.example")

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "browser-origin RPC requests are not allowed") {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
}

func TestRPCRejectsCORSSafelistedContentTypes(t *testing.T) {
	server := newRPCSecurityTestServer(t)
	for _, contentType := range []string{
		"text/plain",
		"application/x-www-form-urlencoded",
		"multipart/form-data; boundary=x",
		"",
	} {
		t.Run(contentType, func(t *testing.T) {
			body := []byte(`{"method":"getStatus"}`)
			req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}

			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusUnsupportedMediaType {
				t.Fatalf(
					"content-type=%q status=%d want=%d body=%s",
					contentType,
					rec.Code,
					http.StatusUnsupportedMediaType,
					rec.Body.String(),
				)
			}
		})
	}
}

func TestRPCSecurityHeadersAndJSONMediaTypeParameters(t *testing.T) {
	server := newRPCSecurityTestServer(t)
	body := []byte(`{"method":"getStatus"}`)
	req := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q want no-store", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q want nosniff", got)
	}
}

func TestRPCRejectsOversizeBody(t *testing.T) {
	server := newRPCSecurityTestServer(t)
	payload := `{"method":"getStatus"}` + strings.Repeat(" ", maxRPCRequestBytes)

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
