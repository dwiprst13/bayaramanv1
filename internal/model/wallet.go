package model

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance     float64   `gorm:"type:decimal(15,2);default:0" json:"balance"`
	HeldBalance float64   `gorm:"type:decimal(15,2);default:0" json:"held_balance"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type WalletTransaction struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	WalletID    uuid.UUID `gorm:"type:uuid;not null;index" json:"wallet_id"`
	Amount      float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Type        string    `gorm:"type:varchar(20);not null" json:"type"` // credit, debit
	ReferenceID string    `gorm:"type:varchar(100)" json:"reference_id"`   // EscrowID, PayoutID, dll.
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`

	Wallet Wallet `gorm:"foreignKey:WalletID" json:"-"`
}

type Payout struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Amount        float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	BankCode      string    `gorm:"type:varchar(50);not null" json:"bank_code"`
	AccountNumber string    `gorm:"type:varchar(50);not null" json:"account_number"`
	Status        string    `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, processing, completed, failed
	XenditDisbID  string    `gorm:"type:varchar(100)" json:"xendit_disbursement_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}
