package domain

import (
	"errors"
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
		return "", errors.New("invalid status")
	}
}

type Gender string

const (
	GenderMale   = Gender("Male")
	GenderFemale = Gender("Female")
)

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
		return "", errors.New("invalid equipment status")
	}
}
