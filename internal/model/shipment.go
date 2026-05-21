package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EscrowShipment stores shipping/tracking information linked to an escrow transaction.
type EscrowShipment struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	EscrowTransactionID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"escrow_transaction_id"`
	TrackingNumber      string    `gorm:"type:varchar(100);not null" json:"tracking_number"`
	CourierCode         string    `gorm:"type:varchar(50);not null" json:"courier_code"`
	CourierName         string    `gorm:"type:varchar(100)" json:"courier_name"`
	Status              string    `gorm:"type:varchar(50);default:'pending'" json:"status"` // pending, picked_up, in_transit, out_for_delivery, delivered, returned, failed
	LastUpdate          time.Time `json:"last_update"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	EscrowTransaction EscrowTransaction `gorm:"foreignKey:EscrowTransactionID" json:"-"`
}

func (s *EscrowShipment) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return
}

// ShippingRate represents a single courier rate option returned from the aggregator.
type ShippingRate struct {
	CourierCode  string  `json:"courier_code"`
	CourierName  string  `json:"courier_name"`
	ServiceType  string  `json:"service_type"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	ETDDays      int     `json:"etd_days"` // Estimated time of delivery in days
}

// ShippingRateRequest is the input for getting shipping rates.
type ShippingRateRequest struct {
	OriginPostalCode      string  `json:"origin_postal_code"`
	DestinationPostalCode string  `json:"destination_postal_code"`
	WeightGrams           int     `json:"weight_grams"`
}

// TrackingEvent represents a single event in the shipment tracking history.
type TrackingEvent struct {
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
}

// TrackingResult is the response for tracking a shipment.
type TrackingResult struct {
	TrackingNumber string          `json:"tracking_number"`
	CourierCode    string          `json:"courier_code"`
	Status         string          `json:"status"`
	Events         []TrackingEvent `json:"events"`
}
