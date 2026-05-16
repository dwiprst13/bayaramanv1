package chat

import (
	"context"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type ChatRepository interface {
	Create(ctx context.Context, message *model.Message) error
	GetHistory(ctx context.Context, escrowID uuid.UUID) ([]model.Message, error)
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *chatRepository) GetHistory(ctx context.Context, escrowID uuid.UUID) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).Where("escrow_id = ?", escrowID).Order("created_at asc").Limit(100).Find(&messages).Error
	return messages, err
}
