package domain

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

type Gender string

const (
	// GenderMale is the male gender
	GenderMale = Gender("Male")

	// GenderFemale is the female gender
	GenderFemale = Gender("Female")
)

type EquipmentStatus string

const (
	// EquipmentStatusAvailable is the equipment is available for use
	EquipmentStatusAvailable = EquipmentStatus("Available")

	// EquipmentStatusOOS is the equipment is out of service
	EquipmentStatusOOS = EquipmentStatus("OutOfService")

	// EquipmentStatusAtMaintenance is the equipment is at maintenance
	EquipmentStatusAtMaintenance = EquipmentStatus("AtMaintenance")

	// EquipmentStatusSold is the equipment is sold
	EquipmentStatusSold = EquipmentStatus("Sold")
)

type RoutingProvider string

const (
	// PCMiler is the provider for PCMiler
	RoutingProviderPCMiler = RoutingProvider("PCMiler")
)

type Validatable interface {
	Validate(ctx context.Context, multiErr *errors.MultiError)
	GetTableName() string
}

