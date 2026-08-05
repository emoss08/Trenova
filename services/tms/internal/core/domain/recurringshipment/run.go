package recurringshipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RecurringShipmentRun)(nil)

type RecurringShipmentRun struct {
	bun.BaseModel `json:"-" bun:"table:recurring_shipment_runs,alias:rsr"`

	ID                   pulid.ID   `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID   `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID   `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RecurringShipmentID  pulid.ID   `json:"recurringShipmentId"  bun:"recurring_shipment_id,type:VARCHAR(100),notnull"`
	GeneratedShipmentID  pulid.ID   `json:"generatedShipmentId"  bun:"generated_shipment_id,type:VARCHAR(100),nullzero"`
	TriggeredByID        pulid.ID   `json:"triggeredById"        bun:"triggered_by_id,type:VARCHAR(100),nullzero"`
	Status               RunStatus  `json:"status"               bun:"status,type:recurring_shipment_run_status_enum,notnull"`
	Trigger              RunTrigger `json:"trigger"              bun:"trigger,type:recurring_shipment_run_trigger_enum,notnull"`
	OccurrenceAt         int64      `json:"occurrenceAt"         bun:"occurrence_at,type:BIGINT,notnull"`
	OriginalOccurrenceAt *int64     `json:"originalOccurrenceAt" bun:"original_occurrence_at,type:BIGINT,nullzero"`
	Detail               string     `json:"detail"               bun:"detail,type:TEXT,nullzero"`
	CreatedAt            int64      `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	RecurringShipment *RecurringShipment   `json:"recurringShipment,omitempty" bun:"rel:belongs-to,join:recurring_shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	GeneratedShipment *shipment.Shipment   `json:"generatedShipment,omitempty" bun:"rel:belongs-to,join:generated_shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	TriggeredBy       *tenant.User         `json:"triggeredBy,omitempty"       bun:"rel:belongs-to,join:triggered_by_id=id"`
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

func (rr *RecurringShipmentRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if rr.ID.IsNil() {
			rr.ID = pulid.MustNew("rsr_")
		}

		rr.CreatedAt = timeutils.NowUnix()
	}

	return nil
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
