package payment

import (
	"context"
	"errors"
	"fmt"
	"log"

	auditLogRepo "github.com/prast13/bayaraman/internal/repository/auditlog"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"
	paymentRepo "github.com/prast13/bayaraman/internal/repository/payment"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
)

type PaymentService interface {
	CreateInvoice(ctx context.Context, escrow *model.EscrowTransaction) (*model.Payment, string, error)
	ProcessWebhook(ctx context.Context, payload map[string]interface{}) error
}

type paymentService struct {
	paymentRepo  paymentRepo.PaymentRepository
	escrowRepo   escrowRepo.EscrowRepository
	auditLogRepo auditLogRepo.AuditLogRepository
	xenditKey    string
}

func NewPaymentService(paymentRepo paymentRepo.PaymentRepository, escrowRepo escrowRepo.EscrowRepository, auditLogRepo auditLogRepo.AuditLogRepository, xenditKey string) PaymentService {
	return &paymentService{
		paymentRepo:  paymentRepo,
		escrowRepo:   escrowRepo,
		auditLogRepo: auditLogRepo,
		xenditKey:    xenditKey,
	}
}

func (s *paymentService) CreateInvoice(ctx context.Context, escrow *model.EscrowTransaction) (*model.Payment, string, error) {
	totalAmount := escrow.Amount + escrow.Fee
	invoiceID := "inv_" + uuid.New().String()[:8]
	checkoutURL := ""

	if s.xenditKey == "" {
		log.Printf("[STUB XENDIT] Creating Invoice for Escrow %s. Amount: %.2f\n", escrow.ID.String(), totalAmount)
		checkoutURL = fmt.Sprintf("https://mock.xendit.co/checkout/%s", invoiceID)
	} else {
		checkoutURL = fmt.Sprintf("https://checkout.xendit.co/web/%s", invoiceID)
	}

	payment := &model.Payment{
		EscrowTransactionID: escrow.ID,
		XenditReferenceID:   invoiceID,
		Amount:              totalAmount,
		Status:              "pending",
		Type:                "pay_in",
	}

	err := s.paymentRepo.Create(ctx, payment)
	if err != nil {
		return nil, "", err
	}

	return payment, checkoutURL, nil
}

func (s *paymentService) ProcessWebhook(ctx context.Context, payload map[string]interface{}) error {
	invoiceID, ok := payload["id"].(string)
	if !ok {
		return errors.New("invalid payload: missing invoice id")
	}

	status, ok := payload["status"].(string)
	if !ok {
		return errors.New("invalid payload: missing status")
	}

	payment, err := s.paymentRepo.FindByXenditID(ctx, invoiceID)
	if err != nil {
		return errors.New("payment not found")
	}

	// Idempotent: already processed
	if payment.Status == "paid" {
		return nil
	}

	if status == "PAID" || status == "SETTLED" {
		payment.Status = "paid"

		// Use atomic conditional update to prevent race with expired escrow worker.
		// If the escrow was already cancelled by the worker, TransitionStatus returns an error
		// and we simply skip the funding — no data corruption.
		if err := s.escrowRepo.TransitionStatus(ctx, payment.EscrowTransactionID, "pending", "funded"); err != nil {
			log.Printf("[WEBHOOK] Could not transition escrow %s to funded (likely already cancelled): %v\n", payment.EscrowTransactionID, err)
		} else {
			// Fetch escrow only for audit log (non-critical)
			escrow, fetchErr := s.escrowRepo.FindByID(ctx, payment.EscrowTransactionID)
			if fetchErr == nil {
				s.auditLogRepo.Create(ctx, &model.AuditLog{
					UserID: escrow.BuyerID,
					Action: "ESCROW_FUNDED",
				})
			}
		}
	} else if status == "EXPIRED" {
		payment.Status = "expired"
	}

	return s.paymentRepo.Update(ctx, payment)
}
