package model

import "time"

// CarrierProfile models 承运资质 as an independently versioned aggregate. The fields
// cover ownership, operational context, evidence and measured risk so later
// changes naturally span persistence, service and UI layers.
type CarrierProfile struct {
	BaseModel
	LicenseNumber    string    `json:"licenseNumber" gorm:"size:80;uniqueIndex;not null"`
	LicenseExpiresAt time.Time `json:"licenseExpiresAt" gorm:"index;not null"`
	VehicleCount     int       `json:"vehicleCount" gorm:"not null"`
	Facility         string    `json:"facility" gorm:"size:120;index"`
	Owner            string    `json:"owner" gorm:"size:120;index"`
	Category         string    `json:"category" gorm:"size:80;index"`
	RiskLevel        string    `json:"riskLevel" gorm:"size:32;index"`
	MetricValue      float64   `json:"metricValue"`
	MetricUnit       string    `json:"metricUnit" gorm:"size:24"`
	EffectiveAt      time.Time `json:"effectiveAt"`
	Evidence         string    `json:"evidence" gorm:"size:2000"`
	RelatedCode      string    `json:"relatedCode" gorm:"size:64;index"`
}

func (item *CarrierProfile) GetBase() *BaseModel { return &item.BaseModel }

func (item CarrierProfile) TableName() string { return "carrier_profiles" }

var CarrierProfileInitialStatus = "pending"
