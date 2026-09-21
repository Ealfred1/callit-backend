package domain

// Field names below are transcribed from https://docs.panta.market (fetched
// 2026-09-21) — the request/response shapes shown on each endpoint's docs
// page, not guessed. See docs/NOTES.md for which endpoints were confirmed
// via a literal curl example vs inferred from the doc's slug/pattern.

type Category string

const (
	CategorySports        Category = "sports"
	CategoryCrypto        Category = "crypto"
	CategoryPolitics      Category = "politics"
	CategoryEntertainment Category = "entertainment"
	CategoryFinance       Category = "finance"
	CategoryScience       Category = "science"
	CategoryWorld         Category = "world"
	CategoryOther         Category = "other"
)

type Side string

const (
	SideYes Side = "yes"
	SideNo  Side = "no"
)

// --- Create market pipeline ---

type CreateQuoteRequest struct {
	Wallet          string   `json:"wallet"`
	Question        string   `json:"question"`
	ResolutionRule  string   `json:"resolutionRule"`
	SourcesOfTruth  []string `json:"sourcesOfTruth"`
	Category        Category `json:"category"`
	StartTime       int64    `json:"startTime"`
	EndTime         int64    `json:"endTime"`
	ResolutionTime  int64    `json:"resolutionTime"`
	ImageURL        string   `json:"imageUrl"`
	MarketType      string   `json:"marketType,omitempty"`      // "standard" | "breaking"
	EventInProgress bool     `json:"eventInProgress,omitempty"` // breaking markets only
	Title           string   `json:"title,omitempty"`
	Description     string   `json:"description,omitempty"`
	Region          string   `json:"region,omitempty"`
	Oracle          string   `json:"oracle,omitempty"`
}

type CreateQuoteResponse struct {
	CreateID                string `json:"createId"`
	ExpectedEventPda        string `json:"expectedEventPda"`
	PaymentUsdc             string `json:"paymentUsdc"` // base units, 6 decimals
	LiquidityInjectionUsdc  string `json:"liquidityInjectionUsdc"`
	PlatformRevenueUsdc     string `json:"platformRevenueUsdc"`
	MarketType              string `json:"marketType"`
	ExpiresAt                string `json:"expiresAt"` // ISO-8601, ~5 min out
	BlockhashExpiryHintSec   int    `json:"blockhashExpiryHintSec"`
}

type CreateBuildRequest struct {
	CreateID string `json:"createId"`
	Wallet   string `json:"wallet,omitempty"`
}

type CreateBuildDerived struct {
	Event         string `json:"event"`
	VaultAuthority string `json:"vaultAuthority"`
	MarketConfig  string `json:"marketConfig"`
}

type CreateBuildResponse struct {
	CreateID               string             `json:"createId"`
	ExpectedEventPda       string             `json:"expectedEventPda"`
	Transaction             string             `json:"transaction"` // base64 VersionedTransaction
	RecentBlockhash         string             `json:"recentBlockhash"`
	LastValidBlockHeight    int64              `json:"lastValidBlockHeight"`
	BlockhashExpiryHintSec  int                `json:"blockhashExpiryHintSec"`
	BuildFingerprint        string             `json:"buildFingerprint"`
	PaymentUsdc              string             `json:"paymentUsdc"`
	LiquidityInjectionUsdc   string             `json:"liquidityInjectionUsdc"`
	PlatformRevenueUsdc      string             `json:"platformRevenueUsdc"`
	MarketType                string             `json:"marketType"`
	Derived                    CreateBuildDerived `json:"derived"`
	ExpiresAt                   string             `json:"expiresAt"`
}

type RegisterMarketRequest struct {
	CreateID  string `json:"createId"`
	Signature string `json:"signature"`
}

type RegisterMarketResponse struct {
	CreateID  string   `json:"createId"`
	MarketID  string   `json:"marketId"`
	Status    string   `json:"status"`
	Signature string   `json:"signature"`
	Category  string   `json:"category"`
	Title     string   `json:"title"`
	Images    []string `json:"images"`
}

type ImageUploadSignResponse struct {
	UploadURL string            `json:"uploadUrl"`
	PublicID  string            `json:"publicId"`
	ExpiresAt string            `json:"expiresAt"`
	Fields    map[string]string `json:"fields"`
}

// --- Catalog ---

type MarketListItem struct {
	MarketID           string   `json:"marketId"`
	Category           Category `json:"category"`
	Title              string   `json:"title"`
	Description        *string  `json:"description"`
	Images             []string `json:"images"`
	Phase              string   `json:"phase"` // primary | secondary | resolved | cancelled
	MarketType         string   `json:"marketType"`
	StartTime          int64    `json:"startTime"`
	EndTime            int64    `json:"endTime"`
	ResolutionTime     int64    `json:"resolutionTime"`
	Region             string   `json:"region"`
	Resolved           bool     `json:"resolved"`
	VolumeUsdc         string   `json:"volumeUsdc"`
	CampaignID         *string  `json:"campaignId"`
	YesPrice           *string  `json:"yesPrice"`
	NoPrice            *string  `json:"noPrice"`
	PrimaryYesPrice    *string  `json:"primaryYesPrice"`
	PrimaryNoPrice     *string  `json:"primaryNoPrice"`
	SecondaryYesPrice  *string  `json:"secondaryYesPrice"`
	SecondaryNoPrice   *string  `json:"secondaryNoPrice"`
}

type MarketDetail struct {
	MarketListItem
	Status            string `json:"status"`
	CreatedByPartner  bool   `json:"createdByPartner"`
}

type MarketListResponse struct {
	Items      []MarketListItem `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type MarketListQuery struct {
	Category  string
	Status    string
	CreatedBy string
	Cursor    string
	Limit     string
}

// --- Primary buy pipeline ---

type OrderQuoteRequest struct {
	Wallet     string `json:"wallet"`
	MarketID   string `json:"marketId"`
	Side       Side   `json:"side"`
	AmountUsdc string `json:"amountUsdc"` // human-readable decimal e.g. "20.00"
	UserID     string `json:"userId,omitempty"`
}

type OrderQuoteResponse struct {
	QuoteID               string `json:"quoteId"`
	MarketID               string `json:"marketId"`
	Side                    Side   `json:"side"`
	AmountUsdc               string `json:"amountUsdc"`
	Shares                   string `json:"shares"`
	AvgPrice                 string `json:"avgPrice"`
	FeeUsdc                  string `json:"feeUsdc"`
	ExpiresAt                 string `json:"expiresAt"` // ~90s out
	BlockhashExpiryHintSec     int    `json:"blockhashExpiryHintSec"`
}

type OrderBuildRequest struct {
	QuoteID        string `json:"quoteId"`
	Wallet         string `json:"wallet"`
	UserID         string `json:"userId,omitempty"`
	MaxSlippageBps int    `json:"maxSlippageBps,omitempty"` // default 100, max 5000
}

type OrderInstructionAccount struct {
	Pubkey     string `json:"pubkey"`
	IsSigner   bool   `json:"isSigner"`
	IsWritable bool   `json:"isWritable"`
}

type OrderInstruction struct {
	ProgramID string                     `json:"programId"`
	Data      string                     `json:"data"` // base64
	Accounts  []OrderInstructionAccount `json:"accounts"`
}

type OrderBuildDerived struct {
	Event          string `json:"event"`
	VaultAuthority string `json:"vaultAuthority"`
}

type OrderBuildResponse struct {
	OrderID                string             `json:"orderId"`
	QuoteID                 string             `json:"quoteId"`
	Wallet                   string             `json:"wallet"`
	MarketID                  string             `json:"marketId"`
	Side                       Side               `json:"side"`
	AmountUsdc                  string             `json:"amountUsdc"`
	ExpectedShares                string             `json:"expectedShares"`
	FeeUsdc                        string             `json:"feeUsdc"`
	Status                          string             `json:"status"`
	Instructions                     []OrderInstruction `json:"instructions"`
	Derived                            OrderBuildDerived  `json:"derived"`
	RecentBlockhash                     string             `json:"recentBlockhash"`
	LastValidBlockHeight                  int64              `json:"lastValidBlockHeight"`
	ExpiresAt                               string             `json:"expiresAt"`
	BlockhashExpiryHintSec                    int                `json:"blockhashExpiryHintSec"`
}

type OrderSubmitRequest struct {
	OrderID   string `json:"orderId"`
	Signature string `json:"signature"`
	Wallet    string `json:"wallet,omitempty"`
}

type OrderSubmitResponse struct {
	OrderID   string `json:"orderId"`
	Status    string `json:"status"` // "submitted"
	Signature string `json:"signature"`
}

type OrderVerifyRequest struct {
	OrderID   string `json:"orderId"`
	Signature string `json:"signature,omitempty"`
	Wallet    string `json:"wallet,omitempty"`
}

type OrderVerifyResponse struct {
	OrderID    string  `json:"orderId"`
	Status     string  `json:"status"` // built | submitted | confirmed | failed | expired
	Signature  *string `json:"signature"`
	MarketID   string  `json:"marketId"`
	Side       Side    `json:"side"`
	AmountUsdc float64 `json:"amountUsdc"`
}

// --- Positions & claims ---

type Position struct {
	MarketID  string   `json:"marketId"`
	Category  Category `json:"category"`
	Side      Side     `json:"side"`
	Shares    string   `json:"shares"`
	Phase     string   `json:"phase"`
	Claimable bool     `json:"claimable"`
	Claimed   bool     `json:"claimed"`
	Outcome   *Side    `json:"outcome"`
}

type PositionsResponse struct {
	Wallet    string     `json:"wallet"`
	Positions []Position `json:"positions"`
}

type ClaimRequest struct {
	Wallet   string `json:"wallet"`
	MarketID string `json:"marketId"`
}

type ClaimBuildDerived struct {
	WinClaim       string `json:"winClaim"`
	PositionPda    string `json:"positionPda"`
	VaultAuthority string `json:"vaultAuthority"`
}

type ClaimBuildResponse struct {
	Outcome              Side               `json:"outcome"`
	WinningShares        string             `json:"winningShares"`
	Instructions         []OrderInstruction `json:"instructions"`
	Derived              ClaimBuildDerived  `json:"derived"`
	RecentBlockhash      string             `json:"recentBlockhash"`
	LastValidBlockHeight int64              `json:"lastValidBlockHeight"`
}

type ClaimCreatorFeesDerived struct {
	CreatorFeeVault string `json:"creatorFeeVault"`
	MarketConfig    string `json:"marketConfig"`
}

type ClaimCreatorFeesResponse struct {
	ClaimableFeesUsdc    string                  `json:"claimableFeesUsdc"`
	Instructions         []OrderInstruction      `json:"instructions"`
	Derived              ClaimCreatorFeesDerived `json:"derived"`
	RecentBlockhash      string                  `json:"recentBlockhash"`
	LastValidBlockHeight int64                   `json:"lastValidBlockHeight"`
}

// --- Trade attribution ---

type ReportTradeRequest struct {
	Signature     string `json:"signature"`
	Wallet        string `json:"wallet"`
	MarketID      string `json:"marketId"`
	QuoteID       string `json:"quoteId,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
	UserID        string `json:"userId,omitempty"`
}

type ReportTradeResponse struct {
	Signature string `json:"signature"`
	Status    string `json:"status"` // "processed"
	MarketID  string `json:"marketId"`
	Wallet    string `json:"wallet"`
	Side      Side   `json:"side"`
	Kind      string `json:"kind"` // buy | claim
}

type TradeStatusResponse struct {
	Signature string  `json:"signature"`
	Status    string  `json:"status"` // processed | pending_attribution | unknown | failed
	MarketID  *string `json:"marketId,omitempty"`
	Wallet    *string `json:"wallet,omitempty"`
}

// --- Errors ---

// PantaError mirrors Panta's own error envelope ({code,message,field|fields})
// so a passthrough failure keeps its original code — see the unified
// response envelope in internal/adapter/handler/respond.
type PantaError struct {
	HTTPStatus int
	Code       string              `json:"code"`
	Message    string              `json:"message"`
	Field      string              `json:"field,omitempty"`
	Fields     map[string][]string `json:"fields,omitempty"`
}

func (e *PantaError) Error() string { return e.Message }
