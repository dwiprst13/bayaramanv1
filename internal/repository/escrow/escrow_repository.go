package escrow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EscrowRepository interface {
	DB() *gorm.DB
	Create(ctx context.Context, escrow *model.EscrowTransaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.EscrowTransaction, error)
	FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.EscrowTransaction, error)
	Update(ctx context.Context, escrow *model.EscrowTransaction) error
	UpdateWithTx(ctx context.Context, tx *gorm.DB, escrow *model.EscrowTransaction) error
	// TransitionStatus atomically updates status only if current status matches expectedStatus.
	// Returns error if no row was affected (concurrent modification or invalid state).
	TransitionStatus(ctx context.Context, id uuid.UUID, expectedStatus string, newStatus string) error
	FindByBuyerID(ctx context.Context, buyerID uuid.UUID) ([]model.EscrowTransaction, error)
	FindBySellerID(ctx context.Context, sellerID uuid.UUID) ([]model.EscrowTransaction, error)
}

type escrowRepository struct {
	db *gorm.DB
}

func NewEscrowRepository(db *gorm.DB) EscrowRepository {
	return &escrowRepository{db: db}
}

func (r *escrowRepository) DB() *gorm.DB {
	return r.db
}

func (r *escrowRepository) Create(ctx context.Context, escrow *model.EscrowTransaction) error {
	return r.db.WithContext(ctx).Create(escrow).Error
}

func (r *escrowRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.EscrowTransaction, error) {
	var escrow model.EscrowTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Buyer").Preload("Seller").First(&escrow).Error
	return &escrow, err
}

// FindByIDForUpdate acquires a row-level lock (SELECT ... FOR UPDATE)
// to prevent concurrent state modifications on the same escrow.
func (r *escrowRepository) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.EscrowTransaction, error) {
	var escrow model.EscrowTransaction
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&escrow).Error
	return &escrow, err
}

func (r *escrowRepository) Update(ctx context.Context, escrow *model.EscrowTransaction) error {
	return r.db.WithContext(ctx).Save(escrow).Error
}

func (r *escrowRepository) UpdateWithTx(ctx context.Context, tx *gorm.DB, escrow *model.EscrowTransaction) error {
	return tx.WithContext(ctx).Save(escrow).Error
}

// TransitionStatus performs an atomic conditional update: UPDATE ... SET status = newStatus WHERE id = ? AND status = expectedStatus.
// If RowsAffected == 0, it means another process already changed the status (race condition detected).
func (r *escrowRepository) TransitionStatus(ctx context.Context, id uuid.UUID, expectedStatus string, newStatus string) error {
	result := r.db.WithContext(ctx).
		Model(&model.EscrowTransaction{}).
		Where("id = ? AND status = ?", id, expectedStatus).
		Update("status", newStatus)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("concurrent modification: escrow %s is no longer in '%s' state", id, expectedStatus)
	}
	return nil
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
