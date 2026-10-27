package worker

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ domaintypes.PostgresSearchable     = (*WorkerPTO)(nil)
	_ pagination.CursorEntity            = (*WorkerPTO)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTO)(nil)
)

// Time-off requests and the days booked against them
type WorkerPTO struct {
	keel.Table         `keel:"worker_pto,report=Parties,label=Time Off Request"`
	ID                 pulid.ID            `keel:"id,pk,autoid:wrkpto_"                             json:"id"`
	WorkerID           pulid.ID            `keel:"worker_id,pk"                                     json:"workerId"`
	BusinessUnitID     pulid.ID            `keel:"business_unit_id,pk,tenant"                       json:"businessUnitId"`
	OrganizationID     pulid.ID            `keel:"organization_id,pk,tenant"                        json:"organizationId"`
	ApproverID         pulid.ID            `keel:"approver_id,nullzero"                             json:"approverId"`
	RejectorID         pulid.ID            `keel:"rejector_id,nullzero"                             json:"rejectorId"`
	CancelledByID      pulid.ID            `keel:"cancelled_by_id,nullzero"                         json:"cancelledById"`
	Status             PTOStatus           `keel:"status"                                           json:"status"`
	Type               PTOType             `keel:"type"                                             json:"type"`
	StartDate          int64               `keel:"start_date,label=Starts"                          json:"startDate"`
	EndDate            int64               `keel:"end_date,label=Ends"                              json:"endDate"`
	Reason             string              `keel:"reason"                                           json:"reason"`
	RejectionReason    string              `keel:"rejection_reason,nullzero"                        json:"rejectionReason"`
	CancellationReason string              `keel:"cancellation_reason,nullzero"                     json:"cancellationReason"`
	Days               decimal.Decimal     `keel:"days"                                             json:"days"`
	BalanceAfterDays   decimal.NullDecimal `keel:"balance_after_days"                               json:"balanceAfterDays"`
	AutoApproved       bool                `keel:"auto_approved"                                    json:"autoApproved"`
	SearchVector       string              `keel:"search_vector,tsvector,readonly,noselect"         json:"-"`
	Version            int64               `keel:"version,version"                                  json:"version"`
	CreatedAt          int64               `keel:"created_at,autocreate"                            json:"createdAt"`
	UpdatedAt          int64               `keel:"updated_at,autoupdate"                            json:"updatedAt"`

	Worker      *Worker      `json:"worker,omitempty"      keel:"belongs:worker_id"`
	Approver    *tenant.User `json:"approver,omitempty"    keel:"belongs:approver_id,noreport"`
	Rejector    *tenant.User `json:"rejector,omitempty"    keel:"belongs:rejector_id,noreport"`
	CancelledBy *tenant.User `json:"cancelledBy,omitempty" keel:"belongs:cancelled_by_id,noreport"`
}

func (wpto *WorkerPTO) BeforeCreate(_ context.Context) error {
	if wpto.Status == "" {
		wpto.Status = PTOStatusRequested
	}
	if wpto.Type == "" {
		wpto.Type = PTOTypeVacation
	}

	return nil
}

func (wpto *WorkerPTO) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(wpto,
		validation.Field(&wpto.WorkerID,
			validation.Required.Error("Worker is required"),
			validation.By(func(value any) error {
				id, ok := value.(pulid.ID)
				if !ok {
					return errors.New("invalid worker ID type")
				}
				if id.IsNil() {
					return errors.New("Worker is required")
				}
				return nil
			}),
		),
		validation.Field(
			&wpto.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PTOStatus](
				"status must be one of: Requested, Approved, Rejected, Cancelled",
			),
		),
		validation.Field(
			&wpto.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[PTOType](
				"type must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			),
		),
		validation.Field(&wpto.StartDate,
			validation.Required.Error("Start date is required"),
			validation.Min(int64(1)).Error("Start date must be a positive value"),
		),
		validation.Field(&wpto.EndDate,
			validation.Required.Error("End date is required"),
			validation.Min(int64(1)).Error("End date must be a positive value"),
		),
		validation.Field(&wpto.Reason,
			validation.Required.Error("Reason is required"),
			validation.Length(1, 255).Error("Reason must be between 1 and 255 characters"),
		),
	))

	if wpto.EndDate < wpto.StartDate {
		multiErr.Add("endDate", errortypes.ErrInvalid, "End date cannot be before start date")
	}
}

func (wpto *WorkerPTO) IsOpen() bool {
	return wpto.Status == PTOStatusRequested || wpto.Status == PTOStatusApproved
}

func (wpto *WorkerPTO) GetTableName() string {
	return "worker_pto"
}

func (wpto *WorkerPTO) GetID() pulid.ID {
	return wpto.ID
}

func (wpto *WorkerPTO) GetCreatedAt() int64 {
	return wpto.CreatedAt
}

func (wpto *WorkerPTO) GetResourceType() string {
	return "worker_pto"
}

func (wpto *WorkerPTO) GetResourceID() string {
	return wpto.ID.String()
}

func (wpto *WorkerPTO) GetBusinessUnitID() pulid.ID {
	return wpto.BusinessUnitID
}

func (wpto *WorkerPTO) GetOrganizationID() pulid.ID {
	return wpto.OrganizationID
}

func (wpto *WorkerPTO) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wpto",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "type", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{
				Name:   "start_date",
				Type:   domaintypes.FieldTypeDate,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "end_date", Type: domaintypes.FieldTypeDate, Weight: domaintypes.SearchWeightB},
			{Name: "reason", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (WorkerPTO) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_pto_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}
