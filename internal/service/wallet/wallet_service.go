package wallet

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"github.com/prast13/bayaraman/internal/repository/wallet"
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
	w, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		w = &model.Wallet{UserID: userID, Balance: 0, HeldBalance: 0}
		err = s.walletRepo.Create(ctx, w)
		if err != nil {
			return err
		}
	}

	w.Balance += amount
	err = s.walletRepo.Update(ctx, w)
	if err != nil {
		return err
	}

	tx := &model.WalletTransaction{
		WalletID:    w.ID,
		Amount:      amount,
		Type:        "credit",
		ReferenceID: referenceID,
		Description: description,
	}

	return s.walletRepo.CreateTransaction(ctx, tx)
}

func (s *walletService) Withdraw(ctx context.Context, userID uuid.UUID, amount float64, bankCode string, accountNumber string) (*model.Payout, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	w, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("wallet not found")
	}

	if w.Balance < amount {
		return nil, errors.New("insufficient balance")
	}

	w.Balance -= amount
	err = s.walletRepo.Update(ctx, w)
	if err != nil {
		return nil, err
	}

	tx := &model.WalletTransaction{
		WalletID:    w.ID,
		Amount:      amount,
		Type:        "debit",
		ReferenceID: "WITHDRAWAL",
		Description: "Withdraw to bank account",
	}

	if err := s.walletRepo.CreateTransaction(ctx, tx); err != nil {
		w.Balance += amount
		s.walletRepo.Update(ctx, w)
		return nil, err
	}

	payout := &model.Payout{
		UserID:        userID,
		Amount:        amount,
		BankCode:      bankCode,
		AccountNumber: accountNumber,
		Status:        "processing",
	}

	if err := s.walletRepo.CreatePayout(ctx, payout); err != nil {
		return nil, err
	}

	log.Printf("[STUB DISBURSEMENT] Calling Xendit Disbursement API for %v to %s %s\n", amount, bankCode, accountNumber)
	payout.Status = "completed"
	payout.XenditDisbID = "disb_" + uuid.New().String()[:8]
	s.walletRepo.UpdatePayout(ctx, payout)

	return payout, nil
}
