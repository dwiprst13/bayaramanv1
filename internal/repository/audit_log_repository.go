package repository

import (
	"context"

	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}
 