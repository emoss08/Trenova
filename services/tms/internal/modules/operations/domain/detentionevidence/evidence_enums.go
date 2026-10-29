package detentionevidence

import (
	"errors"
)

type EvidenceKind string

const (
	EvidenceKindArrival      = EvidenceKind("Arrival")
	EvidenceKindDeparture    = EvidenceKind("Departure")
	EvidenceKindAppointment  = EvidenceKind("Appointment")
	EvidenceKindNotice       = EvidenceKind("Notice")
	EvidenceKindDocument     = EvidenceKind("Document")
	EvidenceKindStatusChange = EvidenceKind("StatusChange")
	EvidenceKindRecalculated = EvidenceKind("Recalculated")
	EvidenceKindWaiver       = EvidenceKind("Waiver")
	EvidenceKindDispute      = EvidenceKind("Dispute")
)

func (k EvidenceKind) String() string { return string(k) }

func EvidenceKindFromString(v string) (EvidenceKind, error) {
	switch v {
	case "Arrival":
		return EvidenceKindArrival, nil
	case "Departure":
		return EvidenceKindDeparture, nil
	case "Appointment":
		return EvidenceKindAppointment, nil
	case "Notice":
		return EvidenceKindNotice, nil
	case "Document":
		return EvidenceKindDocument, nil
	case "StatusChange":
		return EvidenceKindStatusChange, nil
	case "Recalculated":
		return EvidenceKindRecalculated, nil
	case "Waiver":
		return EvidenceKindWaiver, nil
	case "Dispute":
		return EvidenceKindDispute, nil
	default:
		return "", errors.New("invalid detention evidence kind")
	}
}

// EvidenceSource ranks how defensible a captured fact is in a dispute. The
// ordering drives the collectability score: a geofence crossing is testimony a
// customer rarely argues with, a dispatcher's keystroke is not.
type EvidenceSource string

const (
	EvidenceSourceTelematics = EvidenceSource("Telematics")
	EvidenceSourceGeofence   = EvidenceSource("Geofence")
	EvidenceSourceDriverApp  = EvidenceSource("DriverApp")
	EvidenceSourceEDI        = EvidenceSource("EDI")
	EvidenceSourceDocument   = EvidenceSource("Document")
	EvidenceSourceManual     = EvidenceSource("Manual")
	EvidenceSourceSystem     = EvidenceSource("System")
)

func (s EvidenceSource) String() string { return string(s) }

// Weight returns the defensibility contribution of a source, 0-100.
func (s EvidenceSource) Weight() int {
	switch s {
	case EvidenceSourceGeofence, EvidenceSourceTelematics:
		return 100
	case EvidenceSourceEDI:
		return 85
	case EvidenceSourceDocument:
		return 80
	case EvidenceSourceDriverApp:
		return 70
	case EvidenceSourceManual:
		return 35
	case EvidenceSourceSystem:
		return 50
	default:
		return 0
	}
}

func EvidenceSourceFromString(v string) (EvidenceSource, error) {
	switch v {
	case "Telematics":
		return EvidenceSourceTelematics, nil
	case "Geofence":
		return EvidenceSourceGeofence, nil
	case "DriverApp":
		return EvidenceSourceDriverApp, nil
	case "EDI":
		return EvidenceSourceEDI, nil
	case "Document":
		return EvidenceSourceDocument, nil
	case "Manual":
		return EvidenceSourceManual, nil
	case "System":
		return EvidenceSourceSystem, nil
	default:
		return "", errors.New("invalid detention evidence source")
	}
}
