package recurringshipment

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type RecurringShipmentRun struct {
	keel.Table `keel:"recurring_shipment_runs"`

	ID                   pulid.ID   `json:"id"                   keel:"id,pk,autoid:rsr_"`
	BusinessUnitID       pulid.ID   `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID   `json:"organizationId"       keel:"organization_id,pk,tenant"`
	RecurringShipmentID  pulid.ID   `json:"recurringShipmentId"  keel:"recurring_shipment_id"`
	GeneratedShipmentID  pulid.ID   `json:"generatedShipmentId"  keel:"generated_shipment_id,nullzero"`
	TriggeredByID        pulid.ID   `json:"triggeredById"        keel:"triggered_by_id,nullzero"`
	Status               RunStatus  `json:"status"               keel:"status"`
	Trigger              RunTrigger `json:"trigger"              keel:"trigger"`
	OccurrenceAt         int64      `json:"occurrenceAt"         keel:"occurrence_at"`
	OriginalOccurrenceAt *int64     `json:"originalOccurrenceAt" keel:"original_occurrence_at"`
	Detail               string     `json:"detail"               keel:"detail,nullzero"`
	CreatedAt            int64      `json:"createdAt"            keel:"created_at,autocreate"`

	BusinessUnit      *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization      *tenant.Organization `json:"organization,omitempty"`
	RecurringShipment *RecurringShipment   `json:"recurringShipment,omitempty" keel:"belongs:recurring_shipment_id"`
	GeneratedShipment *shipment.Shipment   `json:"generatedShipment,omitempty" keel:"belongs:generated_shipment_id"`
	TriggeredBy       *tenant.User         `json:"triggeredBy,omitempty"       keel:"belongs:triggered_by_id"`
}

func (rr *RecurringShipmentRun) GetID() pulid.ID {
	return rr.ID
}

func (rr *RecurringShipmentRun) GetTableName() string {
	return "recurring_shipment_runs"
}

func (rr *RecurringShipmentRun) GetOrganizationID() pulid.ID {
	return rr.OrganizationID
}

func (rr *RecurringShipmentRun) GetBusinessUnitID() pulid.ID {
	return rr.BusinessUnitID
}

func (r *RecurringShipmentRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.RecurringShipmentID,
			validation.Required.Error("Recurring shipment is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RunStatus]("Status is invalid"),
		),
		validation.Field(&r.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[RunTrigger]("Trigger is invalid"),
		),
		// The occurrence is which scheduled slot this run stands for, and it is
		// what stops the same slot being generated twice.
		validation.Field(&r.OccurrenceAt,
			validation.Required.Error("Occurrence is required"),
			validation.Min(int64(1)).Error("Occurrence must be a valid timestamp"),
		),
		validation.Field(&r.GeneratedShipmentID,
			validation.When(
				r.Status == RunStatusGenerated,
				validation.Required.Error("A generated run must name the shipment it created"),
			),
		),
		validation.Field(&r.Detail,
			validation.When(
				r.Status == RunStatusFailed || r.Status == RunStatusSkipped,
				validation.Required.Error("A failed or skipped run must record why"),
			),
		),
	))
}
