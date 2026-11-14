package handler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/labstack/echo/v4"
)

// WalletBalanceHandler reads a wallet's SOL and USDC balances on the server.
//
// Browsers cannot call Solana's public RPC (it answers 403 to any web origin),
// which made the web app show a zero balance forever. The app asks this
// endpoint instead; it tries several RPCs and keeps the answer for a few
// seconds so repeat opens are instant.
type WalletBalanceHandler struct {
	rpcs   []string
	client *http.Client
	mu     sync.Mutex
	cache  map[string]cachedBalance
}

type cachedBalance struct {
	at   time.Time
	data WalletBalances
}

// WalletBalances is the JSON the app receives.
type WalletBalances struct {
	Wallet   string  `json:"wallet"`
	SOL      float64 `json:"sol"`
	USDC     float64 `json:"usdc"`
	Source   string  `json:"source"`
	CachedAt string  `json:"asOf"`
}

const usdcMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"

var base58Wallet = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

func NewWalletBalanceHandler() *WalletBalanceHandler {
	rpcs := []string{"https://solana-rpc.publicnode.com", "https://api.mainnet-beta.solana.com"}
	if v := strings.TrimSpace(os.Getenv("SOLANA_RPC_URLS")); v != "" {
		rpcs = append(strings.Split(v, ","), rpcs...) // your own RPC first when configured
	}
	return &WalletBalanceHandler{rpcs: rpcs, client: &http.Client{Timeout: 6 * time.Second, Transport: &http.Transport{TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{}, MaxIdleConnsPerHost: 4, DialContext: ipv4Dialer}}, cache: map[string]cachedBalance{}}
}

func (h *WalletBalanceHandler) call(ctx context.Context, url, method string, params []any) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(url), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CallitBalance/1.0; +https://callit.my)")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%s http %d: %.80s", method, res.StatusCode, string(raw))
	}
	if len(out.Error) > 0 && string(out.Error) != "null" {
		return nil, fmt.Errorf("%s rpc error: %.120s", method, string(out.Error))
	}
	return out.Result, nil
}

func (h *WalletBalanceHandler) fetch(ctx context.Context, rpc, wallet string) (*WalletBalances, error) {
	solRaw, err := h.call(ctx, rpc, "getBalance", []any{wallet})
	if err != nil {
		return nil, err
	}
	var sol struct {
		Value float64 `json:"value"`
	}
	_ = json.Unmarshal(solRaw, &sol)

	// USDC: read the wallet's standard USDC token account with a plain account
	// lookup and decode the raw SPL token layout (amount = u64 at byte 64).
	// Token-specific methods are blocked or key-gated on free RPCs.
	var usdc float64
	ata, err := associatedTokenAddress(wallet, usdcMint)
	if err != nil {
		return nil, err
	}
	accRaw, err := h.call(ctx, rpc, "getAccountInfo", []any{ata, map[string]string{"encoding": "base64"}})
	if err != nil {
		return nil, err
	}
	var acc struct {
		Value *struct {
			Data []string `json:"data"`
		} `json:"value"`
	}
	_ = json.Unmarshal(accRaw, &acc)
	if acc.Value != nil && len(acc.Value.Data) > 0 {
		raw, derr := base64.StdEncoding.DecodeString(acc.Value.Data[0])
		if derr == nil && len(raw) >= 72 {
			usdc = float64(binary.LittleEndian.Uint64(raw[64:72])) / 1e6
		}
	} // value null: no token account yet, the wallet holds no USDC
	return &WalletBalances{Wallet: wallet, SOL: sol.Value / 1e9, USDC: usdc, Source: rpc}, nil
}

// Get handles GET /v1/wallets/:wallet/balances
func (h *WalletBalanceHandler) Get(c echo.Context) error {
	wallet := c.Param("wallet")
	if !base58Wallet.MatchString(wallet) {
		return respond.Err(c, http.StatusBadRequest, "INVALID_WALLET", "wallet must be a Solana address", nil)
	}
	fresh := c.QueryParam("fresh") == "1"

	h.mu.Lock()
	if cb, ok := h.cache[wallet]; ok && !fresh && time.Since(cb.at) < 8*time.Second {
		h.mu.Unlock()
		return respond.OK(c, cb.data)
	}
	h.mu.Unlock()

	var errs []string
	for _, rpc := range h.rpcs {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 7*time.Second)
		b, err := h.fetch(ctx, rpc, wallet)
		cancel()
		if err != nil {
			errs = append(errs, rpc+": "+err.Error())
			continue
		}
		b.CachedAt = time.Now().UTC().Format(time.RFC3339)
		h.mu.Lock()
		h.cache[wallet] = cachedBalance{at: time.Now(), data: *b}
		h.mu.Unlock()
		return respond.OK(c, *b)
	}
	// Serve a stale value rather than a misleading zero.
	h.mu.Lock()
	cb, ok := h.cache[wallet]
	h.mu.Unlock()
	if ok {
		return respond.OK(c, cb.data)
	}
	msg := "could not reach Solana right now"
	if len(errs) > 0 {
		msg += " (" + strings.Join(errs, "; ") + ")"
	}
	return respond.Err(c, http.StatusBadGateway, "RPC_UNAVAILABLE", msg, nil)
}

// ipv4Dialer forces IPv4: the VPS IPv6 range is blocked by Solana's public RPC
// ("Your IP or provider is blocked"), while its IPv4 address is accepted.
func ipv4Dialer(ctx context.Context, _, addr string) (net.Conn, error) {
	d := net.Dialer{Timeout: 4 * time.Second}
	return d.DialContext(ctx, "tcp4", addr)
}
