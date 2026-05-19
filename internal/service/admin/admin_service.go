package admin

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"github.com/prast13/bayaraman/internal/repository/auditlog"
	"github.com/prast13/bayaraman/internal/repository/escrow"
	"github.com/prast13/bayaraman/internal/repository/user"
	walletSvc "github.com/prast13/bayaraman/internal/service/wallet"
	"gorm.io/gorm"
)

type AdminService interface {
	GetUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	SuspendUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error

	FreezeEscrow(ctx context.Context, escrowID uuid.UUID, adminID uuid.UUID, reason string) error
	OverrideDispute(ctx context.Context, escrowID uuid.UUID, adminID uuid.UUID, winnerRole string, reason string) error
	GetEscrowTimeline(ctx context.Context, escrowID uuid.UUID) ([]model.AuditLog, error)

	RetryPayout(ctx context.Context, payoutID uuid.UUID, adminID uuid.UUID) error
}

type adminService struct {
	userRepo     user.UserRepository
	escrowRepo   escrow.EscrowRepository
	auditLogRepo auditlog.AuditLogRepository
	walletSvc    walletSvc.WalletService
}

func NewAdminService(userRepo user.UserRepository, escrowRepo escrow.EscrowRepository, auditLogRepo auditlog.AuditLogRepository, walletSvc walletSvc.WalletService) AdminService {
	return &adminService{
		userRepo:     userRepo,
		escrowRepo:   escrowRepo,
		auditLogRepo: auditLogRepo,
		walletSvc:    walletSvc,
	}
}

func (s *adminService) GetUsers(ctx context.Context) ([]model.User, error) {
	return s.userRepo.FindAll(ctx)
}

func (s *adminService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *adminService) SuspendUser(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	u.Status = "suspended"
	if err := s.userRepo.Update(ctx, u); err != nil {
		return err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: adminID,
		Action: "SUSPEND_USER_" + id.String(),
	})
	return nil
}

func (s *adminService) FreezeEscrow(ctx context.Context, escrowID uuid.UUID, adminID uuid.UUID, reason string) error {
	db := s.escrowRepo.DB()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the escrow row to prevent concurrent state changes
		e, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return err
		}

		if err := model.ValidateTransition(e.Status, "frozen"); err != nil {
			return err
		}

		e.Status = "frozen"
		if err := s.escrowRepo.UpdateWithTx(ctx, tx, e); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: adminID,
			Action: "FREEZE_ESCROW_" + escrowID.String() + "_REASON:" + reason,
		})
		return nil
	})
}

func (s *adminService) OverrideDispute(ctx context.Context, escrowID uuid.UUID, adminID uuid.UUID, winnerRole string, reason string) error {
	db := s.escrowRepo.DB()

	var sellerID uuid.UUID
	var amount float64
	var title string
	var escrowIDStr string
	var shouldCredit bool

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the escrow row to prevent concurrent dispute resolution
		e, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return err
		}

		if e.Status != "disputed" && e.Status != "frozen" {
			return errors.New("escrow must be disputed or frozen to override")
		}

		if winnerRole == "buyer" {
			e.Status = "cancelled"
		} else if winnerRole == "seller" {
			e.Status = "completed"
			sellerID = e.SellerID
			amount = e.Amount
			title = e.Title
			escrowIDStr = e.ID.String()
			shouldCredit = true
		} else {
			return errors.New("invalid winner role")
		}

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, e); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: adminID,
			Action: "OVERRIDE_DISPUTE_" + escrowID.String() + "_WINNER:" + winnerRole + "_REASON:" + reason,
		})

		return nil
	})

	if err != nil {
		return err
	}

	// Credit wallet outside the escrow transaction (wallet has its own locking)
	if shouldCredit {
		_ = s.walletSvc.CreditWallet(ctx, sellerID, amount, "Dispute Won: "+title, escrowIDStr)
	}

	return nil
}

func (s *adminService) GetEscrowTimeline(ctx context.Context, escrowID uuid.UUID) ([]model.AuditLog, error) {
	return s.auditLogRepo.FindByActionLike(ctx, escrowID.String())
}

func (s *adminService) RetryPayout(ctx context.Context, payoutID uuid.UUID, adminID uuid.UUID) error {
	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: adminID,
		Action: "RETRY_PAYOUT_" + payoutID.String(),
	})
	return nil
}
