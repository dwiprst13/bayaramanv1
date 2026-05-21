package shipment

import (
	"context"

	"github.com/google/uuid"
	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type ShipmentRepository interface {
	Create(ctx context.Context, shipment *model.EscrowShipment) error
	FindByEscrowID(ctx context.Context, escrowID uuid.UUID) (*model.EscrowShipment, error)
	FindByTrackingNumber(ctx context.Context, trackingNumber string) (*model.EscrowShipment, error)
	Update(ctx context.Context, shipment *model.EscrowShipment) error
	FindActiveShipments(ctx context.Context) ([]model.EscrowShipment, error)
}

type shipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) ShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *model.EscrowShipment) error {
	return r.db.WithContext(ctx).Create(shipment).Error
}

func (r *shipmentRepository) FindByEscrowID(ctx context.Context, escrowID uuid.UUID) (*model.EscrowShipment, error) {
	var shipment model.EscrowShipment
	err := r.db.WithContext(ctx).Where("escrow_transaction_id = ?", escrowID).First(&shipment).Error
	return &shipment, err
}

func (r *shipmentRepository) FindByTrackingNumber(ctx context.Context, trackingNumber string) (*model.EscrowShipment, error) {
	var shipment model.EscrowShipment
	err := r.db.WithContext(ctx).Where("tracking_number = ?", trackingNumber).First(&shipment).Error
	return &shipment, err
}

func (r *shipmentRepository) Update(ctx context.Context, shipment *model.EscrowShipment) error {
	return r.db.WithContext(ctx).Save(shipment).Error
}

// FindActiveShipments returns all shipments that are not yet delivered or in a terminal state.
func (r *shipmentRepository) FindActiveShipments(ctx context.Context) ([]model.EscrowShipment, error) {
	var shipments []model.EscrowShipment
	err := r.db.WithContext(ctx).
		Where("status NOT IN ?", []string{"delivered", "returned", "failed"}).
		Find(&shipments).Error
	return shipments, err
}
