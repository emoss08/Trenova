package models

import "github.com/google/uuid"

type ShipmentControl struct {
	TimeStampedModel
	BusinessUnitID           uuid.UUID `gorm:"type:uuid;not null;index"            json:"businessUnitId"`
	OrganizationID           uuid.UUID `gorm:"type:uuid;not null;unique"           json:"organizationId"`
	AutoRateShipment         bool      `gorm:"type:boolean;not null;default:true"  json:"autoRateShipment"         validate:"omitempty"`
	CalculateDistance        bool      `gorm:"type:boolean;not null;default:true"  json:"calculateDistance"        validate:"omitempty"`
	EnforceRevCode           bool      `gorm:"type:boolean;not null;default:false" json:"enforceRevCode"           validate:"omitempty"`
	EnforceVoidedComm        bool      `gorm:"type:boolean;not null;default:false" json:"enforceVoidedComm"        validate:"omitempty"`
	GenerateRoutes           bool      `gorm:"type:boolean;not null;default:false" json:"generateRoutes"           validate:"omitempty"`
	EnforceCommodity         bool      `gorm:"type:boolean;not null;default:false" json:"enforceCommodity"         validate:"omitempty"`
	AutoSequenceStops        bool      `gorm:"type:boolean;not null;default:true"  json:"autoSequenceStops"        validate:"omitempty"`
	AutoShipmentTotal        bool      `gorm:"type:boolean;not null;default:true"  json:"autoShipmentTotal"        validate:"omitempty"`
	EnforceOriginDestination bool      `gorm:"type:boolean;not null;default:false" json:"enforceOriginDestination" validate:"omitempty"`
	CheckForDuplicateBOL     bool      `gorm:"type:boolean;not null;default:false" json:"checkForDuplicateBol"     validate:"omitempty"`
	SendPlacardInfo          bool      `gorm:"type:boolean;not null;default:false" json:"sendPlacardInfo"          validate:"omitempty"`
	EnforceHazmatSegRules    bool      `gorm:"type:boolean;not null;default:true"  json:"enforceHazmatSegRules"    validate:"omitempty"`
}
