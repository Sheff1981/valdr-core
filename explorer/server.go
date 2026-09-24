package explorer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/core/utxo"
	"github.com/Sheff1981/valdr-core/rpc"
)

const (
	defaultRequestTimeout = 5 * time.Second
	defaultRecentBlocks   = uint64(20)
	maxRecentBlocks       = uint64(100)
)

type Server struct {
	client         RPCClient
	index          *Index
	requestTimeout time.Duration
}

type StatusAPI struct {
	Status                 rpc.StatusResult    `json:"status"`
	Mining                 rpc.MiningInfoResult `json:"mining"`
	Peers                  []rpc.PeerResult    `json:"peers"`
	AverageBlockIntervalS  float64             `json:"average_block_interval_seconds"`
	RecentIntervalsSeconds []int64             `json:"recent_intervals_seconds"`
	IndexHeight            uint64              `json:"index_height"`
	IndexTipHash           string              `json:"index_tip_hash"`
}

type AddressAPI struct {
	Balance    rpc.BalanceResult `json:"balance"`
	UTXOs      []utxo.UTXO       `json:"utxos"`
	History    []AddressActivity `json:"history"`
}

func New(client RPCClient) (*Server, error) {
	return NewWithIndex(client, "")
}

func NewWithIndex(client RPCClient, indexPath string) (*Server, error) {
	if client == nil {
		return nil, ErrInvalidClient
	}
	index, err := NewIndex(client, indexPath)
	if err != nil {
		return nil, err
	}
	return &Server{
		client:         client,
		index:          index,
		requestTimeout: defaultRequestTimeout,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleOverview)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/block/", s.handleBlockView)
	mux.HandleFunc("/tx/", s.handleTransactionView)
	mux.HandleFunc("/address/", s.handleAddressView)
	mux.HandleFunc("/healthz", s.handleHealth)

	mux.HandleFunc("/api/v1/status", s.handleAPIStatus)
	mux.HandleFunc("/api/v1/blocks", s.handleAPIBlocks)
	mux.HandleFunc("/api/v1/block/", s.handleAPIBlock)
	mux.HandleFunc("/api/v1/tx/", s.handleAPITransaction)
	mux.HandleFunc("/api/v1/address/", s.handleAPIAddress)
	mux.HandleFunc("/api/v1/mempool", s.handleAPIMempool)
	return securityHeaders(mux)
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	result, err := s.status(ctx)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) status(ctx context.Context) (StatusAPI, error) {
	if err := s.index.Refresh(ctx); err != nil {
		return StatusAPI{}, err
	}
	var status rpc.StatusResult
	if err := s.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		return StatusAPI{}, err
	}
	var mining rpc.MiningInfoResult
	if err := s.client.Call(ctx, rpc.MethodGetMiningInfo, nil, &mining); err != nil {
		return StatusAPI{}, err
	}
	var peers []rpc.PeerResult
	if err := s.client.Call(ctx, rpc.MethodGetPeers, nil, &peers); err != nil {
		return StatusAPI{}, err
	}
	intervals, average, err := s.blockIntervals(ctx, status.Height, 20)
	if err != nil {
		return StatusAPI{}, err
	}
	indexHeight, indexTip := s.index.State()
	return StatusAPI{
		Status: status, Mining: mining, Peers: peers,
		AverageBlockIntervalS: average,
		RecentIntervalsSeconds: intervals,
		IndexHeight: indexHeight, IndexTipHash: indexTip,
	}, nil
}

func (s *Server) blockIntervals(
	ctx context.Context,
	height uint64,
	limit uint64,
) ([]int64, float64, error) {
	if height == 0 {
		return []int64{}, 0, nil
	}
	count := limit
	if count > height { count = height }
	start := height - count
	var previous rpc.BlockResult
	if err := s.client.Call(ctx, rpc.MethodGetBlock, rpc.HeightParams{Height: start}, &previous); err != nil {
		return nil, 0, err
	}
	intervals := make([]int64, 0, count)
	var total int64
	for h := start + 1; h <= height; h++ {
		var current rpc.BlockResult
		if err := s.client.Call(ctx, rpc.MethodGetBlock, rpc.HeightParams{Height: h}, &current); err != nil {
			return nil, 0, err
		}
		delta := current.Timestamp - previous.Timestamp
		intervals = append(intervals, delta)
		total += delta
		previous = current
	}
	if len(intervals) == 0 { return intervals, 0, nil }
	return intervals, float64(total) / float64(len(intervals)), nil
}

func (s *Server) handleAPIBlocks(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	limit := defaultRecentBlocks
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || value == 0 || value > maxRecentBlocks {
			writeAPIError(w, http.StatusBadRequest, errors.New("limit must be 1..100"))
			return
		}
		limit = value
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()
	var status rpc.StatusResult
	if err := s.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		writeAPIError(w, http.StatusBadGateway, err); return
	}
	blocks := make([]rpc.BlockResult, 0, limit)
	for offset := uint64(0); offset < limit && offset <= status.Height; offset++ {
		var candidate rpc.BlockResult
		if err := s.client.Call(ctx, rpc.MethodGetBlock, rpc.HeightParams{Height: status.Height-offset}, &candidate); err != nil {
			writeAPIError(w, http.StatusBadGateway, err); return
		}
		blocks = append(blocks, candidate)
	}
	writeJSON(w, http.StatusOK, blocks)
}

func (s *Server) handleAPIBlock(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/block/"))
	candidate, err := s.lookupBlock(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err); return
	}
	writeJSON(w, http.StatusOK, candidate)
}

func (s *Server) lookupBlock(parent context.Context, id string) (rpc.BlockResult, error) {
	if id == "" || strings.Contains(id, "/") {
		return rpc.BlockResult{}, errors.New("block height or hash required")
	}
	ctx, cancel := context.WithTimeout(parent, s.requestTimeout)
	defer cancel()
	var candidate rpc.BlockResult
	if height, err := strconv.ParseUint(id, 10, 64); err == nil {
		err = s.client.Call(ctx, rpc.MethodGetBlock, rpc.HeightParams{Height: height}, &candidate)
		return candidate, err
	}
	if !isHash(id) { return candidate, errors.New("invalid block hash") }
	err := s.client.Call(ctx, rpc.MethodGetBlockByHash, rpc.HashParams{Hash: id}, &candidate)
	return candidate, err
}

func (s *Server) handleAPITransaction(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	txid := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/tx/"))
	if !isHash(txid) { writeAPIError(w, http.StatusBadRequest, errors.New("invalid transaction id")); return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	var result rpc.TransactionResult
	if err := s.client.Call(ctx, rpc.MethodGetTransaction, rpc.TransactionParams{TransactionID: txid}, &result); err != nil {
		writeAPIError(w, http.StatusNotFound, err); return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPIAddress(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	address, err := url.PathUnescape(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/address/")))
	if err != nil || !valdrcrypto.ValidateAddress(address) {
		writeAPIError(w, http.StatusBadRequest, errors.New("invalid VALDR address")); return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	if err := s.index.Refresh(ctx); err != nil { writeAPIError(w, http.StatusBadGateway, err); return }
	var balance rpc.BalanceResult
	if err := s.client.Call(ctx, rpc.MethodGetBalance, rpc.AddressParams{Address: address}, &balance); err != nil {
		writeAPIError(w, http.StatusBadGateway, err); return
	}
	writeJSON(w, http.StatusOK, AddressAPI{
		Balance: balance,
		UTXOs: s.index.UTXOs(address),
		History: s.index.Activities(address),
	})
}

func (s *Server) handleAPIMempool(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	var result any
	var txs []json.RawMessage
	if err := s.client.Call(ctx, rpc.MethodGetMempool, nil, &txs); err != nil {
		writeAPIError(w, http.StatusBadGateway, err); return
	}
	result = txs
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" { http.NotFound(w, r); return }
	if !requireGET(w, r) { return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	status, err := s.status(ctx)
	if err != nil { renderPage(w, http.StatusBadGateway, "VALDR Explorer", nil, err); return }
	renderPage(w, http.StatusOK, "VALDR Explorer", status, nil)
}

func (s *Server) handleBlockView(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	id := strings.TrimPrefix(r.URL.Path, "/block/")
	value, err := s.lookupBlock(r.Context(), id)
	renderPage(w, mapStatus(err), "VALDR Block", value, err)
}

func (s *Server) handleTransactionView(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	txid := strings.TrimPrefix(r.URL.Path, "/tx/")
	if !isHash(txid) { renderPage(w, http.StatusBadRequest, "VALDR Transaction", nil, errors.New("invalid transaction id")); return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	var value rpc.TransactionResult
	err := s.client.Call(ctx, rpc.MethodGetTransaction, rpc.TransactionParams{TransactionID: txid}, &value)
	renderPage(w, mapStatus(err), "VALDR Transaction", value, err)
}

func (s *Server) handleAddressView(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	address, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/address/"))
	if err != nil || !valdrcrypto.ValidateAddress(address) {
		renderPage(w, http.StatusBadRequest, "VALDR Address", nil, errors.New("invalid VALDR address")); return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	if err := s.index.Refresh(ctx); err != nil { renderPage(w, http.StatusBadGateway, "VALDR Address", nil, err); return }
	var balance rpc.BalanceResult
	err = s.client.Call(ctx, rpc.MethodGetBalance, rpc.AddressParams{Address: address}, &balance)
	value := AddressAPI{Balance: balance, UTXOs: s.index.UTXOs(address), History: s.index.Activities(address)}
	renderPage(w, mapStatus(err), "VALDR Address", value, err)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" { http.Error(w, "search query required", http.StatusBadRequest); return }
	if valdrcrypto.ValidateAddress(q) {
		http.Redirect(w, r, "/address/"+url.PathEscape(q), http.StatusSeeOther); return
	}
	if _, err := strconv.ParseUint(q, 10, 64); err == nil {
		http.Redirect(w, r, "/block/"+q, http.StatusSeeOther); return
	}
	if !isHash(q) { http.Error(w, "unsupported search query", http.StatusBadRequest); return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	var candidate rpc.BlockResult
	if s.client.Call(ctx, rpc.MethodGetBlockByHash, rpc.HashParams{Hash: q}, &candidate) == nil {
		http.Redirect(w, r, "/block/"+q, http.StatusSeeOther); return
	}
	var tx rpc.TransactionResult
	if s.client.Call(ctx, rpc.MethodGetTransaction, rpc.TransactionParams{TransactionID: q}, &tx) == nil {
		http.Redirect(w, r, "/tx/"+q, http.StatusSeeOther); return
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) { return }
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout); defer cancel()
	var status rpc.StatusResult
	if err := s.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status":"unavailable"}); return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status":"ok","chain_id":status.ChainID,"height":status.Height,"tip_hash":status.TipHash})
}

func requireGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead { return true }
	w.Header().Set("Allow", http.MethodGet)
	http.Error(w, "GET required", http.StatusMethodNotAllowed)
	return false
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func mapStatus(err error) int {
	if err != nil { return http.StatusNotFound }
	return http.StatusOK
}

func isHash(value string) bool {
	if len(value) != 64 { return false }
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func renderPage(w http.ResponseWriter, status int, title string, value any, err error) {
	var pretty string
	if value != nil {
		raw, _ := json.MarshalIndent(value, "", "  ")
		pretty = string(raw)
	}
	message := ""
	if err != nil { message = err.Error() }
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = pageTemplate.Execute(w, map[string]any{
		"Title": title, "JSON": pretty, "Error": message,
	})
}

var pageTemplate = template.Must(template.New("explorer").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}}</title><style>body{font-family:system-ui;max-width:1100px;margin:auto;padding:24px;background:#10141c;color:#eef}a{color:#9cf}input{padding:8px;width:65%}button{padding:8px}pre{white-space:pre-wrap;overflow-wrap:anywhere;border:1px solid #344;border-radius:8px;padding:16px}.err{color:#f99}</style></head>
<body><h1><a href="/">VALDR Explorer</a></h1>
<form action="/search"><input name="q" placeholder="height, block hash, txid, VDR address"><button>Search</button></form>
<p><a href="/api/v1/status">Status API</a> · <a href="/api/v1/blocks">Blocks API</a> · <a href="/api/v1/mempool">Mempool API</a></p>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}{{if .JSON}}<pre>{{.JSON}}</pre>{{end}}
<footer>VALDR read-only explorer</footer></body></html>`))
