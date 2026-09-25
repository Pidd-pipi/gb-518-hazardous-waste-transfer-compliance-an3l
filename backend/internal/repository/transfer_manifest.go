package repository

import (
	"context"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"gorm.io/gorm"
)

// TransferManifestRepository owns all persistence operations for 转运清单.
type TransferManifestRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.TransferManifest], error)
	Get(context.Context, uint) (model.TransferManifest, error)
	FindByCode(context.Context, string) (model.TransferManifest, error)
	Create(context.Context, *model.TransferManifest) error
	CreateAudited(context.Context, *model.TransferManifest, *model.AuditLog) error
	Update(context.Context, uint, uint, *model.TransferManifest) error
	UpdateAudited(context.Context, uint, uint, *model.TransferManifest, *model.AuditLog) error
	Delete(context.Context, uint) error
	DeleteAudited(context.Context, uint, *model.AuditLog) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type transferManifestRepository struct {
	store *Store[model.TransferManifest]
}

func NewTransferManifestRepository(db *gorm.DB) TransferManifestRepository {
	return &transferManifestRepository{store: NewStore[model.TransferManifest](db)}
}

func (r *transferManifestRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.TransferManifest], error) {
	return r.store.List(ctx, q)
}
func (r *transferManifestRepository) Get(ctx context.Context, id uint) (model.TransferManifest, error) {
	return r.store.Get(ctx, id)
}
func (r *transferManifestRepository) FindByCode(ctx context.Context, code string) (model.TransferManifest, error) {
	return r.store.FindByCode(ctx, code)
}
func (r *transferManifestRepository) Create(ctx context.Context, item *model.TransferManifest) error {
	return r.store.Create(ctx, item)
}
func (r *transferManifestRepository) CreateAudited(ctx context.Context, item *model.TransferManifest, audit *model.AuditLog) error {
	return r.store.CreateAudited(ctx, item, audit)
}
func (r *transferManifestRepository) Update(ctx context.Context, id, version uint, item *model.TransferManifest) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *transferManifestRepository) UpdateAudited(ctx context.Context, id, version uint, item *model.TransferManifest, audit *model.AuditLog) error {
	return r.store.UpdateAudited(ctx, id, version, item, audit)
}
func (r *transferManifestRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *transferManifestRepository) DeleteAudited(ctx context.Context, id uint, audit *model.AuditLog) error {
	return r.store.DeleteAudited(ctx, id, audit)
}
func (r *transferManifestRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
