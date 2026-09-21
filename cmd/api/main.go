package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eric/callit-backend/internal/adapter/ai"
	"github.com/eric/callit-backend/internal/adapter/handler"
	authmw "github.com/eric/callit-backend/internal/adapter/handler/middleware"
	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/adapter/mailer"
	"github.com/eric/callit-backend/internal/adapter/panta"
	"github.com/eric/callit-backend/internal/adapter/repository"
	"github.com/eric/callit-backend/internal/config"
	"github.com/eric/callit-backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = respond.GlobalErrorHandler
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	// ── Database ─────────────────────────────────────────────────────────
	dbPool, err := repository.NewPostgresDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("Database connected")

	// ── Repositories ─────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(dbPool)
	communityRepo := repository.NewCommunityRepository(dbPool)
	createSessionRepo := repository.NewCreateSessionRepository(dbPool)
	marketMetaRepo := repository.NewMarketMetaRepository(dbPool)

	// ── Email (real Resend if configured, else logs the code) ──────────
	var emailService = mailer.NewMockEmailService(logger)
	if cfg.ResendAPIKey != "" {
		emailService = mailer.NewResendEmailService(cfg.ResendAPIKey, cfg.ResendSenderEmail, cfg.ResendSenderName, logger)
		logger.Info("Resend email service active")
	} else {
		logger.Warn("Mock email service active — set RESEND_API_KEY for production")
	}

	// ── Panta client (real if PANTA_MOCK=false, else in-memory fixtures) ─
	pantaClient := panta.NewMockClient()
	if !cfg.PantaMock {
		pantaClient = panta.NewRealClient(cfg.PantaAPIBase, cfg.PantaAPIKey)
		logger.Info("Panta client: live", "base", cfg.PantaAPIBase)
	} else {
		logger.Warn("Panta client: MOCK — set PANTA_MOCK=false with a real PANTA_API_KEY for production")
	}

	// ── AI compiler (real Anthropic if AI_MOCK=false, else deterministic) ─
	compiler := ai.NewMockCompiler()
	if !cfg.AIMock {
		compiler = ai.NewAnthropicCompiler(cfg.AnthropicAPIKey, cfg.AnthropicModel)
		logger.Info("AI compiler: live", "model", cfg.AnthropicModel)
	} else {
		logger.Warn("AI compiler: MOCK — set AI_MOCK=false with a real ANTHROPIC_API_KEY for production")
	}

	// ── Services ─────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, emailService)
	communitySvc := service.NewCommunityService(communityRepo, marketMetaRepo, pantaClient)
	marketCreateSvc := service.NewMarketCreateService(pantaClient, compiler, createSessionRepo, marketMetaRepo, communityRepo)
	orderSvc := service.NewOrderService(pantaClient, communitySvc)
	shareCardSvc := service.NewShareCardService(pantaClient, marketMetaRepo, cfg.StaticDir, cfg.PublicURL)

	// ── Handlers ─────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(authSvc, cfg.JWTSecret)
	marketsH := handler.NewMarketsHandler(pantaClient)
	marketCreateH := handler.NewMarketCreateHandler(marketCreateSvc, pantaClient)
	orderH := handler.NewOrderHandler(orderSvc)
	positionsH := handler.NewPositionsHandler(pantaClient)
	claimsH := handler.NewClaimsHandler(pantaClient)
	tradesH := handler.NewTradesHandler(pantaClient, communitySvc)
	communityH := handler.NewCommunityHandler(communitySvc)
	shareCardH := handler.NewShareCardHandler(shareCardSvc)
	meH := handler.NewMeHandler(marketMetaRepo, pantaClient)

	// ── Static + docs + health ───────────────────────────────────────────
	e.Static("/static", cfg.StaticDir)
	e.File("/docs/openapi.yaml", "docs/openapi.yaml")
	e.GET("/docs", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `<!doctype html><html><head><title>Callit API</title>
<meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>
<style>body{margin:0}</style></head><body>
<script id="api-reference" data-url="/docs/openapi.yaml"></script>
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body></html>`)
	})
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// ── Routes ───────────────────────────────────────────────────────────
	v1 := e.Group("/v1")
	limiter := rate.NewLimiter(rate.Limit(20), 50)
	v1.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}
			return next(c)
		}
	})

	// Public
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/verify", authH.Verify)
	v1.POST("/auth/login", authH.Login)
	v1.GET("/markets", marketsH.List)
	v1.GET("/markets/:id", marketsH.Get)
	v1.GET("/markets/:id/trades", marketsH.Trades)
	v1.GET("/wallets/:wallet/trades", marketsH.WalletTrades)
	v1.GET("/categories", marketsH.Categories)
	v1.GET("/communities", communityH.List)
	v1.GET("/communities/:id", communityH.Get)
	v1.GET("/share/:marketId", shareCardH.Unfurl)

	// Protected
	protected := v1.Group("")
	protected.Use(authmw.JWTMiddleware(cfg.JWTSecret))

	protected.GET("/me", authH.Me)
	protected.GET("/me/markets", meH.Markets)

	protected.POST("/ai/compile", marketCreateH.AICompile)
	protected.POST("/markets/create/image-upload", marketCreateH.ImageUpload)
	protected.POST("/markets/create/quote", marketCreateH.Quote)
	protected.POST("/markets/create/build", marketCreateH.Build)
	protected.POST("/markets/create/register", marketCreateH.Register)

	protected.POST("/orders/quote", orderH.Quote)
	protected.POST("/orders/build", orderH.Build)
	protected.POST("/orders/submit", orderH.Submit)
	protected.POST("/orders/verify", orderH.Verify)

	protected.GET("/positions", positionsH.Get)
	protected.POST("/claims/win", claimsH.Win)
	protected.POST("/claims/creator-fees", claimsH.CreatorFees)

	protected.POST("/trades/report", tradesH.Report)
	protected.GET("/trades/:signature", tradesH.Status)

	protected.POST("/communities", communityH.Create)
	protected.POST("/share-card", shareCardH.Render)

	// ── Start + graceful shutdown ────────────────────────────────────────
	shutCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := e.Start(":" + cfg.AppPort); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()
	logger.Info("Callit API started", "port", cfg.AppPort)

	<-shutCtx.Done()
	logger.Info("Shutdown signal received")

	graceCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(graceCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("Server exited cleanly")
}
