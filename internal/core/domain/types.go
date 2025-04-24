package domain

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/rotisserie/eris"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

func StatusFromString(s string) (Status, error) {
	switch s {
	case "Active":
		return StatusActive, nil
	case "Inactive":
		return StatusInactive, nil
	default:
		return "", eris.New("invalid status")
	}
}

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

func EquipmentStatusFromString(s string) (EquipmentStatus, error) {
	switch s {
	case "Available":
		return EquipmentStatusAvailable, nil
	case "OutOfService":
		return EquipmentStatusOOS, nil
	case "AtMaintenance":
		return EquipmentStatusAtMaintenance, nil
	case "Sold":
		return EquipmentStatusSold, nil
	default:
		return "", eris.New("invalid equipment status")
	}
}

type RoutingProvider string

const (
	// PCMiler is the provider for PCMiler
	RoutingProviderPCMiler = RoutingProvider("PCMiler")
)

type Validatable interface {
	Validate(ctx context.Context, multiErr *errors.MultiError)
	GetTableName() string
}
