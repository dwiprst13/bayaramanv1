package user

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) InitiateKYC(c echo.Context) error {
	// In a real scenario, this would call Privy's API to get a registration token/URL
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "KYC initiation successful",
		"kyc_url": "https://registration.privy.id/mock-url-for-user-" + userID.String(),
	})
}
