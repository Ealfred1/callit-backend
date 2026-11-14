package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DatabaseURL string
	JWTSecret   string

	PantaAPIBase       string
	PantaAPIKey        string
	PantaMock          bool
	PantaPrimaryFeeBps int // not published as a fixed value by Panta; PRD says "often 200bps" — confirm on live MarketConfig

	AnthropicAPIKey string
	AnthropicModel  string
	GroqAPIKey      string
	GroqModel       string
	AIMock          bool

	ResendAPIKey      string
	ResendSenderEmail string
	ResendSenderName  string

	FossapayAPIBase       string
	FossapayAPIKey        string
	FossapayWebhookSecret string
	FossapayMock          bool
	TreasurySecretKey     string // base58 or JSON array; enables automatic USDC payouts
	PayoutMaxUSDC         float64
	SolanaRPCURL          string

	StaticDir string // where generated share-card PNGs are written/served from
	PublicURL string // base URL this server is reachable at, for share links
}

func Load() *Config {
	// Overload, not Load: pm2 snapshots a process's OS environment at
	// `pm2 start` time and keeps reusing it across `pm2 restart`. deploy.sh
	// used to `source .env` into the shell before the first `pm2 start`, so
	// every var from that first deploy got baked into pm2's cached env —
	// godotenv.Load() never overrides an already-set OS var, so editing
	// .env and restarting silently kept using the stale value (caught when
	// PUBLIC_URL changes on the server didn't take effect). Overload makes
	// the file always win, matching what "edit .env, restart" should do.
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found — reading from environment")
	}

	return &Config{
		AppPort:     getEnv("APP_PORT", "8095"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://callit:secret@localhost:5434/callit_db?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),

		PantaAPIBase:       getEnv("PANTA_API_BASE", "https://live-api.panta.market/api/v1"),
		PantaAPIKey:        getEnv("PANTA_API_KEY", ""),
		PantaMock:          getEnv("PANTA_MOCK", "true") == "true",
		PantaPrimaryFeeBps: getEnvInt("PANTA_PRIMARY_FEE_BPS", 200),

		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicModel:  getEnv("ANTHROPIC_MODEL", "claude-sonnet-5"),
		GroqAPIKey:      getEnv("GROQ_API_KEY", ""),
		GroqModel:       getEnv("GROQ_MODEL", "qwen/qwen3.8-27b"),
		AIMock:          getEnv("AI_MOCK", "true") == "true",

		ResendAPIKey:      getEnv("RESEND_API_KEY", ""),
		ResendSenderEmail: getEnv("RESEND_SENDER_EMAIL", "onboarding@resend.dev"),
		ResendSenderName:  getEnv("RESEND_SENDER_NAME", "Callit"),

		FossapayAPIBase:       getEnv("FOSSAPAY_API_BASE", "https://api-production.fossapay.com/api/v1"),
		FossapayAPIKey:        getEnv("FOSSAPAY_API_KEY", ""),
		FossapayWebhookSecret: getEnv("FOSSAPAY_WEBHOOK_SECRET", ""),
		FossapayMock:          getEnv("FOSSAPAY_MOCK", "false") == "true",
		TreasurySecretKey:     getEnv("TREASURY_SECRET_KEY", ""),
		PayoutMaxUSDC:         getEnvFloat("PAYOUT_MAX_USDC", 100),
		SolanaRPCURL:          getEnv("SOLANA_RPC_URL", "https://solana-rpc.publicnode.com"),

		StaticDir: getEnv("STATIC_DIR", "./static"),
		PublicURL: getEnv("PUBLIC_URL", "http://localhost:8095"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
