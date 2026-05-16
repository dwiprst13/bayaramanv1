package chat

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	chatRepo "github.com/prast13/bayaraman/internal/repository/chat"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"
)

type ChatService interface {
	GetHistory(ctx context.Context, escrowID uuid.UUID, requesterID uuid.UUID, role string) ([]model.Message, error)
	SendMessage(ctx context.Context, escrowID uuid.UUID, senderID uuid.UUID, content string, imageURL string) (*model.Message, error)
	GetHub() *Hub
}

type chatService struct {
	chatRepo   chatRepo.ChatRepository
	escrowRepo escrowRepo.EscrowRepository
	hub        *Hub
}

func NewChatService(chatRepo chatRepo.ChatRepository, escrowRepo escrowRepo.EscrowRepository, hub *Hub) ChatService {
	return &chatService{
		chatRepo:   chatRepo,
		escrowRepo: escrowRepo,
		hub:        hub,
	}
}

func (s *chatService) GetHistory(ctx context.Context, escrowID uuid.UUID, requesterID uuid.UUID, role string) ([]model.Message, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return nil, errors.New("escrow not found")
	}

	// Authorization
	if role != "admin" {
		if escrow.BuyerID != requesterID && escrow.SellerID != requesterID {
			return nil, errors.New("unauthorized access to chat")
		}
	}

	return s.chatRepo.GetHistory(ctx, escrowID)
}

func (s *chatService) SendMessage(ctx context.Context, escrowID uuid.UUID, senderID uuid.UUID, content string, imageURL string) (*model.Message, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return nil, errors.New("escrow not found")
	}

	if escrow.BuyerID != senderID && escrow.SellerID != senderID {
		return nil, errors.New("unauthorized access to chat")
	}

	if content == "" && imageURL == "" {
		return nil, errors.New("message cannot be empty")
	}

	msg := &model.Message{
		EscrowID: escrowID,
		SenderID: senderID,
		Content:  content,
		ImageURL: imageURL,
	}

	if err := s.chatRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Broadcast
	var recipientID uuid.UUID
	if escrow.BuyerID == senderID {
		recipientID = escrow.SellerID
	} else {
		recipientID = escrow.BuyerID
	}

	s.hub.BroadcastToUser(recipientID, msg)
	s.hub.BroadcastToUser(senderID, msg)

	return msg, nil
}

func (s *chatService) GetHub() *Hub {
	return s.hub
}
