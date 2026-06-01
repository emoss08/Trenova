package servicefailure

import "github.com/emoss08/trenova/internal/core/domain/shipment"

type ReasonCategory string

const (
	ReasonCategoryCarrier       ReasonCategory = "Carrier"
	ReasonCategoryCustomer      ReasonCategory = "Customer"
	ReasonCategoryFacility      ReasonCategory = "Facility"
	ReasonCategoryWeather       ReasonCategory = "Weather"
	ReasonCategoryEquipment     ReasonCategory = "Equipment"
	ReasonCategoryDocumentation ReasonCategory = "Documentation"
	ReasonCategoryOther         ReasonCategory = "Other"
)

type ReasonCodeAppliesTo string

const (
	ReasonCodeAppliesToPickup   ReasonCodeAppliesTo = "Pickup"
	ReasonCodeAppliesToDelivery ReasonCodeAppliesTo = "Delivery"
	ReasonCodeAppliesToBoth     ReasonCodeAppliesTo = "Both"
)

type Type string

const (
	TypeLatePickup   Type = "LatePickup"
	TypeLateDelivery Type = "LateDelivery"
)

type Source string

const (
	SourceDetected Source = "Detected"
	SourceManual   Source = "Manual"
)

type Status string

const (
	StatusOpen     Status = "Open"
	StatusReviewed Status = "Reviewed"
	StatusResolved Status = "Resolved"
	StatusVoided   Status = "Voided"
)

func (c ReasonCategory) IsValid() bool {
	switch c {
	case ReasonCategoryCarrier,
		ReasonCategoryCustomer,
		ReasonCategoryFacility,
		ReasonCategoryWeather,
		ReasonCategoryEquipment,
		ReasonCategoryDocumentation,
		ReasonCategoryOther:
		return true
	default:
		return false
	}
}

func (a ReasonCodeAppliesTo) IsValid() bool {
	switch a {
	case ReasonCodeAppliesToPickup, ReasonCodeAppliesToDelivery, ReasonCodeAppliesToBoth:
		return true
	default:
		return false
	}
}

func (a ReasonCodeAppliesTo) AllowsStopType(stopType shipment.StopType) bool {
	switch a {
	case ReasonCodeAppliesToBoth:
		return true
	case ReasonCodeAppliesToPickup:
		return stopType == shipment.StopTypePickup || stopType == shipment.StopTypeSplitPickup
	case ReasonCodeAppliesToDelivery:
		return stopType == shipment.StopTypeDelivery || stopType == shipment.StopTypeSplitDelivery
	default:
		return false
	}
}

func (t Type) IsValid() bool {
	switch t {
	case TypeLatePickup, TypeLateDelivery:
		return true
	default:
		return false
	}
}

func (s Source) IsValid() bool {
	switch s {
	case SourceDetected, SourceManual:
		return true
	default:
		return false
	}
}

func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusReviewed, StatusResolved, StatusVoided:
		return true
	default:
		return false
	}
}

func TypeForStop(stop *shipment.Stop) Type {
	if stop != nil && stop.IsOriginStop() {
		return TypeLatePickup
	}
	return TypeLateDelivery
}

func AppliesToForStop(stop *shipment.Stop) ReasonCodeAppliesTo {
	if stop != nil && stop.IsOriginStop() {
		return ReasonCodeAppliesToPickup
	}
	return ReasonCodeAppliesToDelivery
}
