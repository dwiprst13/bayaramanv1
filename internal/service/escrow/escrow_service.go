package escrow

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	auditLogRepo "github.com/prast13/bayaraman/internal/repository/auditlog"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"
	configSvc "github.com/prast13/bayaraman/internal/service/config"
	paymentSvc "github.com/prast13/bayaraman/internal/service/payment"
	storageSvc "github.com/prast13/bayaraman/internal/service/storage"
	walletSvc "github.com/prast13/bayaraman/internal/service/wallet"
	"gorm.io/gorm"
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
	UploadPackingVideo(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, file *multipart.FileHeader) (string, error)
	UploadUnboxingVideo(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, file *multipart.FileHeader) (string, error)
	UploadPackingPhoto(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, files []*multipart.FileHeader) ([]string, error)
	UploadUnboxingPhoto(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, files []*multipart.FileHeader) ([]string, error)
	UploadReceipt(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, trackingNumber string, courier string, file *multipart.FileHeader) (string, error)
	DeliverEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) error
}

type escrowService struct {
	escrowRepo   escrowRepo.EscrowRepository
	paymentSvc   paymentSvc.PaymentService
	auditLogRepo auditLogRepo.AuditLogRepository
	storageSvc   storageSvc.StorageService
	walletSvc    walletSvc.WalletService
	configSvc    configSvc.ConfigService
}

func NewEscrowService(escrowRepo escrowRepo.EscrowRepository, paymentSvc paymentSvc.PaymentService, auditLogRepo auditLogRepo.AuditLogRepository, storageSvc storageSvc.StorageService, walletSvc walletSvc.WalletService, configSvc configSvc.ConfigService) EscrowService {
	return &escrowService{
		escrowRepo:   escrowRepo,
		paymentSvc:   paymentSvc,
		auditLogRepo: auditLogRepo,
		storageSvc:   storageSvc,
		walletSvc:    walletSvc,
		configSvc:    configSvc,
	}
}

func (s *escrowService) checkAndExpire(ctx context.Context, escrow *model.EscrowTransaction) {
	if escrow.Status != "pending" {
		return
	}
	expiryHours := s.configSvc.GetEscrowExpiryHours(ctx)
	if time.Now().After(escrow.CreatedAt.Add(time.Duration(expiryHours) * time.Hour).Add(15 * time.Minute)) {
		// Use atomic transition to avoid race with webhook funding
		if err := s.escrowRepo.TransitionStatus(ctx, escrow.ID, "pending", "cancelled"); err != nil {
			// Another process already changed the status — not an error, just skip
			log.Printf("[LAZY_EXPIRE] Skipped expiry for escrow %s: %v", escrow.ID, err)
			return
		}
		escrow.Status = "cancelled"
		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: escrow.BuyerID,
			Action: "ESCROW_EXPIRED_SYSTEM",
		})
	}
}

func (s *escrowService) CreateEscrow(ctx context.Context, buyerID uuid.UUID, req CreateEscrowRequest) (*model.EscrowTransaction, error) {
	if buyerID == req.SellerID {
		return nil, errors.New("buyer and seller cannot be the same")
	}

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	feePercent := s.configSvc.GetPlatformFeePercent(ctx)
	fee := req.Amount * (feePercent / 100.0)

	escrow := &model.EscrowTransaction{
		BuyerID:     buyerID,
		SellerID:    req.SellerID,
		Title:       req.Title,
		Description: req.Description,
		Amount:      req.Amount,
		Fee:         fee,
		Status:      "pending",
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

	s.checkAndExpire(ctx, escrow)

	if escrow.BuyerID != buyerID {
		return nil, "", errors.New("unauthorized")
	}

	if err := model.ValidateTransition(escrow.Status, "funded"); err != nil {
		return nil, "", err
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
	for i := range all {
		s.checkAndExpire(ctx, &all[i])
	}
	return all, nil
}

func (s *escrowService) CompleteEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) error {
	db := s.escrowRepo.DB()

	var escrow *model.EscrowTransaction

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the escrow row to prevent concurrent completion
		var err error
		escrow, err = s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.BuyerID != buyerID {
			return errors.New("unauthorized")
		}

		if err := model.ValidateTransition(escrow.Status, "completed"); err != nil {
			return err
		}

		escrow.Status = "completed"
		return s.escrowRepo.UpdateWithTx(ctx, tx, escrow)
	})

	if err != nil {
		return err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: buyerID,
		Action: "ESCROW_COMPLETED",
	})

	// Credit wallet to seller (outside escrow transaction — wallet has its own locking)
	amountToCredit := escrow.Amount
	if err := s.walletSvc.CreditWallet(ctx, escrow.SellerID, amountToCredit, "Escrow payout for "+escrow.Title, escrow.ID.String()); err != nil {
		log.Printf("[CRITICAL] Failed to credit wallet for escrow %s to user %s: %v", escrow.ID, escrow.SellerID, err)
	}

	log.Printf("[WALLET LEDGER] Credited %.2f to Seller %s Wallet\n", amountToCredit, escrow.SellerID.String())

	return nil
}

func (s *escrowService) UploadPackingVideo(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, file *multipart.FileHeader) (string, error) {
	db := s.escrowRepo.DB()
	var url string

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.SellerID != sellerID {
			return errors.New("unauthorized")
		}

		if escrow.Status != "funded" && escrow.Status != "in_progress" {
			return errors.New("escrow must be funded or in_progress to upload packing video")
		}

		filename := fmt.Sprintf("packing_%s_%s%s", escrow.ID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
		uploadedURL, err := s.storageSvc.UploadFile(ctx, file, "videos", filename)
		if err != nil {
			return err
		}

		escrow.PackingVideoURL = uploadedURL
		url = uploadedURL

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: sellerID,
			Action: "UPLOAD_PACKING_VIDEO",
		})

		return nil
	})

	return url, err
}

func (s *escrowService) UploadUnboxingVideo(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, file *multipart.FileHeader) (string, error) {
	db := s.escrowRepo.DB()
	var url string

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.BuyerID != buyerID {
			return errors.New("unauthorized")
		}

		if escrow.Status != "in_progress" && escrow.Status != "completed" {
			return errors.New("escrow must be in_progress to upload unboxing video")
		}

		filename := fmt.Sprintf("unboxing_%s_%s%s", escrow.ID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
		uploadedURL, err := s.storageSvc.UploadFile(ctx, file, "videos", filename)
		if err != nil {
			return err
		}

		escrow.UnboxingVideoURL = uploadedURL
		url = uploadedURL

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: buyerID,
			Action: "UPLOAD_UNBOXING_VIDEO",
		})

		return nil
	})

	return url, err
}

func (s *escrowService) UploadPackingPhoto(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, files []*multipart.FileHeader) ([]string, error) {
	db := s.escrowRepo.DB()
	var urls []string

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.SellerID != sellerID {
			return errors.New("unauthorized")
		}

		if escrow.Status != "funded" && escrow.Status != "in_progress" {
			return errors.New("escrow must be funded or in_progress to upload packing photo")
		}

		if len(files) == 0 {
			return errors.New("no photos provided")
		}

		if len(escrow.PackingPhotoURLs)+len(files) > 3 {
			return errors.New("maximum 3 photos allowed")
		}

		for i, file := range files {
			filename := fmt.Sprintf("packing_photo_%s_%s_%d%s", escrow.ID.String(), time.Now().Format("20060102150405"), i, filepath.Ext(file.Filename))
			uploadedURL, err := s.storageSvc.UploadFile(ctx, file, "photos", filename)
			if err != nil {
				return err
			}
			urls = append(urls, uploadedURL)
		}

		escrow.PackingPhotoURLs = append(escrow.PackingPhotoURLs, urls...)

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: sellerID,
			Action: "UPLOAD_PACKING_PHOTO",
		})

		return nil
	})

	return urls, err
}

func (s *escrowService) UploadUnboxingPhoto(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, files []*multipart.FileHeader) ([]string, error) {
	db := s.escrowRepo.DB()
	var urls []string

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.BuyerID != buyerID {
			return errors.New("unauthorized")
		}

		if escrow.Status != "in_progress" && escrow.Status != "completed" {
			return errors.New("escrow must be in_progress to upload unboxing photo")
		}

		if len(files) == 0 {
			return errors.New("no photos provided")
		}

		if len(escrow.UnboxingPhotoURLs)+len(files) > 3 {
			return errors.New("maximum 3 photos allowed")
		}

		for i, file := range files {
			filename := fmt.Sprintf("unboxing_photo_%s_%s_%d%s", escrow.ID.String(), time.Now().Format("20060102150405"), i, filepath.Ext(file.Filename))
			uploadedURL, err := s.storageSvc.UploadFile(ctx, file, "photos", filename)
			if err != nil {
				return err
			}
			urls = append(urls, uploadedURL)
		}

		escrow.UnboxingPhotoURLs = append(escrow.UnboxingPhotoURLs, urls...)

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: buyerID,
			Action: "UPLOAD_UNBOXING_PHOTO",
		})

		return nil
	})

	return urls, err
}

func (s *escrowService) UploadReceipt(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, trackingNumber string, courier string, file *multipart.FileHeader) (string, error) {
	db := s.escrowRepo.DB()
	var url string

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.SellerID != sellerID {
			return errors.New("unauthorized")
		}

		if err := model.ValidateTransition(escrow.Status, "shipped"); err != nil {
			return err
		}

		filename := fmt.Sprintf("receipt_%s_%s%s", escrow.ID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
		uploadedURL, err := s.storageSvc.UploadFile(ctx, file, "photos", filename)
		if err != nil {
			return err
		}

		escrow.ReceiptPhotoURL = uploadedURL
		escrow.TrackingNumber = trackingNumber
		escrow.Courier = courier
		escrow.Status = "shipped"
		url = uploadedURL

		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: sellerID,
			Action: "UPLOAD_RECEIPT",
		})

		return nil
	})

	return url, err
}

func (s *escrowService) DeliverEscrow(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID) error {
	db := s.escrowRepo.DB()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		escrow, err := s.escrowRepo.FindByIDForUpdate(ctx, tx, escrowID)
		if err != nil {
			return errors.New("escrow not found")
		}

		if escrow.BuyerID != buyerID {
			return errors.New("unauthorized")
		}

		if err := model.ValidateTransition(escrow.Status, "delivered"); err != nil {
			return err
		}

		escrow.Status = "delivered"
		if err := s.escrowRepo.UpdateWithTx(ctx, tx, escrow); err != nil {
			return err
		}

		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: buyerID,
			Action: "ESCROW_DELIVERED",
		})

		return nil
	})
}
