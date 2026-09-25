package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/constants"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/dto"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/model"
	"github.com/blueship581/hazardous-waste-transfer-compliance/backend/internal/repository"
)

type CarrierProfileService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.CarrierProfile], error)
	Get(context.Context, uint) (model.CarrierProfile, error)
	Create(context.Context, dto.CreateCarrierProfile, string, string) (model.CarrierProfile, error)
	Update(context.Context, uint, dto.UpdateCarrierProfile, string, string) (model.CarrierProfile, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.CarrierProfile, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type carrierProfileService struct {
	repository repository.CarrierProfileRepository
	security   SecurityService
}

func NewCarrierProfileService(repo repository.CarrierProfileRepository, security SecurityService) CarrierProfileService {
	return &carrierProfileService{repository: repo, security: security}
}

func (s *carrierProfileService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.CarrierProfile], error) {
	return s.repository.List(ctx, query)
}

func (s *carrierProfileService) Get(ctx context.Context, id uint) (model.CarrierProfile, error) {
	return s.repository.Get(ctx, id)
}

func (s *carrierProfileService) Create(ctx context.Context, input dto.CreateCarrierProfile, actor, requestID string) (model.CarrierProfile, error) {
	if err := validateCarrierProfileBusinessFields(input.Code, input.Name, input.Facility, input.Owner, input.LicenseNumber, input.Evidence, input.VehicleCount, input.LicenseExpiresAt); err != nil {
		return model.CarrierProfile{}, err
	}
	item := model.CarrierProfile{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.CarrierProfileInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		LicenseNumber: strings.ToUpper(strings.TrimSpace(input.LicenseNumber)), LicenseExpiresAt: input.LicenseExpiresAt.UTC(), VehicleCount: input.VehicleCount,
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.CreateAudited(ctx, &item, newAuditLog(actor, requestID, "create", "CarrierProfile", "", item.Status, "created carrier license")); err != nil {
		return model.CarrierProfile{}, fmt.Errorf("create 承运资质: %w", err)
	}
	return item, nil
}

func (s *carrierProfileService) Update(ctx context.Context, id uint, input dto.UpdateCarrierProfile, actor, requestID string) (model.CarrierProfile, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.CarrierProfile{}, err
	}
	if err := validateCarrierProfileBusinessFields(current.Code, input.Name, input.Facility, input.Owner, input.LicenseNumber, input.Evidence, input.VehicleCount, input.LicenseExpiresAt); err != nil {
		return model.CarrierProfile{}, err
	}
	current.Name = strings.TrimSpace(input.Name)
	current.LicenseNumber = strings.ToUpper(strings.TrimSpace(input.LicenseNumber))
	current.LicenseExpiresAt = input.LicenseExpiresAt.UTC()
	current.VehicleCount = input.VehicleCount
	current.Description = strings.TrimSpace(input.Description)
	current.Facility = strings.TrimSpace(input.Facility)
	current.Owner = strings.TrimSpace(input.Owner)
	current.Category = strings.TrimSpace(input.Category)
	current.RiskLevel = input.RiskLevel
	current.MetricValue = input.MetricValue
	current.MetricUnit = strings.TrimSpace(input.MetricUnit)
	current.EffectiveAt = input.EffectiveAt.UTC()
	current.Evidence = strings.TrimSpace(input.Evidence)
	current.RelatedCode = strings.ToUpper(strings.TrimSpace(input.RelatedCode))
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "update", "CarrierProfile", current.Status, current.Status, "updated carrier license and evidence")); err != nil {
		return model.CarrierProfile{}, fmt.Errorf("update 承运资质: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *carrierProfileService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.CarrierProfile, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.CarrierProfile{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.CarrierProfileTransitions, current.Status, target) {
		return model.CarrierProfile{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == "verified" && !current.LicenseExpiresAt.After(time.Now().UTC()) {
		return model.CarrierProfile{}, fmt.Errorf("%w: expired carrier license cannot be verified", ErrInvalidInput)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "transition", "CarrierProfile", before, target, input.Reason)); err != nil {
		return model.CarrierProfile{}, fmt.Errorf("transition 承运资质: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *carrierProfileService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteAudited(ctx, id, newAuditLog(actor, requestID, "delete", "CarrierProfile", current.Status, "deleted", "soft deleted carrier license"))
}

func (s *carrierProfileService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateCarrierProfileBusinessFields(code, name, facility, owner, licenseNumber, evidence string, vehicleCount int, expiresAt time.Time) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" || strings.TrimSpace(licenseNumber) == "" {
		return fmt.Errorf("%w: carrier identity and license fields are required", ErrInvalidInput)
	}
	if strings.TrimSpace(evidence) == "" || vehicleCount < 1 {
		return fmt.Errorf("%w: license evidence and at least one vehicle are required", ErrInvalidInput)
	}
	if !expiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: carrier license must not be expired", ErrInvalidInput)
	}
	return nil
}
