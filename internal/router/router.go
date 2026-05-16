package router

import (
	"net/http"

	authHdl "github.com/prast13/bayaraman/internal/handler/auth"
	escrowHdl "github.com/prast13/bayaraman/internal/handler/escrow"
	userHdl "github.com/prast13/bayaraman/internal/handler/user"
	webhookHdl "github.com/prast13/bayaraman/internal/handler/webhook"

	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/config"

	authMiddleware "github.com/prast13/bayaraman/internal/middleware"
)

type RouterParams struct {
	Echo           *echo.Echo
	Config         *config.Config
	AuthHandler    *authHdl.AuthHandler
	UserHandler    *userHdl.UserHandler
	WebhookHandler *webhookHdl.WebhookHandler
	EscrowHandler  *escrowHdl.EscrowHandler
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
		user.Use(authMiddleware.RequireAuth(p.Config.JWTSecret))
		user.POST("/kyc/initiate", p.UserHandler.InitiateKYC)

		// Escrow Routes (Protected)
		escrowGroup := api.Group("/escrows")
		escrowGroup.Use(authMiddleware.RequireAuth(p.Config.JWTSecret))
		escrowGroup.POST("/", p.EscrowHandler.Create)
		escrowGroup.GET("/", p.EscrowHandler.MyEscrows)
		escrowGroup.POST("/:id/fund", p.EscrowHandler.Fund)
		escrowGroup.POST("/:id/complete", p.EscrowHandler.Complete)
		escrowGroup.POST("/:id/videos/packing", p.EscrowHandler.UploadPackingVideo)
		escrowGroup.POST("/:id/videos/unboxing", p.EscrowHandler.UploadUnboxingVideo)
		escrowGroup.POST("/:id/photos/packing", p.EscrowHandler.UploadPackingPhoto)
		escrowGroup.POST("/:id/photos/unboxing", p.EscrowHandler.UploadUnboxingPhoto)
	}
}
