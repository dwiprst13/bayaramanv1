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
	paymentSvc "github.com/prast13/bayaraman/internal/service/payment"
	storageSvc "github.com/prast13/bayaraman/internal/service/storage"
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
}

type escrowService struct {
	escrowRepo   escrowRepo.EscrowRepository
	paymentSvc   paymentSvc.PaymentService
	auditLogRepo auditLogRepo.AuditLogRepository
	storageSvc   storageSvc.StorageService
}

func NewEscrowService(escrowRepo escrowRepo.EscrowRepository, paymentSvc paymentSvc.PaymentService, auditLogRepo auditLogRepo.AuditLogRepository, storageSvc storageSvc.StorageService) EscrowService {
	return &escrowService{
		escrowRepo:   escrowRepo,
		paymentSvc:   paymentSvc,
		auditLogRepo: auditLogRepo,
		storageSvc:   storageSvc,
	}
}

func (s *escrowService) CreateEscrow(ctx context.Context, buyerID uuid.UUID, req CreateEscrowRequest) (*model.EscrowTransaction, error) {
	if buyerID == req.SellerID {
		return nil, errors.New("buyer and seller cannot be the same")
	}

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	fee := req.Amount * 0.05 

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

func (s *escrowService) UploadPackingVideo(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, file *multipart.FileHeader) (string, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return "", errors.New("escrow not found")
	}

	if escrow.SellerID != sellerID {
		return "", errors.New("unauthorized")
	}

	if escrow.Status != "funded" && escrow.Status != "in_progress" {
		return "", errors.New("escrow must be funded or in_progress to upload packing video")
	}

	filename := fmt.Sprintf("packing_%s_%s%s", escrow.ID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
	url, err := s.storageSvc.UploadFile(ctx, file, "videos", filename)
	if err != nil {
		return "", err
	}

	escrow.PackingVideoURL = url
	if escrow.Status == "funded" {
		escrow.Status = "in_progress"
	}
	
	if err := s.escrowRepo.Update(ctx, escrow); err != nil {
		return "", err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: sellerID,
		Action: "UPLOAD_PACKING_VIDEO",
	})

	return url, nil
}

func (s *escrowService) UploadUnboxingVideo(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, file *multipart.FileHeader) (string, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return "", errors.New("escrow not found")
	}

	if escrow.BuyerID != buyerID {
		return "", errors.New("unauthorized")
	}

	if escrow.Status != "in_progress" && escrow.Status != "completed" {
		return "", errors.New("escrow must be in_progress to upload unboxing video")
	}

	filename := fmt.Sprintf("unboxing_%s_%s%s", escrow.ID.String(), time.Now().Format("20060102150405"), filepath.Ext(file.Filename))
	url, err := s.storageSvc.UploadFile(ctx, file, "videos", filename)
	if err != nil {
		return "", err
	}

	escrow.UnboxingVideoURL = url
	
	if err := s.escrowRepo.Update(ctx, escrow); err != nil {
		return "", err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: buyerID,
		Action: "UPLOAD_UNBOXING_VIDEO",
	})

	return url, nil
}

func (s *escrowService) UploadPackingPhoto(ctx context.Context, escrowID uuid.UUID, sellerID uuid.UUID, files []*multipart.FileHeader) ([]string, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return nil, errors.New("escrow not found")
	}

	if escrow.SellerID != sellerID {
		return nil, errors.New("unauthorized")
	}

	if escrow.Status != "funded" && escrow.Status != "in_progress" {
		return nil, errors.New("escrow must be funded or in_progress to upload packing photo")
	}

	if len(files) == 0 {
		return nil, errors.New("no photos provided")
	}

	if len(escrow.PackingPhotoURLs)+len(files) > 3 {
		return nil, errors.New("maximum 3 photos allowed")
	}

	var urls []string
	for i, file := range files {
		filename := fmt.Sprintf("packing_photo_%s_%s_%d%s", escrow.ID.String(), time.Now().Format("20060102150405"), i, filepath.Ext(file.Filename))
		url, err := s.storageSvc.UploadFile(ctx, file, "photos", filename)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	escrow.PackingPhotoURLs = append(escrow.PackingPhotoURLs, urls...)
	if escrow.Status == "funded" {
		escrow.Status = "in_progress"
	}
	
	if err := s.escrowRepo.Update(ctx, escrow); err != nil {
		return nil, err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: sellerID,
		Action: "UPLOAD_PACKING_PHOTO",
	})

	return urls, nil
}

func (s *escrowService) UploadUnboxingPhoto(ctx context.Context, escrowID uuid.UUID, buyerID uuid.UUID, files []*multipart.FileHeader) ([]string, error) {
	escrow, err := s.escrowRepo.FindByID(ctx, escrowID)
	if err != nil {
		return nil, errors.New("escrow not found")
	}

	if escrow.BuyerID != buyerID {
		return nil, errors.New("unauthorized")
	}

	if escrow.Status != "in_progress" && escrow.Status != "completed" {
		return nil, errors.New("escrow must be in_progress to upload unboxing photo")
	}

	if len(files) == 0 {
		return nil, errors.New("no photos provided")
	}

	if len(escrow.UnboxingPhotoURLs)+len(files) > 3 {
		return nil, errors.New("maximum 3 photos allowed")
	}

	var urls []string
	for i, file := range files {
		filename := fmt.Sprintf("unboxing_photo_%s_%s_%d%s", escrow.ID.String(), time.Now().Format("20060102150405"), i, filepath.Ext(file.Filename))
		url, err := s.storageSvc.UploadFile(ctx, file, "photos", filename)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	escrow.UnboxingPhotoURLs = append(escrow.UnboxingPhotoURLs, urls...)
	
	if err := s.escrowRepo.Update(ctx, escrow); err != nil {
		return nil, err
	}

	s.auditLogRepo.Create(ctx, &model.AuditLog{
		UserID: buyerID,
		Action: "UPLOAD_UNBOXING_PHOTO",
	})

	return urls, nil
}
