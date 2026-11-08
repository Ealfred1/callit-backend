package nearintents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

const Default1ClickBaseURL = "https://1click.chaindefuser.com/v0"

// Standard asset IDs on NEAR Intents
const (
	AssetIdNearWNear = "nep141:wrap.near"
	AssetIdSolSOL    = "nep141:sol.omft.near"
	AssetIdEthETH    = "nep141:eth.omft.near"
	AssetIdBaseETH   = "nep141:base.omft.near"
	AssetIdEthUSDC   = "nep141:eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near"
	AssetIdNearUSDC  = "nep141:17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1"
)

type realClient struct {
	baseURL string
	http    *http.Client
}

func NewRealClient(baseURL string) ports.NearIntentsClient {
	if baseURL == "" {
		baseURL = Default1ClickBaseURL
	}
	return &realClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type tokenRaw struct {
	AssetID         string  `json:"assetId"`
	Symbol          string  `json:"symbol"`
	Blockchain      string  `json:"blockchain"`
	Decimals        int     `json:"decimals"`
	Price           float64 `json:"price"`
	ContractAddress string  `json:"contractAddress"`
}

func (c *realClient) GetTokens(ctx context.Context) ([]domain.NearIntentToken, error) {
	u := c.baseURL + "/tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Callit/1.0 (NEAR Hackathon; +https://callit.my)")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("1click tokens request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("1click tokens error (%d): %s", res.StatusCode, string(b))
	}

	var raw []tokenRaw
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode tokens response: %w", err)
	}

	out := make([]domain.NearIntentToken, 0, len(raw))
	for _, t := range raw {
		out = append(out, domain.NearIntentToken{
			AssetID:         t.AssetID,
			Symbol:          t.Symbol,
			Blockchain:      t.Blockchain,
			Decimals:        t.Decimals,
			PriceUSD:        t.Price,
			ContractAddress: t.ContractAddress,
		})
	}
	return out, nil
}

type quoteReqPayload struct {
	Dry               bool   `json:"dry"`
	SwapType          string `json:"swapType"`
	SlippageTolerance int    `json:"slippageTolerance"`
	OriginAsset       string `json:"originAsset"`
	DepositType       string `json:"depositType"`
	DestinationAsset  string `json:"destinationAsset"`
	Amount            string `json:"amount"`
	RefundTo          string `json:"refundTo"`
	RefundType        string `json:"refundType"`
	Recipient         string `json:"recipient"`
	RecipientType     string `json:"recipientType"`
	Deadline          string `json:"deadline"`
}

type quoteRespPayload struct {
	Quote struct {
		AmountIn           string `json:"amountIn"`
		AmountInFormatted  string `json:"amountInFormatted"`
		AmountInUSD        string `json:"amountInUsd"`
		AmountOut          string `json:"amountOut"`
		AmountOutFormatted string `json:"amountOutFormatted"`
		AmountOutUSD       string `json:"amountOutUsd"`
		DepositAddress     string `json:"depositAddress"`
		TimeEstimate       int    `json:"timeEstimate"`
		Deadline           string `json:"deadline"`
	} `json:"quote"`
	CorrelationID string `json:"correlationId"`
	StatusCode    int    `json:"statusCode"`
	Message       any    `json:"message"`
	Error         string `json:"error"`
}

func (c *realClient) CreateQuote(ctx context.Context, req domain.NearIntentQuoteReq) (*domain.NearIntentQuoteRes, error) {
	originAsset := resolveAssetID(req.OriginAsset)
	destinationAsset := resolveAssetID(req.DestinationAsset)
	if destinationAsset == "" {
		destinationAsset = AssetIdSolSOL // default destination is Solana SOL or USDC
	}

	// Calculate base unit amount if decimals can be determined
	amountBaseUnits := req.Amount
	if req.AmountFormatted != "" {
		amountBaseUnits = parseAmountToBaseUnits(req.AmountFormatted, getAssetDecimals(originAsset))
	} else if strings.Contains(req.Amount, ".") {
		amountBaseUnits = parseAmountToBaseUnits(req.Amount, getAssetDecimals(originAsset))
	}

	slippage := req.SlippageTolerance
	if slippage <= 0 {
		slippage = 100 // 1%
	}

	refundTo := req.RefundTo
	if refundTo == "" {
		refundTo = req.Recipient
	}

	deadline := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	payload := quoteReqPayload{
		Dry:               req.Dry,
		SwapType:          "EXACT_INPUT",
		SlippageTolerance: slippage,
		OriginAsset:       originAsset,
		DepositType:       "ORIGIN_CHAIN",
		DestinationAsset:  destinationAsset,
		Amount:            amountBaseUnits,
		RefundTo:          refundTo,
		RefundType:        "ORIGIN_CHAIN",
		Recipient:         req.Recipient,
		RecipientType:     "DESTINATION_CHAIN",
		Deadline:          deadline,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	u := c.baseURL + "/quote"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Callit/1.0 (NEAR Hackathon; +https://callit.my)")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("1click quote request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("1click quote rejected (HTTP %d): %s", res.StatusCode, string(respBytes))
	}

	var parsed quoteRespPayload
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode quote response: %w", err)
	}

	deadlineTime, _ := time.Parse(time.RFC3339, parsed.Quote.Deadline)
	if deadlineTime.IsZero() {
		deadlineTime = time.Now().Add(24 * time.Hour)
	}

	return &domain.NearIntentQuoteRes{
		DepositAddress:     parsed.Quote.DepositAddress,
		OriginAsset:        originAsset,
		DestinationAsset:   destinationAsset,
		AmountIn:           parsed.Quote.AmountIn,
		AmountInFormatted:  parsed.Quote.AmountInFormatted,
		AmountInUSD:        parsed.Quote.AmountInUSD,
		AmountOut:          parsed.Quote.AmountOut,
		AmountOutFormatted: parsed.Quote.AmountOutFormatted,
		AmountOutUSD:       parsed.Quote.AmountOutUSD,
		TimeEstimateSec:    parsed.Quote.TimeEstimate,
		Deadline:           deadlineTime,
		Status:             "PENDING_DEPOSIT",
		CorrelationID:      parsed.CorrelationID,
	}, nil
}

type statusRespPayload struct {
	Status      string `json:"status"`
	UpdatedAt   string `json:"updatedAt"`
	SwapDetails struct {
		DepositedAmount          string   `json:"depositedAmount"`
		AmountOut                string   `json:"amountOut"`
		NearTxHashes             []string `json:"nearTxHashes"`
		OriginChainTxHashes      []string `json:"originChainTxHashes"`
		DestinationChainTxHashes []string `json:"destinationChainTxHashes"`
	} `json:"swapDetails"`
}

func (c *realClient) GetStatus(ctx context.Context, depositAddress string) (*domain.NearIntentStatusRes, error) {
	u := fmt.Sprintf("%s/status?depositAddress=%s", c.baseURL, depositAddress)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Callit/1.0 (NEAR Hackathon; +https://callit.my)")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("1click status request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("1click status error (%d): %s", res.StatusCode, string(b))
	}

	var parsed statusRespPayload
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}

	return &domain.NearIntentStatusRes{
		Status:          parsed.Status,
		DepositAddress:  depositAddress,
		DepositedAmount: parsed.SwapDetails.DepositedAmount,
		AmountOut:       parsed.SwapDetails.AmountOut,
		NearTxHashes:    parsed.SwapDetails.NearTxHashes,
		OriginTxHashes:  parsed.SwapDetails.OriginChainTxHashes,
		DestTxHashes:    parsed.SwapDetails.DestinationChainTxHashes,
		UpdatedAt:       parsed.UpdatedAt,
	}, nil
}

func resolveAssetID(input string) string {
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case "near", "wnear", "wrap.near":
		return AssetIdNearWNear
	case "sol", "solana":
		return AssetIdSolSOL
	case "eth", "ethereum":
		return AssetIdEthETH
	case "base", "base-eth":
		return AssetIdBaseETH
	case "usdc":
		return AssetIdEthUSDC
	default:
		return input
	}
}

func getAssetDecimals(assetID string) int {
	switch {
	case strings.Contains(assetID, "wrap.near"):
		return 24
	case strings.Contains(assetID, "sol.omft"):
		return 9
	case strings.Contains(assetID, "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48") || strings.Contains(assetID, "17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1"):
		return 6 // USDC
	default:
		return 18
	}
}

func parseAmountToBaseUnits(formatted string, decimals int) string {
	parts := strings.Split(formatted, ".")
	whole := parts[0]
	frac := ""
	if len(parts) > 1 {
		frac = parts[1]
	}

	if len(frac) > decimals {
		frac = frac[:decimals]
	} else {
		frac = frac + strings.Repeat("0", decimals-len(frac))
	}

	combined := strings.TrimLeft(whole+frac, "0")
	if combined == "" {
		return "0"
	}
	// Verify it parses as big.Int
	n := new(big.Int)
	n.SetString(combined, 10)
	return n.String()
}
