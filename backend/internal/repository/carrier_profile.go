package repository

import (
	"context"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"gorm.io/gorm"
)

// CarrierProfileRepository owns all persistence operations for 承运资质.
type CarrierProfileRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.CarrierProfile], error)
	Get(context.Context, uint) (model.CarrierProfile, error)
	FindByCode(context.Context, string) (model.CarrierProfile, error)
	Create(context.Context, *model.CarrierProfile) error
	CreateAudited(context.Context, *model.CarrierProfile, *model.AuditLog) error
	Update(context.Context, uint, uint, *model.CarrierProfile) error
	UpdateAudited(context.Context, uint, uint, *model.CarrierProfile, *model.AuditLog) error
	Delete(context.Context, uint) error
	DeleteAudited(context.Context, uint, *model.AuditLog) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type carrierProfileRepository struct {
	store *Store[model.CarrierProfile]
}

func NewCarrierProfileRepository(db *gorm.DB) CarrierProfileRepository {
	return &carrierProfileRepository{store: NewStore[model.CarrierProfile](db)}
}

func (r *carrierProfileRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.CarrierProfile], error) {
	return r.store.List(ctx, q)
}
func (r *carrierProfileRepository) Get(ctx context.Context, id uint) (model.CarrierProfile, error) {
	return r.store.Get(ctx, id)
}
func (r *carrierProfileRepository) FindByCode(ctx context.Context, code string) (model.CarrierProfile, error) {
	return r.store.FindByCode(ctx, code)
}
func (r *carrierProfileRepository) Create(ctx context.Context, item *model.CarrierProfile) error {
	return r.store.Create(ctx, item)
}
func (r *carrierProfileRepository) CreateAudited(ctx context.Context, item *model.CarrierProfile, audit *model.AuditLog) error {
	return r.store.CreateAudited(ctx, item, audit)
}
func (r *carrierProfileRepository) Update(ctx context.Context, id, version uint, item *model.CarrierProfile) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *carrierProfileRepository) UpdateAudited(ctx context.Context, id, version uint, item *model.CarrierProfile, audit *model.AuditLog) error {
	return r.store.UpdateAudited(ctx, id, version, item, audit)
}
func (r *carrierProfileRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *carrierProfileRepository) DeleteAudited(ctx context.Context, id uint, audit *model.AuditLog) error {
	return r.store.DeleteAudited(ctx, id, audit)
}
func (r *carrierProfileRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
