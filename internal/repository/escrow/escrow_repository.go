package escrow

import (
	"context"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type EscrowRepository interface {
	Create(ctx context.Context, escrow *model.EscrowTransaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.EscrowTransaction, error)
	Update(ctx context.Context, escrow *model.EscrowTransaction) error
	FindByBuyerID(ctx context.Context, buyerID uuid.UUID) ([]model.EscrowTransaction, error)
	FindBySellerID(ctx context.Context, sellerID uuid.UUID) ([]model.EscrowTransaction, error)
}

type escrowRepository struct {
	db *gorm.DB
}

func NewEscrowRepository(db *gorm.DB) EscrowRepository {
	return &escrowRepository{db: db}
}

func (r *escrowRepository) Create(ctx context.Context, escrow *model.EscrowTransaction) error {
	return r.db.WithContext(ctx).Create(escrow).Error
}

func (r *escrowRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.EscrowTransaction, error) {
	var escrow model.EscrowTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Buyer").Preload("Seller").First(&escrow).Error
	return &escrow, err
}

func (r *escrowRepository) Update(ctx context.Context, escrow *model.EscrowTransaction) error {
	return r.db.WithContext(ctx).Save(escrow).Error
}

func (r *escrowRepository) FindByBuyerID(ctx context.Context, buyerID uuid.UUID) ([]model.EscrowTransaction, error) {
	var escrows []model.EscrowTransaction
	err := r.db.WithContext(ctx).Where("buyer_id = ?", buyerID).Preload("Seller").Find(&escrows).Error
	return escrows, err
}

func (r *escrowRepository) FindBySellerID(ctx context.Context, sellerID uuid.UUID) ([]model.EscrowTransaction, error) {
	var escrows []model.EscrowTransaction
	err := r.db.WithContext(ctx).Where("seller_id = ?", sellerID).Preload("Buyer").Find(&escrows).Error
	return escrows, err
}
