package escrow

import (
	"net/http"

	escrowSvc "github.com/prast13/bayaraman/internal/service/escrow"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type EscrowHandler struct {
	escrowService escrowSvc.EscrowService
}

func NewEscrowHandler(escrowService escrowSvc.EscrowService) *EscrowHandler {
	return &EscrowHandler{escrowService: escrowService}
}

func (h *EscrowHandler) Create(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req escrowSvc.CreateEscrowRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	escrow, err := h.escrowService.CreateEscrow(c.Request().Context(), userID, req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Escrow created successfully",
		"data":    escrow,
	})
}

func (h *EscrowHandler) Fund(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid escrow id"})
	}

	payment, url, err := h.escrowService.FundEscrow(c.Request().Context(), escrowID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Invoice created successfully",
		"checkout_url": url,
		"payment":      payment,
	})
}

func (h *EscrowHandler) Complete(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid escrow id"})
	}

	err = h.escrowService.CompleteEscrow(c.Request().Context(), escrowID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Escrow completed and funds released to seller",
	})
}

func (h *EscrowHandler) UploadPackingVideo(c echo.Context) error {
	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	sellerID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	file, err := c.FormFile("video")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Video file is required"})
	}

	url, err := h.escrowService.UploadPackingVideo(c.Request().Context(), escrowID, sellerID, file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Packing video uploaded successfully",
		"url":     url,
	})
}

func (h *EscrowHandler) UploadUnboxingVideo(c echo.Context) error {
	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	buyerID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	file, err := c.FormFile("video")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Video file is required"})
	}

	url, err := h.escrowService.UploadUnboxingVideo(c.Request().Context(), escrowID, buyerID, file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Unboxing video uploaded successfully",
		"url":     url,
	})
}

func (h *EscrowHandler) UploadPackingPhoto(c echo.Context) error {
	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	sellerID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}
	
	files := form.File["photos"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one photo is required"})
	}

	urls, err := h.escrowService.UploadPackingPhoto(c.Request().Context(), escrowID, sellerID, files)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Packing photos uploaded successfully",
		"urls":    urls,
	})
}

func (h *EscrowHandler) UploadUnboxingPhoto(c echo.Context) error {
	escrowIDStr := c.Param("id")
	escrowID, err := uuid.Parse(escrowIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	buyerID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}
	
	files := form.File["photos"]
	if len(files) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one photo is required"})
	}

	urls, err := h.escrowService.UploadUnboxingPhoto(c.Request().Context(), escrowID, buyerID, files)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Unboxing photos uploaded successfully",
		"urls":    urls,
	})
}

func (h *EscrowHandler) MyEscrows(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	role, _ := c.Get("role").(string)

	escrows, err := h.escrowService.GetMyEscrows(c.Request().Context(), userID, role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch escrows"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": escrows,
	})
}
