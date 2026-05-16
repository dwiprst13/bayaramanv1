package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/internal/repository"
)

type WebhookHandler struct {
	userRepo           repository.UserRepository
	privyWebhookSecret string
}

func NewWebhookHandler(userRepo repository.UserRepository, privyWebhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		userRepo:           userRepo,
		privyWebhookSecret: privyWebhookSecret,
	}
}

type PrivyWebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		Email   string `json:"email"`
		PrivyID string `json:"privy_id"`
		Status  string `json:"status"` // verified, rejected
	} `json:"data"`
}

func (h *WebhookHandler) PrivyWebhook(c echo.Context) error {
	// Simple Secret Check (Can be HMAC signature depending on Privy's exact spec)
	token := c.Request().Header.Get("X-Webhook-Token")
	if h.privyWebhookSecret != "" && token != h.privyWebhookSecret {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized webhook"})
	}

	var payload PrivyWebhookPayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if payload.Event != "kyc_status_updated" {
		return c.JSON(http.StatusOK, map[string]string{"message": "ignored event"})
	}

	// Find user by email
	user, err := h.userRepo.FindByEmail(context.Background(), payload.Data.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Update KYC Status
	user.KYCStatus = payload.Data.Status
	user.PrivyID = &payload.Data.PrivyID
	
	err = h.userRepo.Update(context.Background(), user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "kyc status updated successfully"})
}
