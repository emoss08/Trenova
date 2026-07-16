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
	ReasonCategoryDriver        ReasonCategory = "Driver"
	ReasonCategoryShipper       ReasonCategory = "Shipper"
	ReasonCategoryConsignee     ReasonCategory = "Consignee"
	ReasonCategoryAppointment   ReasonCategory = "Appointment"
	ReasonCategoryOther         ReasonCategory = "Other"
)

type ReasonCodeAppliesTo string

const (
	ReasonCodeAppliesToPickup   ReasonCodeAppliesTo = "Pickup"
	ReasonCodeAppliesToDelivery ReasonCodeAppliesTo = "Delivery"
	ReasonCodeAppliesToBoth     ReasonCodeAppliesTo = "Both"
	ReasonCodeAppliesToAll      ReasonCodeAppliesTo = "All"
)

type Type string

const (
	TypeLatePickup        Type = "LatePickup"
	TypeLateDelivery      Type = "LateDelivery"
	TypeMissedPickup      Type = "MissedPickup"
	TypeMissedDelivery    Type = "MissedDelivery"
	TypeAppointmentMissed Type = "AppointmentMissed"
	TypeOther             Type = "Other"
)

type Source string

const (
	SourceDetected    Source = "Detected"
	SourceManual      Source = "Manual"
	SourceEDI         Source = "EDI"
	SourceIntegration Source = "Integration"
)

type Status string

const (
	StatusOpen     Status = "Open"
	StatusReviewed Status = "Reviewed"
	StatusResolved Status = "Resolved"
	StatusVoided   Status = "Voided"
)

func UnresolvedStatuses() []Status {
	return []Status{StatusOpen, StatusReviewed}
}

func (c ReasonCategory) IsValid() bool {
	switch c {
	case ReasonCategoryCarrier,
		ReasonCategoryCustomer,
		ReasonCategoryFacility,
		ReasonCategoryWeather,
		ReasonCategoryEquipment,
		ReasonCategoryDocumentation,
		ReasonCategoryDriver,
		ReasonCategoryShipper,
		ReasonCategoryConsignee,
		ReasonCategoryAppointment,
		ReasonCategoryOther:
		return true
	default:
		return false
	}
}

func (a ReasonCodeAppliesTo) IsValid() bool {
	switch a {
	case ReasonCodeAppliesToPickup, ReasonCodeAppliesToDelivery, ReasonCodeAppliesToBoth, ReasonCodeAppliesToAll:
		return true
	default:
		return false
	}
}

func (a ReasonCodeAppliesTo) AllowsStopType(stopType shipment.StopType) bool {
	switch a {
	case ReasonCodeAppliesToBoth, ReasonCodeAppliesToAll:
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
	case TypeLatePickup,
		TypeLateDelivery,
		TypeMissedPickup,
		TypeMissedDelivery,
		TypeAppointmentMissed,
		TypeOther:
		return true
	default:
		return false
	}
}

func (s Source) IsValid() bool {
	switch s {
	case SourceDetected, SourceManual, SourceEDI, SourceIntegration:
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
