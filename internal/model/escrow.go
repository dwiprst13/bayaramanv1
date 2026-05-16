package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type EscrowTransaction struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	BuyerID     uuid.UUID `gorm:"type:uuid;not null" json:"buyer_id"`
	SellerID    uuid.UUID `gorm:"type:uuid;not null" json:"seller_id"`
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Amount      float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Fee         float64   `gorm:"type:decimal(15,2);not null" json:"fee"`
	Status            string         `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, funded, shipped, delivered, disputed, completed, cancelled, frozen
	TrackingNumber    string         `gorm:"type:varchar(100)" json:"tracking_number"`
	Courier           string         `gorm:"type:varchar(50)" json:"courier"`
	ReceiptPhotoURL   string         `gorm:"type:varchar(255)" json:"receipt_photo_url"`
	PackingVideoURL   string         `gorm:"type:varchar(255)" json:"packing_video_url"`
	UnboxingVideoURL  string         `gorm:"type:varchar(255)" json:"unboxing_video_url"`
	PackingPhotoURLs  pq.StringArray `gorm:"type:text[]" json:"packing_photo_urls"`
	UnboxingPhotoURLs pq.StringArray `gorm:"type:text[]" json:"unboxing_photo_urls"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	Buyer  User `gorm:"foreignKey:BuyerID" json:"buyer"`
	Seller User `gorm:"foreignKey:SellerID" json:"seller"`
}

func (e *EscrowTransaction) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID = uuid.New()
	return
}
