package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prast13/bayaraman/config"
	"github.com/prast13/bayaraman/internal/handler"
	"github.com/prast13/bayaraman/internal/repository"
	"github.com/prast13/bayaraman/internal/router"
	"github.com/prast13/bayaraman/internal/service"
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
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)

	// Services
	emailService := service.NewEmailService()
	otpService := service.NewOTPService(redisClient, emailService)
	authService := service.NewAuthService(userRepo, sessionRepo, auditLogRepo, otpService, redisClient)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, cfg.JWTSecret)
	webhookHandler := handler.NewWebhookHandler(userRepo, cfg.PrivyWebhookSecret)
	userHandler := handler.NewUserHandler()

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
	})

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
