package domain

import "time"

// NearIntentToken represents a token supported by the 1Click NEAR Intents protocol.
type NearIntentToken struct {
	AssetID         string  `json:"assetId"`
	Symbol          string  `json:"symbol"`
	Blockchain      string  `json:"blockchain"`
	Decimals        int     `json:"decimals"`
	PriceUSD        float64 `json:"priceUsd"`
	ContractAddress string  `json:"contractAddress,omitempty"`
}

// NearIntentQuoteReq represents a quote request to deposit funds via NEAR Intents.
type NearIntentQuoteReq struct {
	OriginAsset       string `json:"originAsset"`
	DestinationAsset  string `json:"destinationAsset,omitempty"` // defaults to solana usdc/sol
	Amount            string `json:"amount"`                   // formatted or base units
	AmountFormatted   string `json:"amountFormatted,omitempty"`
	Recipient         string `json:"recipient"`                // user's Callit Solana or NEAR address
	RefundTo          string `json:"refundTo,omitempty"`
	SlippageTolerance int    `json:"slippageTolerance,omitempty"` // in basis points, default 100 (1%)
	Dry               bool   `json:"dry"`                         // true for quote preview, false for active deposit address
}

// NearIntentQuoteRes represents the quote & generated deposit address from 1Click.
type NearIntentQuoteRes struct {
	DepositAddress     string    `json:"depositAddress,omitempty"`
	OriginAsset        string    `json:"originAsset"`
	DestinationAsset   string    `json:"destinationAsset"`
	AmountIn           string    `json:"amountIn"`
	AmountInFormatted  string    `json:"amountInFormatted"`
	AmountInUSD        string    `json:"amountInUsd"`
	AmountOut          string    `json:"amountOut"`
	AmountOutFormatted string    `json:"amountOutFormatted"`
	AmountOutUSD       string    `json:"amountOutUsd"`
	TimeEstimateSec    int       `json:"timeEstimateSec"`
	Deadline           time.Time `json:"deadline"`
	Status             string    `json:"status"`
	CorrelationID      string    `json:"correlationId,omitempty"`
}

// NearIntentStatusRes represents the execution status of a NEAR Intents deposit.
type NearIntentStatusRes struct {
	Status        string  `json:"status"` // PENDING_DEPOSIT, PROCESSING, COMPLETED, REFUNDED, FAILED
	DepositAddress string `json:"depositAddress"`
	DepositedAmount string `json:"depositedAmount,omitempty"`
	AmountOut       string `json:"amountOut,omitempty"`
	NearTxHashes    []string `json:"nearTxHashes,omitempty"`
	OriginTxHashes  []string `json:"originTxHashes,omitempty"`
	DestTxHashes    []string `json:"destTxHashes,omitempty"`
	UpdatedAt       string `json:"updatedAt"`
}
