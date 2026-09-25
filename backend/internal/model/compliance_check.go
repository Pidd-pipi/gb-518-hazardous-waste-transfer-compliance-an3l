package model

import "time"

// ComplianceCheck models 合规核验 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
type ComplianceCheck struct {
	BaseModel
	ManifestCode  string    `json:"manifestCode" gorm:"size:64;index;not null"`
	Checklist     string    `json:"checklist" gorm:"size:2000;not null"`
	DecisionBasis string    `json:"decisionBasis" gorm:"size:1000"`
	Facility      string    `json:"facility" gorm:"size:120;index"`
	Owner         string    `json:"owner" gorm:"size:120;index"`
	Category      string    `json:"category" gorm:"size:80;index"`
	RiskLevel     string    `json:"riskLevel" gorm:"size:32;index"`
	MetricValue   float64   `json:"metricValue"`
	MetricUnit    string    `json:"metricUnit" gorm:"size:24"`
	EffectiveAt   time.Time `json:"effectiveAt"`
	Evidence      string    `json:"evidence" gorm:"size:2000"`
	RelatedCode   string    `json:"relatedCode" gorm:"size:64;index"`
}

func (item *ComplianceCheck) GetBase() *BaseModel { return &item.BaseModel }

func (item ComplianceCheck) TableName() string { return "compliance_checks" }

var ComplianceCheckInitialStatus = "pending"
