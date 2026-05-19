package router

import (
	"net/http"

	adminHdl "github.com/prast13/bayaraman/internal/handler/admin"
	authHdl "github.com/prast13/bayaraman/internal/handler/auth"
	chatHdl "github.com/prast13/bayaraman/internal/handler/chat"
	escrowHdl "github.com/prast13/bayaraman/internal/handler/escrow"
	userHdl "github.com/prast13/bayaraman/internal/handler/user"
	walletHdl "github.com/prast13/bayaraman/internal/handler/wallet"
	webhookHdl "github.com/prast13/bayaraman/internal/handler/webhook"

	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/config"
	"github.com/redis/go-redis/v9"

	authMiddleware "github.com/prast13/bayaraman/internal/middleware"
)

type RouterParams struct {
	Echo           *echo.Echo
	Config         *config.Config
	AuthHandler    *authHdl.AuthHandler
	UserHandler    *userHdl.UserHandler
	WebhookHandler *webhookHdl.WebhookHandler
	EscrowHandler  *escrowHdl.EscrowHandler
	WalletHandler  *walletHdl.WalletHandler
	AdminHandler   *adminHdl.AdminHandler
	ChatHandler    *chatHdl.ChatHandler
	RedisClient    *redis.Client
}

func SetupRoutes(p RouterParams) {
	e := p.Echo

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Webhooks (Unprotected)
	webhooks := e.Group("/webhooks")
	{
		webhooks.POST("/privy", p.WebhookHandler.PrivyWebhook)
		webhooks.POST("/xendit", p.WebhookHandler.XenditWebhook)
	}

	// Static file handler for uploads
	e.Static("/uploads", "uploads")

	api := e.Group("/api/v1")
	{
		// Auth Routes
		auth := api.Group("/auth")
		auth.POST("/register", p.AuthHandler.Register)
		auth.POST("/verify-email", p.AuthHandler.VerifyEmail)
		auth.POST("/login", p.AuthHandler.Login)
		auth.POST("/refresh", p.AuthHandler.Refresh)
		auth.POST("/logout", p.AuthHandler.Logout)

		// User Routes (Protected)
		user := api.Group("/user")
		user.Use(authMiddleware.RequireAuth(p.Config.JWTSecret, authMiddleware.WithTokenBlacklist(p.RedisClient)))
		user.POST("/kyc/initiate", p.UserHandler.InitiateKYC)

		// Escrow Routes (Protected)
		escrowGroup := api.Group("/escrows")
		escrowGroup.Use(authMiddleware.RequireAuth(p.Config.JWTSecret, authMiddleware.WithTokenBlacklist(p.RedisClient)))
		escrowGroup.POST("/", p.EscrowHandler.Create)
		escrowGroup.GET("/", p.EscrowHandler.MyEscrows)
		
		idemp := authMiddleware.Idempotency(p.RedisClient)
		
		escrowGroup.POST("/:id/fund", p.EscrowHandler.Fund, idemp)
		escrowGroup.POST("/:id/complete", p.EscrowHandler.Complete, idemp)
		escrowGroup.POST("/:id/videos/packing", p.EscrowHandler.UploadPackingVideo)
		escrowGroup.POST("/:id/videos/unboxing", p.EscrowHandler.UploadUnboxingVideo)
		escrowGroup.POST("/:id/photos/packing", p.EscrowHandler.UploadPackingPhoto)
		escrowGroup.POST("/:id/photos/unboxing", p.EscrowHandler.UploadUnboxingPhoto)
		escrowGroup.POST("/:id/receipt", p.EscrowHandler.UploadReceipt)
		escrowGroup.POST("/:id/deliver", p.EscrowHandler.DeliverEscrow)

		escrowGroup.GET("/:id/chat/history", p.ChatHandler.GetHistory)
		escrowGroup.POST("/:id/chat/image", p.ChatHandler.UploadImage)

		// Wallet Routes (Protected)
		walletGroup := api.Group("/wallets")
		walletGroup.Use(authMiddleware.RequireAuth(p.Config.JWTSecret, authMiddleware.WithTokenBlacklist(p.RedisClient)))
		walletGroup.GET("/me", p.WalletHandler.GetMyWallet)
		walletGroup.POST("/withdraw", p.WalletHandler.Withdraw, idemp)

		// Admin Routes (Protected + Role Admin)
		adminGroup := api.Group("/admin")
		adminGroup.Use(authMiddleware.RequireAuth(p.Config.JWTSecret, authMiddleware.WithTokenBlacklist(p.RedisClient)))
		adminGroup.Use(authMiddleware.RequireRole("admin"))
		
		adminGroup.GET("/users", p.AdminHandler.GetUsers)
		adminGroup.GET("/users/:id", p.AdminHandler.GetUserByID)
		adminGroup.POST("/users/:id/suspend", p.AdminHandler.SuspendUser)
		
		adminGroup.POST("/escrows/:id/freeze", p.AdminHandler.FreezeEscrow)
		adminGroup.POST("/escrows/:id/disputes/override", p.AdminHandler.OverrideDispute)
		adminGroup.GET("/escrows/:id/timeline", p.AdminHandler.GetEscrowTimeline)
		
		adminGroup.POST("/payouts/:id/retry", p.AdminHandler.RetryPayout)

		adminGroup.GET("/configs", p.AdminHandler.GetConfigs)
		adminGroup.PUT("/configs", p.AdminHandler.UpdateConfig)
		
		// Chat WS Route
		api.GET("/chat/ws", p.ChatHandler.ConnectWS)
	}
}
