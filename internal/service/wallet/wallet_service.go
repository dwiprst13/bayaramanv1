package wallet

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"github.com/prast13/bayaraman/internal/repository/wallet"
	"gorm.io/gorm"
)

type WalletService interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, []model.WalletTransaction, error)
	CreditWallet(ctx context.Context, userID uuid.UUID, amount float64, description string, referenceID string) error
	Withdraw(ctx context.Context, userID uuid.UUID, amount float64, bankCode string, accountNumber string) (*model.Payout, error)
}

type walletService struct {
	walletRepo wallet.WalletRepository
}

func NewWalletService(walletRepo wallet.WalletRepository) WalletService {
	return &walletService{walletRepo: walletRepo}
}

func (s *walletService) GetWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, []model.WalletTransaction, error) {
	w, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		w = &model.Wallet{UserID: userID, Balance: 0, HeldBalance: 0}
		err = s.walletRepo.Create(ctx, w)
		if err != nil {
			return nil, nil, err
		}
	}

	txs, err := s.walletRepo.FindTransactionsByWalletID(ctx, w.ID)
	if err != nil {
		return w, nil, nil
	}

	return w, txs, nil
}

func (s *walletService) CreditWallet(ctx context.Context, userID uuid.UUID, amount float64, description string, referenceID string) error {
	if amount <= 0 {
		return errors.New("credit amount must be greater than zero")
	}

	db := s.walletRepo.DB()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Acquire row lock to prevent concurrent balance modifications
		w, err := s.walletRepo.FindByUserIDForUpdate(ctx, tx, userID)
		if err != nil {
			// Wallet doesn't exist yet, create it within this transaction
			w = &model.Wallet{UserID: userID, Balance: 0, HeldBalance: 0}
			if err := s.walletRepo.CreateWithTx(ctx, tx, w); err != nil {
				return err
			}
		}

		w.Balance += amount
		if err := s.walletRepo.UpdateWithTx(ctx, tx, w); err != nil {
			return err
		}

		walletTx := &model.WalletTransaction{
			WalletID:    w.ID,
			Amount:      amount,
			Type:        "credit",
			ReferenceID: referenceID,
			Description: description,
		}

		return s.walletRepo.CreateTransactionWithTx(ctx, tx, walletTx)
	})
}

func (s *walletService) Withdraw(ctx context.Context, userID uuid.UUID, amount float64, bankCode string, accountNumber string) (*model.Payout, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	db := s.walletRepo.DB()
	var payout *model.Payout

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Acquire row lock to prevent concurrent balance modifications
		w, err := s.walletRepo.FindByUserIDForUpdate(ctx, tx, userID)
		if err != nil {
			return errors.New("wallet not found")
		}

		if w.Balance < amount {
			return errors.New("insufficient balance")
		}

		// Deduct balance atomically within the transaction
		w.Balance -= amount
		if err := s.walletRepo.UpdateWithTx(ctx, tx, w); err != nil {
			return err
		}

		// Record the wallet transaction
		walletTx := &model.WalletTransaction{
			WalletID:    w.ID,
			Amount:      amount,
			Type:        "debit",
			ReferenceID: "WITHDRAWAL",
			Description: "Withdraw to bank account",
		}
		if err := s.walletRepo.CreateTransactionWithTx(ctx, tx, walletTx); err != nil {
			return err
		}

		// Create payout record
		payout = &model.Payout{
			UserID:        userID,
			Amount:        amount,
			BankCode:      bankCode,
			AccountNumber: accountNumber,
			Status:        "processing",
		}
		if err := s.walletRepo.CreatePayoutWithTx(ctx, tx, payout); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Call external disbursement API OUTSIDE the DB transaction
	// to avoid holding the lock during network calls.
	log.Printf("[STUB DISBURSEMENT] Calling Xendit Disbursement API for %v to %s %s\n", amount, bankCode, accountNumber)
	payout.Status = "completed"
	payout.XenditDisbID = "disb_" + uuid.New().String()[:8]
	s.walletRepo.UpdatePayout(ctx, payout)

	return payout, nil
}
