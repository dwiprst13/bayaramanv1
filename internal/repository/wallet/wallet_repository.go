package wallet

import (
	"context"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type WalletRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error)
	Create(ctx context.Context, wallet *model.Wallet) error
	Update(ctx context.Context, wallet *model.Wallet) error
	CreateTransaction(ctx context.Context, transaction *model.WalletTransaction) error
	FindTransactionsByWalletID(ctx context.Context, walletID uuid.UUID) ([]model.WalletTransaction, error)
	CreatePayout(ctx context.Context, payout *model.Payout) error
	UpdatePayout(ctx context.Context, payout *model.Payout) error
}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	var wallet model.Wallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	return &wallet, err
}

func (r *walletRepository) Create(ctx context.Context, wallet *model.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *walletRepository) Update(ctx context.Context, wallet *model.Wallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}

func (r *walletRepository) CreateTransaction(ctx context.Context, transaction *model.WalletTransaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

func (r *walletRepository) FindTransactionsByWalletID(ctx context.Context, walletID uuid.UUID) ([]model.WalletTransaction, error) {
	var transactions []model.WalletTransaction
	err := r.db.WithContext(ctx).Where("wallet_id = ?", walletID).Order("created_at desc").Find(&transactions).Error
	return transactions, err
}

func (r *walletRepository) CreatePayout(ctx context.Context, payout *model.Payout) error {
	return r.db.WithContext(ctx).Create(payout).Error
}

func (r *walletRepository) UpdatePayout(ctx context.Context, payout *model.Payout) error {
	return r.db.WithContext(ctx).Save(payout).Error
}
