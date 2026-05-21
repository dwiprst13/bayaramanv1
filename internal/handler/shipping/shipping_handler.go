package shipping

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prast13/bayaraman/internal/model"
	shippingSvc "github.com/prast13/bayaraman/internal/service/shipping"
)

type ShippingHandler struct {
	shippingService shippingSvc.ShippingService
}

func NewShippingHandler(shippingService shippingSvc.ShippingService) *ShippingHandler {
	return &ShippingHandler{shippingService: shippingService}
}

// GetRates handles POST /api/v1/shipping/rates
func (h *ShippingHandler) GetRates(c echo.Context) error {
	var req model.ShippingRateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	rates, err := h.shippingService.GetRates(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": rates,
	})
}

// TrackShipment handles GET /api/v1/escrows/:id/shipping/track
func (h *ShippingHandler) TrackShipment(c echo.Context) error {
	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid escrow id"})
	}

	result, err := h.shippingService.TrackShipment(c.Request().Context(), escrowID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": result,
	})
}

// TrackingWebhook handles POST /webhooks/shipping
func (h *ShippingHandler) TrackingWebhook(c echo.Context) error {
	var payload struct {
		TrackingNumber string `json:"tracking_number"`
		CourierCode    string `json:"courier_code"`
		Status         string `json:"status"`
	}

	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}

	if payload.TrackingNumber == "" || payload.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tracking_number and status are required"})
	}

	err := h.shippingService.ProcessTrackingWebhook(c.Request().Context(), payload.TrackingNumber, payload.CourierCode, payload.Status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "tracking update processed"})
}
