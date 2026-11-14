package fossapay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

type realClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewRealClient(baseURL, apiKey string) ports.FossapayClient {
	return &realClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    http.DefaultClient,
	}
}

func (c *realClient) CreateCheckout(ctx context.Context, idempotencyKey string, req domain.FossapayCreateCheckoutReq) (*domain.FossapayCheckoutResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	u := c.baseURL + "/checkouts"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		httpReq.Header.Set("X-Idempotency-Key", idempotencyKey)
	}

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fossapay request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("fossapay error (HTTP %d): %s", res.StatusCode, string(respBytes))
	}

	var out domain.FossapayCheckoutResponse
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("decode fossapay response: %w", err)
	}

	return &out, nil
}

func (c *realClient) GetCheckout(ctx context.Context, checkoutID string) (*domain.FossapayCheckoutResponse, error) {
	u := c.baseURL + "/checkouts/" + checkoutID
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var out domain.FossapayCheckoutResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *realClient) GetCheckoutByReference(ctx context.Context, reference string) (*domain.FossapayCheckoutResponse, error) {
	u := c.baseURL + "/checkouts/reference/" + reference
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("fossapay lookup HTTP %d: %.300s", res.StatusCode, string(raw))
	}
	// Lenient on purpose: only the fields we need, and any timestamp shape.
	// A strict struct once made us discard a "completed" payment.
	var lenient struct {
		Status     string `json:"status"`
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
		Data       struct {
			ID            string `json:"id"`
			Reference     string `json:"reference"`
			Status        string `json:"status"`
			Amount        any    `json:"amount"`
			AmountPayable any    `json:"amountPayable"`
			TransactionID any    `json:"transactionId"`
			CompletedAt   any    `json:"completedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &lenient); err != nil {
		return nil, fmt.Errorf("fossapay lookup: bad JSON: %.300s", string(raw))
	}
	if lenient.Data.Status == "" {
		return nil, fmt.Errorf("fossapay lookup: no status in response: %.400s", string(raw))
	}
	var out domain.FossapayCheckoutResponse
	out.Status = lenient.Status
	out.StatusCode = lenient.StatusCode
	out.Message = lenient.Message
	out.Data.ID = lenient.Data.ID
	out.Data.Reference = lenient.Data.Reference
	out.Data.Status = lenient.Data.Status
	out.Data.Amount = fmt.Sprint(lenient.Data.Amount)
	out.Data.AmountPayable = fmt.Sprint(lenient.Data.AmountPayable)
	if lenient.Data.TransactionID != nil {
		out.Data.TransactionID = fmt.Sprint(lenient.Data.TransactionID)
	}
	if str, ok := lenient.Data.CompletedAt.(string); ok && str != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, str); err == nil {
				out.Data.CompletedAt = &t
				break
			}
		}
	}
	return &out, nil
}

func (c *realClient) VerifyWebhookSignature(body []byte, signature, secret string) bool {
	if len(body) == 0 || signature == "" || secret == "" {
		return false
	}
	// FossaPay signs JSON.stringify(body.data)
	var envelope struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	dataBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(dataBytes)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
