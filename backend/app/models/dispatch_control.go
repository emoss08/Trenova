package models

import "github.com/google/uuid"

type DispatchControl struct {
	TimeStampedModel
	BusinessUnitID               uuid.UUID           `gorm:"type:uuid;not null;index"                                                 json:"businessUnitId"`
	OrganizationID               uuid.UUID           `gorm:"type:uuid;not null;unique"                                                json:"organizationId"`
	RecordServiceIncident        ServiceIncidentType `gorm:"type:varchar(3);not null;default:'N'"                                     json:"recordServiceIncident"        validate:"required,oneof=N P PD AEP"`
	DeadheadTarget               *float64            `gorm:"type:numeric(5,2);default:0.00"                                           json:"deadheadTarget"               validate:"omitempty"`
	MaxShipmentWeightLimit       int                 `gorm:"type:integer;check:max_shipment_weight_limit >= 0;not null;default:80000" json:"maxShipmentWeightLimit"       validate:"required"`
	GracePeriod                  uint8               `gorm:"type:smallint;check:grace_period >= 0;not null;default:0"                 json:"gracePeriod"                  validate:"required"`
	EnforceWorkerAssign          bool                `gorm:"type:boolean;not null;default:true"                                       json:"enforceWorkerAssign"          validate:"omitempty"`
	TrailerContinuity            bool                `gorm:"type:boolean;not null;default:false"                                      json:"trailerContinuity"            validate:"omitempty"`
	DupeTrailerCheck             bool                `gorm:"type:boolean;not null;default:false"                                      json:"dupeTrailerCheck"             validate:"omitempty"`
	MaintenanceCompliance        bool                `gorm:"type:boolean;not null;default:true"                                       json:"maintenanceCompliance"        validate:"omitempty"`
	RegulatoryCheck              bool                `gorm:"type:boolean;not null;default:false"                                      json:"regulatoryCheck"              validate:"omitempty"`
	PrevShipmentOnHold           bool                `gorm:"type:boolean;not null;default:false"                                      json:"prevShipmentOnHold"           validate:"omitempty"`
	WorkerTimeAwayRestriction    bool                `gorm:"type:boolean;not null;default:true"                                       json:"workerTimeAwayRestriction"    validate:"omitempty"`
	TractorWorkerFleetConstraint bool                `gorm:"type:boolean;not null;default:false"                                      json:"tractorWorkerFleetConstraint" validate:"omitempty"`
}
