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

type WasteGeneratorService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.WasteGenerator], error)
	Get(context.Context, uint) (model.WasteGenerator, error)
	Create(context.Context, dto.CreateWasteGenerator, string, string) (model.WasteGenerator, error)
	Update(context.Context, uint, dto.UpdateWasteGenerator, string, string) (model.WasteGenerator, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.WasteGenerator, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type wasteGeneratorService struct {
	repository repository.WasteGeneratorRepository
	security   SecurityService
}

func NewWasteGeneratorService(repo repository.WasteGeneratorRepository, security SecurityService) WasteGeneratorService {
	return &wasteGeneratorService{repository: repo, security: security}
}

func (s *wasteGeneratorService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.WasteGenerator], error) {
	return s.repository.List(ctx, query)
}

func (s *wasteGeneratorService) Get(ctx context.Context, id uint) (model.WasteGenerator, error) {
	return s.repository.Get(ctx, id)
}

func (s *wasteGeneratorService) Create(ctx context.Context, input dto.CreateWasteGenerator, actor, requestID string) (model.WasteGenerator, error) {
	if err := validateWasteGeneratorBusinessFields(input.Code, input.Name, input.Facility, input.Owner, input.PermitNumber, input.WasteCategories, input.Evidence, input.PermitExpiresAt); err != nil {
		return model.WasteGenerator{}, err
	}
	item := model.WasteGenerator{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.WasteGeneratorInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		PermitNumber: strings.ToUpper(strings.TrimSpace(input.PermitNumber)), PermitExpiresAt: input.PermitExpiresAt.UTC(),
		WasteCategories: strings.TrimSpace(input.WasteCategories),
		Facility:        strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.CreateAudited(ctx, &item, newAuditLog(actor, requestID, "create", "WasteGenerator", "", item.Status, "created generator permit")); err != nil {
		return model.WasteGenerator{}, fmt.Errorf("create 产废单位: %w", err)
	}
	return item, nil
}

func (s *wasteGeneratorService) Update(ctx context.Context, id uint, input dto.UpdateWasteGenerator, actor, requestID string) (model.WasteGenerator, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.WasteGenerator{}, err
	}
	if err := validateWasteGeneratorBusinessFields(current.Code, input.Name, input.Facility, input.Owner, input.PermitNumber, input.WasteCategories, input.Evidence, input.PermitExpiresAt); err != nil {
		return model.WasteGenerator{}, err
	}
	current.Name = strings.TrimSpace(input.Name)
	current.PermitNumber = strings.ToUpper(strings.TrimSpace(input.PermitNumber))
	current.PermitExpiresAt = input.PermitExpiresAt.UTC()
	current.WasteCategories = strings.TrimSpace(input.WasteCategories)
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
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "update", "WasteGenerator", current.Status, current.Status, "updated generator permit and evidence")); err != nil {
		return model.WasteGenerator{}, fmt.Errorf("update 产废单位: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *wasteGeneratorService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.WasteGenerator, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.WasteGenerator{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.WasteGeneratorTransitions, current.Status, target) {
		return model.WasteGenerator{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == "active" && !current.PermitExpiresAt.After(time.Now().UTC()) {
		return model.WasteGenerator{}, fmt.Errorf("%w: expired permit cannot be activated", ErrInvalidInput)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "transition", "WasteGenerator", before, target, input.Reason)); err != nil {
		return model.WasteGenerator{}, fmt.Errorf("transition 产废单位: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *wasteGeneratorService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.repository.DeleteAudited(ctx, id, newAuditLog(actor, requestID, "delete", "WasteGenerator", current.Status, "deleted", "soft deleted generator permit"))
}

func (s *wasteGeneratorService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateWasteGeneratorBusinessFields(code, name, facility, owner, permitNumber, categories, evidence string, expiresAt time.Time) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" || strings.TrimSpace(permitNumber) == "" || strings.TrimSpace(categories) == "" {
		return fmt.Errorf("%w: generator identity and permit fields are required", ErrInvalidInput)
	}
	if strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("%w: permit evidence reference is required", ErrInvalidInput)
	}
	if !expiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: generator permit must not be expired", ErrInvalidInput)
	}
	return nil
}
