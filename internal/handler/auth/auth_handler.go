package auth

import (
	"net/http"
	"strings"

	authSvc "github.com/prast13/bayaraman/internal/service/auth"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService authSvc.AuthService
	jwtSecret   string
}

func NewAuthHandler(authService authSvc.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{authService: authService, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req authSvc.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	user, err := h.authService.Register(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully. Please check your email for OTP.",
		"user_id": user.ID,
		"email":   user.Email,
	})
}

type verifyRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req verifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	err := h.authService.VerifyEmail(c.Request().Context(), req.Email, req.OTP)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req authSvc.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	req.IPAddress = c.RealIP()
	req.UserAgent = c.Request().UserAgent()

	resp, err := h.authService.Login(c.Request().Context(), req, h.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"data":    resp,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	ip := c.RealIP()
	ua := c.Request().UserAgent()

	resp, err := h.authService.Refresh(c.Request().Context(), req.RefreshToken, ip, ua, h.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": resp,
	})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Extract access token from Authorization header for blacklisting
	accessToken := ""
	authHeader := c.Request().Header.Get("Authorization")
	if parts := strings.Split(authHeader, " "); len(parts) == 2 && parts[0] == "Bearer" {
		accessToken = parts[1]
	}

	err := h.authService.Logout(c.Request().Context(), req.RefreshToken, accessToken)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}
