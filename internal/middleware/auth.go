package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

func RequireAuth(jwtSecret string, opts ...AuthOption) echo.MiddlewareFunc {
	cfg := &authConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing Authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid Authorization header format"})
			}

			tokenString := parts[1]

			// Check if token has been blacklisted (logout)
			if cfg.redisClient != nil {
				blacklistKey := "token_blacklist:" + tokenString
				val, err := cfg.redisClient.Get(c.Request().Context(), blacklistKey).Result()
				if err == nil && val != "" {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Token has been revoked"})
				}
			}

			claims, err := jwt.ParseToken(tokenString, jwtSecret)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
			}

			// Inject user info to context
			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}

// AuthOption configures the RequireAuth middleware.
type AuthOption func(*authConfig)

type authConfig struct {
	redisClient *redis.Client
}

// WithTokenBlacklist enables access token blacklist checking via Redis.
func WithTokenBlacklist(redisClient *redis.Client) AuthOption {
	return func(cfg *authConfig) {
		cfg.redisClient = redisClient
	}
}
