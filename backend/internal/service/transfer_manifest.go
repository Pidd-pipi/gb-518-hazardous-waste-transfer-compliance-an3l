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

type TransferManifestService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.TransferManifest], error)
	Get(context.Context, uint) (model.TransferManifest, error)
	Create(context.Context, dto.CreateTransferManifest, string, string) (model.TransferManifest, error)
	Update(context.Context, uint, dto.UpdateTransferManifest, string, string) (model.TransferManifest, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.TransferManifest, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type transferManifestService struct {
	repository repository.TransferManifestRepository
	generators repository.WasteGeneratorRepository
	carriers   repository.CarrierProfileRepository
}

func NewTransferManifestService(repo repository.TransferManifestRepository, generators repository.WasteGeneratorRepository, carriers repository.CarrierProfileRepository) TransferManifestService {
	return &transferManifestService{repository: repo, generators: generators, carriers: carriers}
}

func (s *transferManifestService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.TransferManifest], error) {
	return s.repository.List(ctx, query)
}

func (s *transferManifestService) Get(ctx context.Context, id uint) (model.TransferManifest, error) {
	return s.repository.Get(ctx, id)
}

func (s *transferManifestService) Create(ctx context.Context, input dto.CreateTransferManifest, actor, requestID string) (model.TransferManifest, error) {
	if err := validateTransferManifestBusinessFields(input.Code, input.Name, input.Facility, input.Owner, input.GeneratorCode, input.CarrierCode, input.WasteCode, input.Destination, input.Evidence, input.QuantityKg); err != nil {
		return model.TransferManifest{}, err
	}
	if _, err := s.generators.FindByCode(ctx, input.GeneratorCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: generator %s does not exist", ErrInvalidInput, input.GeneratorCode)
	}
	if _, err := s.carriers.FindByCode(ctx, input.CarrierCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: carrier %s does not exist", ErrInvalidInput, input.CarrierCode)
	}
	item := model.TransferManifest{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.TransferManifestInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		GeneratorCode: strings.ToUpper(strings.TrimSpace(input.GeneratorCode)), CarrierCode: strings.ToUpper(strings.TrimSpace(input.CarrierCode)),
		WasteCode: strings.ToUpper(strings.TrimSpace(input.WasteCode)), QuantityKg: input.QuantityKg, Destination: strings.TrimSpace(input.Destination),
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.CreateAudited(ctx, &item, newAuditLog(actor, requestID, "create", "TransferManifest", "", item.Status, "created linked transfer manifest")); err != nil {
		return model.TransferManifest{}, fmt.Errorf("create 转运清单: %w", err)
	}
	return item, nil
}

func (s *transferManifestService) Update(ctx context.Context, id uint, input dto.UpdateTransferManifest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	if current.Status != "draft" {
		return model.TransferManifest{}, fmt.Errorf("%w: only draft manifests can be edited", ErrInvalidInput)
	}
	if err := validateTransferManifestBusinessFields(current.Code, input.Name, input.Facility, input.Owner, input.GeneratorCode, input.CarrierCode, input.WasteCode, input.Destination, input.Evidence, input.QuantityKg); err != nil {
		return model.TransferManifest{}, err
	}
	if _, err := s.generators.FindByCode(ctx, input.GeneratorCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: generator %s does not exist", ErrInvalidInput, input.GeneratorCode)
	}
	if _, err := s.carriers.FindByCode(ctx, input.CarrierCode); err != nil {
		return model.TransferManifest{}, fmt.Errorf("%w: carrier %s does not exist", ErrInvalidInput, input.CarrierCode)
	}
	current.Name = strings.TrimSpace(input.Name)
	current.GeneratorCode = strings.ToUpper(strings.TrimSpace(input.GeneratorCode))
	current.CarrierCode = strings.ToUpper(strings.TrimSpace(input.CarrierCode))
	current.WasteCode = strings.ToUpper(strings.TrimSpace(input.WasteCode))
	current.QuantityKg = input.QuantityKg
	current.Destination = strings.TrimSpace(input.Destination)
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
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "update", "TransferManifest", current.Status, current.Status, "updated draft manifest and evidence")); err != nil {
		return model.TransferManifest{}, fmt.Errorf("update 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

// ReceivedWeightTolerance 是计划重量允许的现场偏差上限（5%）。超过上限的签收
// 不允许直接完成，必须带差异原因转为驳回。
const ReceivedWeightTolerance = 0.05

func (s *transferManifestService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.TransferManifest, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.TransferManifest{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.TransferManifestTransitions, current.Status, target) {
		return model.TransferManifest{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == "submitted" || target == "in_transit" {
		if err := s.validateLinkedParties(ctx, current); err != nil {
			return model.TransferManifest{}, err
		}
	}

	before := current.Status
	detail := strings.TrimSpace(input.Reason)

	if target == "received" || (target == "rejected" && current.Status == "in_transit") {
		receivedWeight, diff, overweight, err := validateReceivedWeight(current.QuantityKg, input.ReceivedWeightKg, target == "received")
		if err != nil {
			return model.TransferManifest{}, err
		}
		diffReason := strings.TrimSpace(input.WeightDiffReason)
		if target == "received" && overweight {
			// 超出计划重量 5%：签收不能完成，转驳回并必须填写差异原因。
			if diffReason == "" {
				return model.TransferManifest{}, fmt.Errorf("%w: received weight exceeds planned weight by %.0f%%; manifest must be rejected with a difference reason", ErrInvalidInput, ReceivedWeightTolerance*100)
			}
			target = "rejected"
			applyReceivedWeight(&current, receivedWeight, diff, diffReason)
			detail = fmt.Sprintf("received %.3f kg exceeds planned %.3f kg by more than %.0f%%, rejected: %s", receivedWeight, current.QuantityKg, ReceivedWeightTolerance*100, diffReason)
		} else if receivedWeight > 0 {
			applyReceivedWeight(&current, receivedWeight, diff, diffReason)
			if target == "rejected" {
				detail = fmt.Sprintf("rejected at site after weighing %.3f kg (diff %+.3f kg): %s", receivedWeight, diff, joinReason(detail, diffReason))
			} else {
				detail = fmt.Sprintf("received %.3f kg (planned %.3f kg, diff %+.3f kg): %s", receivedWeight, current.QuantityKg, diff, detail)
			}
		}
	}

	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateAudited(ctx, id, input.ExpectedVersion, &current, newAuditLog(actor, requestID, "transition", "TransferManifest", before, target, detail)); err != nil {
		return model.TransferManifest{}, fmt.Errorf("transition 转运清单: %w", err)
	}
	return s.repository.Get(ctx, id)
}

// validateReceivedWeight 校验签收/现场称重驳回携带的实收重量，返回实收值、与计划
// 的差值（实收-计划）以及是否超出 5% 容差。
func validateReceivedWeight(planned float64, received *float64, required bool) (float64, float64, bool, error) {
	if received == nil {
		if required {
			return 0, 0, false, fmt.Errorf("%w: received weight is required before a manifest can be signed for receipt", ErrInvalidInput)
		}
		return 0, 0, false, nil
	}
	if *received <= 0 {
		return 0, 0, false, fmt.Errorf("%w: received weight must be a positive value", ErrInvalidInput)
	}
	diff := *received - planned
	overweight := diff > planned*ReceivedWeightTolerance
	return *received, diff, overweight, nil
}

// applyReceivedWeight 在同一时间点写入实收重量、与计划的差值、差异原因和签收时间。
func applyReceivedWeight(manifest *model.TransferManifest, receivedWeight, diff float64, diffReason string) {
	receivedAt := time.Now().UTC()
	manifest.ReceivedWeightKg = &receivedWeight
	manifest.WeightDiffKg = &diff
	manifest.WeightDiffReason = diffReason
	manifest.ReceivedAt = &receivedAt
}

func joinReason(reason, diffReason string) string {
	if diffReason == "" {
		return reason
	}
	return reason + " | difference reason: " + diffReason
}

func (s *transferManifestService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.Status != "draft" {
		return fmt.Errorf("%w: submitted manifests must be retained for compliance", ErrInvalidInput)
	}
	return s.repository.DeleteAudited(ctx, id, newAuditLog(actor, requestID, "delete", "TransferManifest", current.Status, "deleted", "soft deleted draft manifest"))
}

func (s *transferManifestService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func (s *transferManifestService) validateLinkedParties(ctx context.Context, manifest model.TransferManifest) error {
	generator, err := s.generators.FindByCode(ctx, manifest.GeneratorCode)
	if err != nil {
		return fmt.Errorf("%w: linked generator is unavailable", ErrInvalidInput)
	}
	if generator.Status != "active" || !generator.PermitExpiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: generator permit must be active and unexpired", ErrInvalidInput)
	}
	carrier, err := s.carriers.FindByCode(ctx, manifest.CarrierCode)
	if err != nil {
		return fmt.Errorf("%w: linked carrier is unavailable", ErrInvalidInput)
	}
	if carrier.Status != "verified" || !carrier.LicenseExpiresAt.After(time.Now().UTC()) {
		return fmt.Errorf("%w: carrier license must be verified and unexpired", ErrInvalidInput)
	}
	return nil
}

func validateTransferManifestBusinessFields(code, name, facility, owner, generatorCode, carrierCode, wasteCode, destination, evidence string, quantityKg float64) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" || strings.TrimSpace(generatorCode) == "" || strings.TrimSpace(carrierCode) == "" || strings.TrimSpace(wasteCode) == "" || strings.TrimSpace(destination) == "" {
		return fmt.Errorf("%w: manifest identity, parties and route are required", ErrInvalidInput)
	}
	if quantityKg <= 0 || strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("%w: positive waste quantity and manifest evidence are required", ErrInvalidInput)
	}
	return nil
}
