package repository

import (
	"context"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"gorm.io/gorm"
)

// ComplianceCheckRepository owns all persistence operations for 合规核验.
type ComplianceCheckRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.ComplianceCheck], error)
	Get(context.Context, uint) (model.ComplianceCheck, error)
	Create(context.Context, *model.ComplianceCheck) error
	CreateAudited(context.Context, *model.ComplianceCheck, *model.AuditLog) error
	Update(context.Context, uint, uint, *model.ComplianceCheck) error
	UpdateAudited(context.Context, uint, uint, *model.ComplianceCheck, *model.AuditLog) error
	Delete(context.Context, uint) error
	DeleteAudited(context.Context, uint, *model.AuditLog) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type complianceCheckRepository struct {
	store *Store[model.ComplianceCheck]
}

func NewComplianceCheckRepository(db *gorm.DB) ComplianceCheckRepository {
	return &complianceCheckRepository{store: NewStore[model.ComplianceCheck](db)}
}

func (r *complianceCheckRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.ComplianceCheck], error) {
	return r.store.List(ctx, q)
}
func (r *complianceCheckRepository) Get(ctx context.Context, id uint) (model.ComplianceCheck, error) {
	return r.store.Get(ctx, id)
}
func (r *complianceCheckRepository) Create(ctx context.Context, item *model.ComplianceCheck) error {
	return r.store.Create(ctx, item)
}
func (r *complianceCheckRepository) CreateAudited(ctx context.Context, item *model.ComplianceCheck, audit *model.AuditLog) error {
	return r.store.CreateAudited(ctx, item, audit)
}
func (r *complianceCheckRepository) Update(ctx context.Context, id, version uint, item *model.ComplianceCheck) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *complianceCheckRepository) UpdateAudited(ctx context.Context, id, version uint, item *model.ComplianceCheck, audit *model.AuditLog) error {
	return r.store.UpdateAudited(ctx, id, version, item, audit)
}
func (r *complianceCheckRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *complianceCheckRepository) DeleteAudited(ctx context.Context, id uint, audit *model.AuditLog) error {
	return r.store.DeleteAudited(ctx, id, audit)
}
func (r *complianceCheckRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
