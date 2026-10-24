package telematics

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type VehicleInspectionDefect struct {
	ID         string `json:"id"`
	DefectType string `json:"defectType"`
	Comment    string `json:"comment,omitempty"`
	Resolved   bool   `json:"resolved"`
	ResolvedAt *int64 `json:"resolvedAt,omitempty"`
}

type VehicleInspection struct {
	keel.Table `keel:"vehicle_inspections"`

	ID                    pulid.ID                  `json:"id" keel:"id,pk"`
	OrganizationID        pulid.ID                  `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID                  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider              string                    `json:"provider" keel:"provider"`
	ProviderDvirID        string                    `json:"providerDvirId" keel:"provider_dvir_id"`
	TractorID             pulid.ID                  `json:"tractorId" keel:"tractor_id,nullzero"`
	WorkerID              pulid.ID                  `json:"workerId" keel:"worker_id,nullzero"`
	InspectionType        string                    `json:"inspectionType" keel:"inspection_type"`
	SafetyStatus          string                    `json:"safetyStatus" keel:"safety_status"`
	StartedAt             int64                     `json:"startedAt" keel:"started_at"`
	EndedAt               int64                     `json:"endedAt" keel:"ended_at"`
	OdometerMeters        *int64                    `json:"odometerMeters" keel:"odometer_meters"`
	Location              string                    `json:"location" keel:"location,nullzero"`
	Signed                bool                      `json:"signed" keel:"signed"`
	DefectCount           int                       `json:"defectCount" keel:"defect_count"`
	UnresolvedDefectCount int                       `json:"unresolvedDefectCount" keel:"unresolved_defect_count"`
	Defects               []VehicleInspectionDefect `json:"defects" keel:"defects,jsonb,nullzero"`
	CreatedAt             int64                     `json:"createdAt" keel:"created_at"`
}

func (i *VehicleInspection) BeforeCreate(_ context.Context) error {
	if i.Provider == "" {
		i.Provider = "Samsara"
	}

	return nil
}

func NewVehicleInspectionID() pulid.ID {
	return pulid.MustNew("vinsp_")
}

func (i *VehicleInspection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&i.ProviderDvirID,
			validation.Required.Error("Provider inspection id is required"),
		),
		validation.Field(&i.InspectionType,
			validation.Required.Error("Inspection type is required"),
			validation.Length(1, maxInspectionTypeLength).
				Error("Inspection type cannot be longer than 32 characters"),
		),
		// The safety status is what takes a tractor out of service, so it is
		// never left blank on a stored inspection.
		validation.Field(&i.SafetyStatus,
			validation.Required.Error("Safety status is required"),
			validation.Length(1, maxSafetyStatusLength).
				Error("Safety status cannot be longer than 16 characters"),
		),
		validation.Field(&i.StartedAt,
			validation.Required.Error("Started at is required"),
			validation.Min(int64(1)).Error("Started at must be a valid timestamp"),
		),
		validation.Field(&i.EndedAt,
			validation.Required.Error("Ended at is required"),
			validation.Min(i.StartedAt).Error("Ended at cannot precede the start"),
		),
		validation.Field(&i.OdometerMeters,
			validation.Min(int64(0)).Error("Odometer cannot be negative"),
		),
		validation.Field(&i.DefectCount,
			validation.Min(0).Error("Defect count cannot be negative"),
		),
		// More unresolved defects than defects would show a tractor as worse
		// than the inspection it came from says it is.
		validation.Field(&i.UnresolvedDefectCount,
			validation.Min(0).Error("Unresolved defect count cannot be negative"),
			validation.Max(i.DefectCount).
				Error("Unresolved defects cannot exceed the total defects"),
		),
	))
}

const (
	maxInspectionTypeLength = 32
	maxSafetyStatusLength   = 16
)
