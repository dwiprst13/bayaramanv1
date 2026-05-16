package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *model.Payment) error
	FindByXenditID(ctx context.Context, refID string) (*model.Payment, error)
	FindByEscrowID(ctx context.Context, escrowID uuid.UUID) ([]model.Payment, error)
	Update(ctx context.Context, payment *model.Payment) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *model.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *paymentRepository) FindByXenditID(ctx context.Context, refID string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.WithContext(ctx).Where("xendit_reference_id = ?", refID).First(&payment).Error
	return &payment, err
}

func (r *paymentRepository) FindByEscrowID(ctx context.Context, escrowID uuid.UUID) ([]model.Payment, error) {
	var payments []model.Payment
	err := r.db.WithContext(ctx).Where("escrow_transaction_id = ?", escrowID).Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) Update(ctx context.Context, payment *model.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}
