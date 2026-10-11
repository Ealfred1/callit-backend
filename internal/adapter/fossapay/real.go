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

	var out domain.FossapayCheckoutResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
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
