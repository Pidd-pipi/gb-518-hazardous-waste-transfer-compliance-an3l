package dto

import "time"

// CreateTransferManifest is the public write contract for 转运清单. Status is deliberately
// omitted so callers cannot bypass the service state machine.
type CreateTransferManifest struct {
	Code          string    `json:"code" binding:"required,min=2,max=64"`
	Name          string    `json:"name" binding:"required,min=2,max=160"`
	GeneratorCode string    `json:"generatorCode" binding:"required,min=2,max=64"`
	CarrierCode   string    `json:"carrierCode" binding:"required,min=2,max=64"`
	WasteCode     string    `json:"wasteCode" binding:"required,min=2,max=64"`
	QuantityKg    float64   `json:"quantityKg" binding:"required,gt=0"`
	Destination   string    `json:"destination" binding:"required,min=2,max=200"`
	Description   string    `json:"description" binding:"max=1000"`
	Facility      string    `json:"facility" binding:"required,max=120"`
	Owner         string    `json:"owner" binding:"required,max=120"`
	Category      string    `json:"category" binding:"required,max=80"`
	RiskLevel     string    `json:"riskLevel" binding:"required,oneof=low medium high critical"`
	MetricValue   float64   `json:"metricValue"`
	MetricUnit    string    `json:"metricUnit" binding:"max=24"`
	EffectiveAt   time.Time `json:"effectiveAt" binding:"required"`
	Evidence      string    `json:"evidence" binding:"max=2000"`
	RelatedCode   string    `json:"relatedCode" binding:"max=64"`
}

type UpdateTransferManifest struct {
	ExpectedVersion uint      `json:"expectedVersion" binding:"required"`
	Name            string    `json:"name" binding:"required,min=2,max=160"`
	GeneratorCode   string    `json:"generatorCode" binding:"required,min=2,max=64"`
	CarrierCode     string    `json:"carrierCode" binding:"required,min=2,max=64"`
	WasteCode       string    `json:"wasteCode" binding:"required,min=2,max=64"`
	QuantityKg      float64   `json:"quantityKg" binding:"required,gt=0"`
	Destination     string    `json:"destination" binding:"required,min=2,max=200"`
	Description     string    `json:"description" binding:"max=1000"`
	Facility        string    `json:"facility" binding:"required,max=120"`
	Owner           string    `json:"owner" binding:"required,max=120"`
	Category        string    `json:"category" binding:"required,max=80"`
	RiskLevel       string    `json:"riskLevel" binding:"required,oneof=low medium high critical"`
	MetricValue     float64   `json:"metricValue"`
	MetricUnit      string    `json:"metricUnit" binding:"max=24"`
	EffectiveAt     time.Time `json:"effectiveAt" binding:"required"`
	Evidence        string    `json:"evidence" binding:"max=2000"`
	RelatedCode     string    `json:"relatedCode" binding:"max=64"`
}
