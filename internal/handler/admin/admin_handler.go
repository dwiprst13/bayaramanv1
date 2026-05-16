package admin

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	adminSvc "github.com/prast13/bayaraman/internal/service/admin"
	configSvc "github.com/prast13/bayaraman/internal/service/config"
)

type AdminHandler struct {
	adminService  adminSvc.AdminService
	configService configSvc.ConfigService
}

func NewAdminHandler(adminService adminSvc.AdminService, configService configSvc.ConfigService) *AdminHandler {
	return &AdminHandler{adminService: adminService, configService: configService}
}

func (h *AdminHandler) GetUsers(c echo.Context) error {
	users, err := h.adminService.GetUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"data": users})
}

func (h *AdminHandler) GetUserByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	user, err := h.adminService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": user})
}

func (h *AdminHandler) SuspendUser(c echo.Context) error {
	adminID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	err = h.adminService.SuspendUser(c.Request().Context(), id, adminID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User suspended successfully"})
}

func (h *AdminHandler) FreezeEscrow(c echo.Context) error {
	adminID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	escrowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	reason := c.FormValue("reason")
	if reason == "" {
		reason = "Fraud detected by admin"
	}

	err = h.adminService.FreezeEscrow(c.Request().Context(), escrowID, adminID, reason)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Escrow frozen"})
}

func (h *AdminHandler) OverrideDispute(c echo.Context) error {
	adminID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	escrowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	var req struct {
		WinnerRole string `json:"winner_role"`
		Reason     string `json:"reason"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	err = h.adminService.OverrideDispute(c.Request().Context(), escrowID, adminID, req.WinnerRole, req.Reason)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Dispute overridden"})
}

func (h *AdminHandler) GetEscrowTimeline(c echo.Context) error {
	escrowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	timeline, err := h.adminService.GetEscrowTimeline(c.Request().Context(), escrowID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": timeline})
}

func (h *AdminHandler) RetryPayout(c echo.Context) error {
	adminID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	payoutID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid payout ID"})
	}

	err = h.adminService.RetryPayout(c.Request().Context(), payoutID, adminID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Payout retry initiated"})
}

func (h *AdminHandler) GetConfigs(c echo.Context) error {
	configs := h.configService.GetAllConfigs(c.Request().Context())
	return c.JSON(http.StatusOK, map[string]interface{}{"data": configs})
}

func (h *AdminHandler) UpdateConfig(c echo.Context) error {
	var req struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid body"})
	}

	ctx := c.Request().Context()
	switch req.Key {
	case "platform_fee_percent":
		val, ok := req.Value.(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid value type for platform_fee_percent"})
		}
		if err := h.configService.SetPlatformFeePercent(ctx, val); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	case "withdraw_delay_hours":
		val, ok := req.Value.(float64)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid value type for withdraw_delay_hours"})
		}
		if err := h.configService.SetWithdrawDelayHours(ctx, int(val)); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unknown config key"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Config updated successfully"})
}
