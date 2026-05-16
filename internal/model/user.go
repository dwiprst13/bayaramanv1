package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	Email           string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone           *string   `gorm:"uniqueIndex" json:"phone"`
	PasswordHash    string    `gorm:"not null" json:"-"`
	Role            string    `gorm:"type:varchar(20);default:'buyer'" json:"role"` // buyer, seller, admin, moderator
	IsEmailVerified bool      `gorm:"default:false" json:"is_email_verified"`
	IsPhoneVerified bool      `gorm:"default:false" json:"is_phone_verified"`
	KYCStatus       string    `gorm:"type:varchar(20);default:'pending'" json:"kyc_status"` // pending, verified, rejected
	PrivyID         *string   `json:"privy_id"`
	Status          string    `gorm:"type:varchar(20);default:'active'" json:"status"` // active, suspended
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
