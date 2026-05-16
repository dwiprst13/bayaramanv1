package main

import (
	"log"

	authHdl "github.com/prast13/bayaraman/internal/handler/auth"
	escrowHdl "github.com/prast13/bayaraman/internal/handler/escrow"
	userHdl "github.com/prast13/bayaraman/internal/handler/user"
	webhookHdl "github.com/prast13/bayaraman/internal/handler/webhook"
	auditLogRepo "github.com/prast13/bayaraman/internal/repository/auditlog"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"
	paymentRepo "github.com/prast13/bayaraman/internal/repository/payment"
	sessionRepo "github.com/prast13/bayaraman/internal/repository/session"
	userRepo "github.com/prast13/bayaraman/internal/repository/user"
	authSvc "github.com/prast13/bayaraman/internal/service/auth"
	emailSvc "github.com/prast13/bayaraman/internal/service/email"
	escrowSvc "github.com/prast13/bayaraman/internal/service/escrow"
	otpSvc "github.com/prast13/bayaraman/internal/service/otp"
	paymentSvc "github.com/prast13/bayaraman/internal/service/payment"
	rateLimiterSvc "github.com/prast13/bayaraman/internal/service/ratelimiter"
	storageSvc "github.com/prast13/bayaraman/internal/service/storage"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prast13/bayaraman/config"
	"github.com/prast13/bayaraman/internal/router"
	"github.com/prast13/bayaraman/internal/worker"
)

func main() {
	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize Database
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis
	redisClient, err := config.InitRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	_ = redisClient

	// Repositories
	userRepo := userRepo.NewUserRepository(db)
	sessionRepo := sessionRepo.NewSessionRepository(db)
	auditLogRepo := auditLogRepo.NewAuditLogRepository(db)
	escrowRepo := escrowRepo.NewEscrowRepository(db)
	paymentRepo := paymentRepo.NewPaymentRepository(db)

	// Services
	emailService := emailSvc.NewEmailService()
	rateLimiterService := rateLimiterSvc.NewRateLimiterService(redisClient)
	
	// Setup storage base URL
	baseURL := "http://localhost:8080/uploads"
	if cfg.Port != "" {
		baseURL = "http://localhost:" + cfg.Port + "/uploads"
	}
	storageService := storageSvc.NewLocalStorageService("./uploads", baseURL)

	otpService := otpSvc.NewOTPService(redisClient, emailService, rateLimiterService)
	authService := authSvc.NewAuthService(userRepo, sessionRepo, auditLogRepo, otpService, redisClient)
	paymentService := paymentSvc.NewPaymentService(paymentRepo, escrowRepo, auditLogRepo, cfg.XenditAPIKey)
	escrowService := escrowSvc.NewEscrowService(escrowRepo, paymentService, auditLogRepo, storageService)

	// Handlers
	authHandler := authHdl.NewAuthHandler(authService, cfg.JWTSecret)
	webhookHandler := webhookHdl.NewWebhookHandler(userRepo, paymentService, cfg.PrivyWebhookSecret, cfg.XenditWebhookToken)
	userHandler := userHdl.NewUserHandler()
	escrowHandler := escrowHdl.NewEscrowHandler(escrowService)

	// Setup Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	router.SetupRoutes(router.RouterParams{
		Echo:           e,
		Config:         cfg,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		WebhookHandler: webhookHandler,
		EscrowHandler:  escrowHandler,
	})

	// Start Background Worker
	go worker.StartVideoCleanupWorker(db, storageService)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
