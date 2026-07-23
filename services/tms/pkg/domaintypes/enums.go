package domaintypes

import "errors"

var (
	ErrInvalidStatus          = errors.New("invalid status")
	ErrInvalidEquipmentStatus = errors.New("invalid equipment status")
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
)

func (s Status) String() string {
	return string(s)
}

func StatusFromString(s string) (Status, error) {
	switch s {
	case "Active":
		return StatusActive, nil
	case "Inactive":
		return StatusInactive, nil
	default:
		return "", ErrInvalidStatus
	}
}

type EquipmentStatus string

const (
	EquipmentStatusAvailable     = EquipmentStatus("Available")
	EquipmentStatusOOS           = EquipmentStatus("OutOfService")
	EquipmentStatusAtMaintenance = EquipmentStatus("AtMaintenance")
	EquipmentStatusSold          = EquipmentStatus("Sold")
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
		return "", ErrInvalidEquipmentStatus
	}
}

type OwnershipType string

const (
	OwnershipTypeCompanyOwned  = OwnershipType("CompanyOwned")
	OwnershipTypeLeased        = OwnershipType("Leased")
	OwnershipTypeOwnerOperator = OwnershipType("OwnerOperator")
)

func (o OwnershipType) String() string {
	return string(o)
}

func (o OwnershipType) IsValid() bool {
	switch o {
	case OwnershipTypeCompanyOwned, OwnershipTypeLeased, OwnershipTypeOwnerOperator:
		return true
	default:
		return false
	}
}

type TimeFormat string

const (
	TimeFormat12Hour TimeFormat = "12-hour"
	TimeFormat24Hour TimeFormat = "24-hour"
)
