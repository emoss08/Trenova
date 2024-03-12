package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type (
	RouteDistanceUnitType string
	DistanceMethodType    string
)

const (
	// RduMetric Metric is the same as Kilometers.
	RduMetric RouteDistanceUnitType = "M"

	// RduImperial Imperial is the same as Miles.
	RduImperial           RouteDistanceUnitType = "I"
	GoogleDistanceMethod  DistanceMethodType    = "G"
	TrenovaDistanceMethod DistanceMethodType    = "T"
)

type RouteControl struct {
	TimeStampedModel
	BusinessUnitID uuid.UUID             `gorm:"type:uuid;not null;index"             json:"businessUnitId"`
	OrganizationID uuid.UUID             `gorm:"type:uuid;not null;unique"            json:"organizationId"`
	DistanceMethod DistanceMethodType    `gorm:"type:varchar(1);not null;default:'T'" json:"distanceMethod" validate:"required,oneof=T G"`
	MileageUnit    RouteDistanceUnitType `gorm:"type:varchar(1);not null;default:'M'" json:"mileageUnit"    validate:"required,oneof=M I"`
	GenerateRoutes bool                  `gorm:"type:boolean;not null;default:false"  json:"generateRoutes" validate:"omitempty"`
}

func (rc *RouteControl) BeforeCreate(_ *gorm.DB) error {
	return rc.validateRouteControl()
}

func (rc *RouteControl) BeforeUpdate(_ *gorm.DB) error {
	return rc.validateRouteControl()
}

var errGenerateRoutesWithTrenova = errors.New("cannot use generate routes with Trenova distance method")

func (rc *RouteControl) validateRouteControl() error {
	// Disallow using the generate routes if the distance method is set to Trenova
	if rc.DistanceMethod == TrenovaDistanceMethod && rc.GenerateRoutes {
		return &AttributeError{
			Attr:    "generateRoutes",
			Message: errGenerateRoutesWithTrenova.Error(),
		}
	}

	return nil
}
