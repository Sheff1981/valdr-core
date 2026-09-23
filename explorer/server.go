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

	"github.com/Sheff1981/valdr-core/config"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/rpc"
)

const (
	defaultRequestTimeout = 5 * time.Second
	defaultRecentBlocks   = uint64(10)
)

var ErrInvalidClient = errors.New("explorer RPC client is nil")

type RPCClient interface {
	Call(context.Context, string, any, any) error
}

type Server struct {
	client         RPCClient
	index          *Index
	requestTimeout time.Duration
	recentBlocks   uint64
}

type pageData struct {
	Title       string
	Description string
	Status      *rpc.StatusResult
	Blocks      []rpc.BlockResult
	Block       *rpc.BlockResult
	Transaction *rpc.TransactionResult
	Balance     *rpc.BalanceResult
	History     []AddressActivity
	Error       string
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
		recentBlocks:   defaultRecentBlocks,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleOverview)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/block/", s.handleBlock)
	mux.HandleFunc("/tx/", s.handleTransaction)
	mux.HandleFunc("/address/", s.handleAddress)
	mux.HandleFunc("/healthz", s.handleHealth)
	return securityHeaders(mux)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !requireGET(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var status rpc.StatusResult
	if err := s.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		s.renderError(w, http.StatusBadGateway, "VALDR node is unavailable")
		return
	}

	blocks := make([]rpc.BlockResult, 0, s.recentBlocks)
	for offset := uint64(0); offset < s.recentBlocks && offset <= status.Height; offset++ {
		var candidate rpc.BlockResult
		if err := s.client.Call(
			ctx,
			rpc.MethodGetBlock,
			rpc.HeightParams{Height: status.Height - offset},
			&candidate,
		); err != nil {
			s.renderError(w, http.StatusBadGateway, "Unable to load recent blocks")
			return
		}
		blocks = append(blocks, candidate)
	}

	s.render(w, http.StatusOK, pageData{
		Title:       "VALDR Explorer",
		Description: "Read-only VALDR blockchain explorer",
		Status:      &status,
		Blocks:      blocks,
	})
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/block/"))
	if id == "" || strings.Contains(id, "/") {
		s.renderError(w, http.StatusBadRequest, "Block height or hash is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var candidate rpc.BlockResult
	if height, err := strconv.ParseUint(id, 10, 64); err == nil {
		if err := s.client.Call(ctx, rpc.MethodGetBlock, rpc.HeightParams{Height: height}, &candidate); err != nil {
			s.renderError(w, http.StatusNotFound, "Block not found")
			return
		}
	} else {
		if !isHash(id) {
			s.renderError(w, http.StatusBadRequest, "Invalid block hash")
			return
		}
		if err := s.client.Call(ctx, rpc.MethodGetBlockByHash, rpc.HashParams{Hash: id}, &candidate); err != nil {
			s.renderError(w, http.StatusNotFound, "Block not found")
			return
		}
	}

	s.render(w, http.StatusOK, pageData{
		Title:       fmt.Sprintf("Block %d - VALDR Explorer", candidate.Height),
		Description: "VALDR block details",
		Block:       &candidate,
	})
}

func (s *Server) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	txid := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/tx/"))
	if !isHash(txid) {
		s.renderError(w, http.StatusBadRequest, "Invalid transaction ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var result rpc.TransactionResult
	if err := s.client.Call(
		ctx,
		rpc.MethodGetTransaction,
		rpc.TransactionParams{TransactionID: txid},
		&result,
	); err != nil {
		s.renderError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	s.render(w, http.StatusOK, pageData{
		Title:       "Transaction - VALDR Explorer",
		Description: "VALDR transaction details",
		Transaction: &result,
	})
}

func (s *Server) handleAddress(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	address, err := url.PathUnescape(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/address/")))
	if err != nil || !valdrcrypto.ValidateAddress(address) {
		s.renderError(w, http.StatusBadRequest, "Invalid VALDR address")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var balance rpc.BalanceResult
	if err := s.client.Call(ctx, rpc.MethodGetBalance, rpc.AddressParams{Address: address}, &balance); err != nil {
		s.renderError(w, http.StatusBadGateway, "Unable to load address balance")
		return
	}
	if err := s.index.Refresh(ctx); err != nil {
		s.renderError(w, http.StatusBadGateway, "Unable to update explorer index")
		return
	}

	s.render(w, http.StatusOK, pageData{
		Title:       "Address - VALDR Explorer",
		Description: "VALDR address balance and confirmed history",
		Balance:     &balance,
		History:     s.index.Activities(address),
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		s.renderError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	if valdrcrypto.ValidateAddress(query) {
		http.Redirect(w, r, "/address/"+url.PathEscape(query), http.StatusSeeOther)
		return
	}
	if _, err := strconv.ParseUint(query, 10, 64); err == nil {
		http.Redirect(w, r, "/block/"+query, http.StatusSeeOther)
		return
	}
	if !isHash(query) {
		s.renderError(w, http.StatusBadRequest, "Search supports block height, block hash, transaction ID, or VDR address")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var block rpc.BlockResult
	if err := s.client.Call(ctx, rpc.MethodGetBlockByHash, rpc.HashParams{Hash: query}, &block); err == nil {
		http.Redirect(w, r, "/block/"+query, http.StatusSeeOther)
		return
	}

	var tx rpc.TransactionResult
	if err := s.client.Call(
		ctx,
		rpc.MethodGetTransaction,
		rpc.TransactionParams{TransactionID: query},
		&tx,
	); err == nil {
		http.Redirect(w, r, "/tx/"+query, http.StatusSeeOther)
		return
	}

	s.renderError(w, http.StatusNotFound, "No matching block or transaction")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireGET(w, r) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	var status rpc.StatusResult
	if err := s.client.Call(ctx, rpc.MethodGetStatus, nil, &status); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "unavailable"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"chain_id": status.ChainID,
		"height":   status.Height,
		"tip_hash": status.TipHash,
	})
}

func (s *Server) renderError(w http.ResponseWriter, status int, message string) {
	s.render(w, status, pageData{
		Title:       "VALDR Explorer",
		Description: "VALDR explorer error",
		Error:       message,
	})
}

func (s *Server) render(w http.ResponseWriter, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := pageTemplate.Execute(w, data); err != nil {
		return
	}
}

func requireGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
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

func isHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func formatVDR(value uint64) string {
	whole := value / config.AtomicUnitsPerVDR
	fraction := value % config.AtomicUnitsPerVDR
	if fraction == 0 {
		return fmt.Sprintf("%d VDR", whole)
	}
	formatted := fmt.Sprintf("%d.%08d", whole, fraction)
	formatted = strings.TrimRight(formatted, "0")
	return formatted + " VDR"
}

func formatTime(timestamp int64) string {
	if timestamp <= 0 {
		return "-"
	}
	return time.Unix(timestamp, 0).UTC().Format("2006-01-02 15:04:05 UTC")
}

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"vdr":  formatVDR,
	"time": formatTime,
}).Parse(pageHTML))

const pageHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title><style>body{font-family:system-ui;margin:auto;max-width:1080px;padding:24px;background:#10141c;color:#eef}a{color:#9cf}code{overflow-wrap:anywhere}section{border:1px solid #344;padding:16px;margin:16px 0;border-radius:10px}table{width:100%;border-collapse:collapse}td,th{padding:8px;border-bottom:1px solid #344;text-align:left}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px}.muted{color:#9aa}input{min-width:55%;padding:8px}button{padding:8px}</style></head><body>
<header><h1><a href="/">VALDR Explorer</a></h1><form action="/search"><input name="q" placeholder="height, hash, txid, VDR address"><button>Search</button></form></header>
{{if .Error}}<section><strong>{{.Error}}</strong></section>{{end}}
{{with .Status}}<section><h2>Network</h2><div class="grid"><div>Chain<br><b>{{.ChainID}}</b></div><div>Height<br><b>{{.Height}}</b></div><div>Peers<br><b>{{.PeerCount}}</b></div><div>Mempool<br><b>{{.MempoolCount}}</b></div></div><p class="muted">Tip <code>{{.TipHash}}</code></p></section>{{end}}
{{if .Blocks}}<section><h2>Recent blocks</h2><table><tr><th>Height</th><th>Hash</th><th>Time</th><th>Tx</th></tr>{{range .Blocks}}<tr><td><a href="/block/{{.Height}}">{{.Height}}</a></td><td><a href="/block/{{.BlockHash}}"><code>{{.BlockHash}}</code></a></td><td>{{time .Timestamp}}</td><td>{{len .Transactions}}</td></tr>{{end}}</table></section>{{end}}
{{with .Block}}<section><h2>Block {{.Height}}</h2><p>Hash <code>{{.BlockHash}}</code></p><p>Previous {{if .PreviousBlockHash}}<a href="/block/{{.PreviousBlockHash}}"><code>{{.PreviousBlockHash}}</code></a>{{else}}Genesis{{end}}</p><p>Time {{time .Timestamp}} · Difficulty {{.Difficulty}} · Nonce {{.Nonce}}</p><p>Merkle <code>{{.MerkleRoot}}</code></p>{{if .Transactions}}<h3>Transactions</h3>{{range .Transactions}}{{if .}}<p><a href="/tx/{{.TransactionID}}"><code>{{.TransactionID}}</code></a></p>{{end}}{{end}}{{end}}</section>{{end}}
{{with .Transaction}}{{with .Transaction}}<section><h2>Transaction</h2><p><code>{{.TransactionID}}</code></p><p>Status: {{if $.Transaction.Confirmed}}Confirmed in <a href="/block/{{$.Transaction.BlockHeight}}">block {{$.Transaction.BlockHeight}}</a>{{else}}Mempool{{end}}</p><h3>Inputs</h3>{{range .Inputs}}<p><code>{{.PreviousTransactionID}}:{{.OutputIndex}}</code></p>{{end}}<h3>Outputs</h3>{{range .Outputs}}<p><a href="/address/{{.Recipient}}"><code>{{.Recipient}}</code></a> — <b>{{vdr .Amount}}</b></p>{{end}}</section>{{end}}{{end}}
{{with .Balance}}<section><h2>Address</h2><p><code>{{.Address}}</code></p><p>Confirmed balance: <b>{{vdr .BalanceVal}}</b></p></section>{{end}}
{{if .History}}<section><h2>Confirmed activity</h2><table><tr><th>Block</th><th>Transaction</th><th>Received</th><th>Spent</th></tr>{{range .History}}<tr><td><a href="/block/{{.BlockHeight}}">{{.BlockHeight}}</a></td><td><a href="/tx/{{.TransactionID}}"><code>{{.TransactionID}}</code></a></td><td>{{vdr .ReceivedVal}}</td><td>{{vdr .SpentVal}}</td></tr>{{end}}</table></section>{{else}}{{with .Balance}}<section><p class="muted">No confirmed transaction activity for this address.</p></section>{{end}}{{end}}
<footer class="muted">VALDR read-only explorer</footer></body></html>`
