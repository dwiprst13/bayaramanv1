package auditlog

import (
	"context"

	"github.com/prast13/bayaraman/internal/model"
	"gorm.io/gorm"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	FindByActionLike(ctx context.Context, action string) ([]model.AuditLog, error)
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

func (r *auditLogRepository) FindByActionLike(ctx context.Context, action string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.db.WithContext(ctx).Where("action LIKE ?", "%"+action+"%").Order("created_at desc").Find(&logs).Error
	return logs, err
}
