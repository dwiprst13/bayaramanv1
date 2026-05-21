package shipping

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	auditLogRepo "github.com/prast13/bayaraman/internal/repository/auditlog"
	escrowRepo "github.com/prast13/bayaraman/internal/repository/escrow"
	shipmentRepo "github.com/prast13/bayaraman/internal/repository/shipment"
)

type ShippingService interface {
	// GetRates returns available shipping rates from the aggregator.
	GetRates(ctx context.Context, req model.ShippingRateRequest) ([]model.ShippingRate, error)

	// TrackShipment returns the current tracking status and history for an escrow's shipment.
	TrackShipment(ctx context.Context, escrowID uuid.UUID) (*model.TrackingResult, error)

	// RegisterTracking creates a shipment record and registers it with the aggregator for webhook updates.
	RegisterTracking(ctx context.Context, escrowID uuid.UUID, trackingNumber string, courierCode string) error

	// ProcessTrackingWebhook handles incoming tracking status updates from the aggregator.
	ProcessTrackingWebhook(ctx context.Context, trackingNumber string, courierCode string, status string) error

	// SyncActiveShipments polls the aggregator for status updates on all active shipments.
	SyncActiveShipments(ctx context.Context) error
}

type shippingService struct {
	shipmentRepo shipmentRepo.ShipmentRepository
	escrowRepo   escrowRepo.EscrowRepository
	auditLogRepo auditLogRepo.AuditLogRepository
	biteshipKey  string
}

func NewShippingService(
	shipmentRepo shipmentRepo.ShipmentRepository,
	escrowRepo escrowRepo.EscrowRepository,
	auditLogRepo auditLogRepo.AuditLogRepository,
	biteshipKey string,
) ShippingService {
	return &shippingService{
		shipmentRepo: shipmentRepo,
		escrowRepo:   escrowRepo,
		auditLogRepo: auditLogRepo,
		biteshipKey:  biteshipKey,
	}
}

func (s *shippingService) GetRates(ctx context.Context, req model.ShippingRateRequest) ([]model.ShippingRate, error) {
	if req.OriginPostalCode == "" || req.DestinationPostalCode == "" {
		return nil, errors.New("origin and destination postal codes are required")
	}
	if req.WeightGrams <= 0 {
		return nil, errors.New("weight must be greater than zero")
	}

	// Stub implementation — in production, call Biteship API:
	// POST https://api.biteship.com/v1/rates
	if s.biteshipKey == "" {
		log.Printf("[STUB BITESHIP] GetRates from %s to %s, weight: %dg\n", req.OriginPostalCode, req.DestinationPostalCode, req.WeightGrams)
		return []model.ShippingRate{
			{CourierCode: "jne", CourierName: "JNE", ServiceType: "REG", Description: "JNE Reguler", Price: 15000, ETDDays: 3},
			{CourierCode: "jne", CourierName: "JNE", ServiceType: "YES", Description: "JNE YES (1 hari)", Price: 25000, ETDDays: 1},
			{CourierCode: "sicepat", CourierName: "SiCepat", ServiceType: "REG", Description: "SiCepat Reguler", Price: 12000, ETDDays: 3},
			{CourierCode: "anteraja", CourierName: "AnterAja", ServiceType: "REG", Description: "AnterAja Reguler", Price: 13000, ETDDays: 4},
		}, nil
	}

	// TODO: Real Biteship API call
	// resp, err := http.Post("https://api.biteship.com/v1/rates", ...)
	return nil, errors.New("biteship integration not yet implemented")
}

func (s *shippingService) TrackShipment(ctx context.Context, escrowID uuid.UUID) (*model.TrackingResult, error) {
	shipment, err := s.shipmentRepo.FindByEscrowID(ctx, escrowID)
	if err != nil {
		return nil, errors.New("shipment not found for this escrow")
	}

	// Stub implementation — in production, call Biteship API:
	// GET https://api.biteship.com/v1/trackings/{tracking_id}
	if s.biteshipKey == "" {
		log.Printf("[STUB BITESHIP] TrackShipment %s via %s\n", shipment.TrackingNumber, shipment.CourierCode)
		return &model.TrackingResult{
			TrackingNumber: shipment.TrackingNumber,
			CourierCode:    shipment.CourierCode,
			Status:         shipment.Status,
			Events: []model.TrackingEvent{
				{Status: "picked_up", Description: "Paket telah diambil kurir", Timestamp: shipment.CreatedAt, Location: "Gudang Asal"},
				{Status: shipment.Status, Description: fmt.Sprintf("Status saat ini: %s", shipment.Status), Timestamp: shipment.LastUpdate, Location: "-"},
			},
		}, nil
	}

	// TODO: Real Biteship API call
	return nil, errors.New("biteship integration not yet implemented")
}

func (s *shippingService) RegisterTracking(ctx context.Context, escrowID uuid.UUID, trackingNumber string, courierCode string) error {
	// Check if shipment already exists
	existing, _ := s.shipmentRepo.FindByEscrowID(ctx, escrowID)
	if existing != nil {
		// Update existing record
		existing.TrackingNumber = trackingNumber
		existing.CourierCode = courierCode
		existing.Status = "pending"
		existing.LastUpdate = time.Now()
		return s.shipmentRepo.Update(ctx, existing)
	}

	shipment := &model.EscrowShipment{
		EscrowTransactionID: escrowID,
		TrackingNumber:      trackingNumber,
		CourierCode:         courierCode,
		Status:              "pending",
		LastUpdate:          time.Now(),
	}

	if err := s.shipmentRepo.Create(ctx, shipment); err != nil {
		return err
	}

	// In production, register the tracking number with Biteship for webhook notifications:
	// POST https://api.biteship.com/v1/trackings
	if s.biteshipKey == "" {
		log.Printf("[STUB BITESHIP] Registered tracking %s (%s) for webhook notifications\n", trackingNumber, courierCode)
	}

	return nil
}

func (s *shippingService) ProcessTrackingWebhook(ctx context.Context, trackingNumber string, courierCode string, status string) error {
	shipment, err := s.shipmentRepo.FindByTrackingNumber(ctx, trackingNumber)
	if err != nil {
		return errors.New("shipment not found")
	}

	shipment.Status = status
	shipment.LastUpdate = time.Now()

	if err := s.shipmentRepo.Update(ctx, shipment); err != nil {
		return err
	}

	log.Printf("[SHIPPING] Tracking %s updated to: %s\n", trackingNumber, status)

	// If delivered, auto-transition escrow from shipped -> delivered
	if status == "delivered" {
		if err := s.autoDeliverEscrow(ctx, shipment.EscrowTransactionID); err != nil {
			log.Printf("[SHIPPING] Failed to auto-deliver escrow %s: %v\n", shipment.EscrowTransactionID, err)
		}
	}

	return nil
}

// autoDeliverEscrow transitions the escrow to "delivered" when the courier confirms delivery.
func (s *shippingService) autoDeliverEscrow(ctx context.Context, escrowID uuid.UUID) error {
	err := s.escrowRepo.TransitionStatus(ctx, escrowID, "shipped", "delivered")
	if err != nil {
		return err
	}

	escrow, _ := s.escrowRepo.FindByID(ctx, escrowID)
	if escrow != nil {
		s.auditLogRepo.Create(ctx, &model.AuditLog{
			UserID: escrow.BuyerID,
			Action: "ESCROW_DELIVERED_AUTO_TRACKING",
		})
	}

	log.Printf("[SHIPPING] Escrow %s auto-delivered via tracking webhook\n", escrowID)
	return nil
}

func (s *shippingService) SyncActiveShipments(ctx context.Context) error {
	shipments, err := s.shipmentRepo.FindActiveShipments(ctx)
	if err != nil {
		return err
	}

	if len(shipments) == 0 {
		return nil
	}

	log.Printf("[SHIPPING_SYNC] Syncing %d active shipments...\n", len(shipments))

	for _, shipment := range shipments {
		// In production, poll Biteship API for each tracking number:
		// GET https://api.biteship.com/v1/trackings/{id}
		if s.biteshipKey == "" {
			log.Printf("[STUB BITESHIP] Polling status for %s (%s) — current: %s\n", shipment.TrackingNumber, shipment.CourierCode, shipment.Status)
			continue
		}

		// TODO: Real API call, then update status if changed
		// newStatus := fetchFromBiteship(shipment.TrackingNumber)
		// if newStatus != shipment.Status {
		//     s.ProcessTrackingWebhook(ctx, shipment.TrackingNumber, shipment.CourierCode, newStatus)
		// }
	}

	return nil
}
