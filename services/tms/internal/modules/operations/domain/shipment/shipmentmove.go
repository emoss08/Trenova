package shipment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const maxDistanceFieldLength = 50

// Movement legs within a shipment
type ShipmentMove struct {
	keel.Table `keel:"shipment_moves,report=Operations"`

	ID                     pulid.ID           `json:"id"                          keel:"id,pk,autoid:sm_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"              keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"              keel:"organization_id,pk,tenant"`
	ShipmentID             pulid.ID           `json:"shipmentId"                  keel:"shipment_id"`
	Status                 MoveStatus         `json:"status"                      keel:"status"`
	CoverageType           MoveCoverageType   `json:"coverageType"                keel:"coverage_type"`
	Loaded                 bool               `json:"loaded"                      keel:"loaded"`
	Sequence               int64              `json:"sequence"                    keel:"sequence,dimension"`
	Distance               *float64           `json:"distance"                    keel:"distance,label=Miles,format=distance"`
	DistanceSource         string             `json:"distanceSource"              keel:"distance_source,nullzero"`
	DistanceProvider       string             `json:"distanceProvider"            keel:"distance_provider,nullzero"`
	DistanceCalculatedAt   *int64             `json:"distanceCalculatedAt"        keel:"distance_calculated_at"`
	DistanceRouteSignature string             `json:"distanceRouteSignature"      keel:"distance_route_signature,nullzero"`
	DistanceDataVersion    string             `json:"distanceDataVersion"         keel:"distance_data_version,nullzero"`
	DistanceRoutingType    string             `json:"distanceRoutingType"         keel:"distance_routing_type,nullzero"`
	DistanceUnits          string             `json:"distanceUnits"               keel:"distance_units,nullzero"`
	DistanceMetadata       map[string]any     `json:"distanceMetadata,omitempty"  keel:"distance_metadata,jsonb,nullzero"`
	Version                int64              `json:"version"                     keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"                   keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"                   keel:"updated_at,autoupdate"`
	Stops                  []*Stop            `json:"stops,omitempty"             keel:"many:shipment_move_id"`
	Assignment             *Assignment        `json:"assignment,omitempty"        keel:"one:shipment_move_id"`
	CarrierAssignment      *CarrierAssignment `json:"carrierAssignment,omitempty" keel:"one:shipment_move_id,noreport"`
	Shipment               *Shipment          `json:"shipment,omitempty"          keel:"belongs:shipment_id"`

	JurisdictionMiles      []*ShipmentMoveJurisdictionMile `json:"jurisdictionMiles,omitempty" keel:"many:shipment_move_id,noreport"`
	JurisdictionMilesDirty bool                            `json:"-"`
}

func (sm *ShipmentMove) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning shipment are stamped by the repository when the
	// move is written, so they are not the caller's to supply and are not
	// checked here — a create would otherwise fail on fields it never sends.
	multiErr.AddOzzoError(validation.ValidateStruct(
		sm,
		validation.Field(
			&sm.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MoveStatus]("Status is invalid"),
		),
		validation.Field(
			&sm.CoverageType,
			validation.When(
				sm.CoverageType != "",
				domainvalidation.ValidEnum[MoveCoverageType]("Coverage type is invalid"),
			),
		),
		validation.Field(
			&sm.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		// Distance is what the move is paid and billed on, so a negative one
		// would credit mileage rather than charge it.
		validation.Field(
			&sm.Distance,
			validation.Min(0.0).Error("Distance cannot be negative"),
		),
		validation.Field(
			&sm.DistanceSource,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance source cannot be longer than 50 characters"),
		),
		validation.Field(
			&sm.DistanceProvider,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance provider cannot be longer than 50 characters"),
		),
		validation.Field(
			&sm.DistanceDataVersion,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance data version cannot be longer than 50 characters"),
		),
		validation.Field(
			&sm.DistanceRoutingType,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance routing type cannot be longer than 50 characters"),
		),
		validation.Field(
			&sm.DistanceUnits,
			validation.Length(0, maxDistanceFieldLength).
				Error("Distance units cannot be longer than 50 characters"),
		),
	))
}

func (m *ShipmentMove) BeforeCreate(ctx context.Context) error {
	if m.Status == "" {
		m.Status = MoveStatusNew
	}
	if m.CoverageType == "" {
		m.CoverageType = MoveCoverageTypeUnassigned
	}

	return nil
}

func (m *ShipmentMove) BeforeUpdate(ctx context.Context) error {
	if m.CoverageType == "" {
		m.CoverageType = MoveCoverageTypeUnassigned
	}

	return nil
}

func (m *ShipmentMove) IsCompleted() bool {
	return m.Status == MoveStatusCompleted
}

func (m *ShipmentMove) IsInTransit() bool {
	return m.Status == MoveStatusInTransit
}

func (m *ShipmentMove) IsAssigned() bool {
	return m.Status == MoveStatusAssigned
}

func (m *ShipmentMove) HasAssignment() bool {
	return m != nil && m.Assignment != nil
}

func (m *ShipmentMove) HasCarrierAssignment() bool {
	return m != nil && m.CarrierAssignment.IsActive()
}

// HasCoverage reports whether the move is covered by either a company driver
// assignment or an active external carrier assignment.
func (m *ShipmentMove) HasCoverage() bool {
	return m.HasAssignment() || m.HasCarrierAssignment()
}

func (m *ShipmentMove) IsCarrierCovered() bool {
	return m != nil && m.CoverageType == MoveCoverageTypeCarrier
}

func (m *ShipmentMove) IsNew() bool {
	return m.Status == MoveStatusNew
}

func (m *ShipmentMove) IsCanceled() bool {
	return m.Status == MoveStatusCanceled
}

func (m *ShipmentMove) JurisdictionMilesSumMiles() float64 {
	if m == nil {
		return 0
	}
	var total float64
	for _, row := range m.JurisdictionMiles {
		if row == nil {
			continue
		}
		total += row.DistanceInMiles()
	}
	return total
}
