package chat

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	chatSvc "github.com/prast13/bayaraman/internal/service/chat"
	storageSvc "github.com/prast13/bayaraman/internal/service/storage"
	"github.com/prast13/bayaraman/pkg/jwt"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatHandler struct {
	chatService chatSvc.ChatService
	storageSvc  storageSvc.StorageService
	jwtSecret   string
}

func NewChatHandler(chatService chatSvc.ChatService, storageSvc storageSvc.StorageService, jwtSecret string) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		storageSvc:  storageSvc,
		jwtSecret:   jwtSecret,
	}
}

func (h *ChatHandler) ConnectWS(c echo.Context) error {
	tokenString := c.QueryParam("token")
	if tokenString == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing token"})
	}

	claims, err := jwt.ParseToken(tokenString, h.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
	}

	userID := claims.UserID

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	hub := h.chatService.GetHub()
	hub.Register(userID, conn)

	defer hub.Unregister(userID, conn)

	for {
		var req struct {
			EscrowID uuid.UUID `json:"escrow_id"`
			Content  string    `json:"content"`
			ImageURL string    `json:"image_url"`
		}

		err := conn.ReadJSON(&req)
		if err != nil {
			break
		}

		_, err = h.chatService.SendMessage(c.Request().Context(), req.EscrowID, userID, req.Content, req.ImageURL)
		if err != nil {
			conn.WriteJSON(map[string]string{"error": err.Error()})
		}
	}

	return nil
}

func (h *ChatHandler) GetHistory(c echo.Context) error {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	role, _ := c.Get("role").(string)

	escrowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	messages, err := h.chatService.GetHistory(c.Request().Context(), escrowID, userID, role)
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": messages})
}

func (h *ChatHandler) UploadImage(c echo.Context) error {
	_, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	escrowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid escrow ID"})
	}

	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing image file"})
	}

	filename := fmt.Sprintf("chat_%s_%s%s", escrowID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
	url, err := h.storageSvc.UploadFile(c.Request().Context(), file, "photos", filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to upload image"})
	}

	return c.JSON(http.StatusOK, map[string]string{"image_url": url})
}
