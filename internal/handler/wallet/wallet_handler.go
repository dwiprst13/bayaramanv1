package wallet

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	walletSvc "github.com/prast13/bayaraman/internal/service/wallet"
)

type WalletHandler struct {
	walletService walletSvc.WalletService
}

func NewWalletHandler(walletService walletSvc.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) GetMyWallet(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	wallet, txs, err := h.walletService.GetWallet(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"wallet":       wallet,
		"transactions": txs,
	})
}

type WithdrawRequest struct {
	Amount        float64 `json:"amount"`
	BankCode      string  `json:"bank_code"`
	AccountNumber string  `json:"account_number"`
}

func (h *WalletHandler) Withdraw(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req WithdrawRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Amount <= 0 || req.BankCode == "" || req.AccountNumber == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "amount, bank_code, and account_number are required"})
	}

	payout, err := h.walletService.Withdraw(c.Request().Context(), userID, req.Amount, req.BankCode, req.AccountNumber)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Withdrawal request processed",
		"payout":  payout,
	})
}
