package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	EscrowTransactionID uuid.UUID `gorm:"type:uuid;not null" json:"escrow_transaction_id"`
	XenditReferenceID   string    `gorm:"uniqueIndex;not null" json:"xendit_reference_id"` // Invoice ID or Disbursement ID
	Amount              float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Status              string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, paid, expired, failed
	Type                string    `gorm:"type:varchar(20);not null" json:"type"`            // pay_in, payout
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	EscrowTransaction EscrowTransaction `gorm:"foreignKey:EscrowTransactionID" json:"escrow_transaction"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = uuid.New()
	return
}
