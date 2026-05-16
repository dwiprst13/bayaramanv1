package escrow

import (
	"context"
	"errors"
	"log"

	paymentSvc "github.com/prast13/bayaraman/internal/service/payment"

	auditLogRepo "github.com/prast13/bayaraman/internal/repository/auditlog"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
)

type CreateEscrowRequest struct {
	SellerID    uuid.UUID `json:"seller_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
}

type EscrowService interface {
	CreateEscrow(ctx context.Context, buyerID uuid.UUID, req CreateEscrowRequest) (*model.EscrowTransaction, error)
	FundEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) (*model.Payment, string, error)
	GetMyEscrows(ctx context.Context, userID uuid.UUID, role string) ([]model.EscrowTransaction, error)
	CompleteEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) error
}

type escrowService struct {
	escrowRepo   escrowRepo.EscrowRepository
	paymentSvc   paymentSvc.PaymentService
	auditLogRepo auditLogRepo.AuditLogRepository
}

func NewEscrowService(escrowRepo escrowRepo.EscrowRepository, paymentSvc paymentSvc.PaymentService, auditLogRepo auditLogRepo.AuditLogRepository) EscrowService {
	return &escrowService{
		escrowRepo:   escrowRepo,
		paymentSvc:   paymentSvc,
		auditLogRepo: auditLogRepo,
	}
}

func (s *escrowService) CreateEscrow(ctx context.Context, buyerID uuid.UUID, req CreateEscrowRequest) (*model.EscrowTransaction, error) {
	if buyerID == req.SellerID {
		return nil, errors.New("buyer and seller cannot be the same")
	}

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	fee := req.Amount * 0.05 // 5% fee for example

	escrow := &model.EscrowTransaction{
		BuyerID:     buyerID,
		SellerID:    req.SellerID,
		Title:       req.Title,
		Description: req.Description,
		Amount:      req.Amount,
		Fee:         fee,
		Status:      "created",
	}

	err := s.escrowRepo.Create(ctx, escrow)
	if err != nil {
		return nil, err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: buyerID,
		Action: "ESCROW_CREATED",
	})

	return escrow, nil
}

func (s *escrowService) FundEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) (*model.Payment, string, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return nil, "", errors.New("escrow not found")
	}

	if escrow.BuyerID != buyerID {
		return nil, "", errors.New("unauthorized")
	}

	if escrow.Status != "created" {
		return nil, "", errors.New("escrow cannot be funded at this status")
	}

	payment, url, err := s.paymentSvc.CreateInvoice(ctx, escrow)
	if err != nil {
		return nil, "", err
	}

	return payment, url, nil
}

func (s *escrowService) GetMyEscrows(ctx context.Context, userID uuid.UUID, role string) ([]model.EscrowTransaction, error) {
	buying, _ := s.escrowRepo.FindByBuyerID(ctx, userID)
	selling, _ := s.escrowRepo.FindBySellerID(ctx, userID)

	all := append(buying, selling...)
	return all, nil
}

func (s *escrowService) CompleteEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) error {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return errors.New("escrow not found")
	}

	if escrow.BuyerID != buyerID {
		return errors.New("unauthorized")
	}

	if escrow.Status != "funded" && escrow.Status != "in_progress" {
		return errors.New("escrow is not funded yet")
	}

	escrow.Status = "completed"
	err = s.escrowRepo.Update(ctx, escrow)
	if err != nil {
		return err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: buyerID,
		Action: "ESCROW_COMPLETED",
	})

	log.Printf("[STUB DISBURSEMENT] Payout %.2f to Seller %s\n", escrow.Amount, escrow.SellerID.String())

	return nil
}
