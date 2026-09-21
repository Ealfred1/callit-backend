package config

import (
	"log"
	"os"

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
	AIMock          bool

	ResendAPIKey      string
	ResendSenderEmail string
	ResendSenderName  string

	StaticDir string // where generated share-card PNGs are written/served from
	PublicURL string // base URL this server is reachable at, for share links
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
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
		AIMock:          getEnv("AI_MOCK", "true") == "true",

		ResendAPIKey:      getEnv("RESEND_API_KEY", ""),
		ResendSenderEmail: getEnv("RESEND_SENDER_EMAIL", "onboarding@resend.dev"),
		ResendSenderName:  getEnv("RESEND_SENDER_NAME", "Callit"),

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
