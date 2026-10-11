package domain

import (
	"time"

	"github.com/google/uuid"
)

type FundingStatus string

const (
	FundingStatusPending   FundingStatus = "pending"
	FundingStatusCompleted FundingStatus = "completed"
	FundingStatusFailed    FundingStatus = "failed"
	FundingStatusExpired   FundingStatus = "expired"
)

type FundingTransaction struct {
	ID               uuid.UUID      `json:"id"`
	UserID           uuid.UUID      `json:"userId"`
	WalletAddress    string         `json:"walletAddress"`
	Reference        string         `json:"reference"`
	CheckoutID       string         `json:"checkoutId"`
	AmountNGN        float64        `json:"amountNgn"`
	AmountPayableNGN float64        `json:"amountPayableNgn"`
	EstimatedUSDC    float64        `json:"estimatedUsdc"`
	Status           FundingStatus  `json:"status"`
	BankName         string         `json:"bankName"`
	AccountName      string         `json:"accountName"`
	AccountNumber    string         `json:"accountNumber"`
	ExpiresAt        *time.Time     `json:"expiresAt,omitempty"`
	CompletedAt      *time.Time     `json:"completedAt,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type CreateFundingRequest struct {
	AmountNGN     float64 `json:"amountNgn"`
	WalletAddress string  `json:"walletAddress"`
	PhoneNumber   string  `json:"phoneNumber,omitempty"`
}

type FundingCheckoutResponse struct {
	Reference        string     `json:"reference"`
	CheckoutID       string     `json:"checkoutId"`
	AmountNGN        float64    `json:"amountNgn"`
	AmountPayableNGN float64    `json:"amountPayableNgn"`
	EstimatedUSDC    float64    `json:"estimatedUsdc"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	BankName         string     `json:"bankName"`
	AccountName      string     `json:"accountName"`
	AccountNumber    string     `json:"accountNumber"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
}

type FossapayCustomer struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

type FossapayCreateCheckoutReq struct {
	Amount      int64            `json:"amount"` // in NGN
	Currency    string           `json:"currency"`
	Reference   string           `json:"reference"`
	FeeBearer   string           `json:"feeBearer"`
	Customer    FossapayCustomer `json:"customer"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	Description string           `json:"description,omitempty"`
}

type FossapayAccount struct {
	Type          string     `json:"type"`
	BankName      string     `json:"bankName"`
	AccountName   string     `json:"accountName"`
	AccountNumber string     `json:"accountNumber"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

type FossapayCheckoutData struct {
	ID               string          `json:"id"`
	Reference        string          `json:"reference"`
	Status           string          `json:"status"`
	Amount           string          `json:"amount"`
	Fee              string          `json:"fee"`
	AmountPayable    string          `json:"amountPayable"`
	SettlementAmount string          `json:"settlementAmount"`
	FeeBearer        string          `json:"feeBearer"`
	Currency         string          `json:"currency"`
	Account          FossapayAccount `json:"account"`
	TransactionID    string          `json:"transactionId"`
	ExpiresAt        *time.Time      `json:"expiresAt,omitempty"`
	CompletedAt      *time.Time      `json:"completedAt,omitempty"`
	CreatedAt        *time.Time      `json:"createdAt,omitempty"`
}

type FossapayCheckoutResponse struct {
	Status     string               `json:"status"`
	StatusCode int                  `json:"statusCode"`
	Message    string               `json:"message"`
	Data       FossapayCheckoutData `json:"data"`
}

type FossapayWebhookPayload struct {
	EventID   string         `json:"eventId"`
	EventType string         `json:"eventType"`
	Data      map[string]any `json:"data"`
	Timestamp string         `json:"timestamp"`
}
