package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/config"
	"github.com/prast13/bayaraman/internal/handler"
	authMiddleware "github.com/prast13/bayaraman/internal/middleware"
)

type RouterParams struct {
	Echo           *echo.Echo
	Config         *config.Config
	AuthHandler    *handler.AuthHandler
	UserHandler    *handler.UserHandler
	WebhookHandler *handler.WebhookHandler
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
	}

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
	}
}
