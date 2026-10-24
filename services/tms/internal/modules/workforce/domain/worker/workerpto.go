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
	keel.Table         `                                           keel:"worker_pto,report=Parties,label=Time Off Request"`
	ID                 pulid.ID            `json:"id" keel:"id,pk,autoid:wrkpto_"`
	WorkerID           pulid.ID            `json:"workerId" keel:"worker_id,pk"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	ApproverID         pulid.ID            `json:"approverId" keel:"approver_id,nullzero"`
	RejectorID         pulid.ID            `json:"rejectorId" keel:"rejector_id,nullzero"`
	CancelledByID      pulid.ID            `json:"cancelledById" keel:"cancelled_by_id,nullzero"`
	Status             PTOStatus           `json:"status" keel:"status"`
	Type               PTOType             `json:"type" keel:"type"`
	StartDate          int64               `json:"startDate" keel:"start_date,label=Starts"`
	EndDate            int64               `json:"endDate" keel:"end_date,label=Ends"`
	Reason             string              `json:"reason" keel:"reason"`
	RejectionReason    string              `json:"rejectionReason" keel:"rejection_reason,nullzero"`
	CancellationReason string              `json:"cancellationReason" keel:"cancellation_reason,nullzero"`
	Days               decimal.Decimal     `json:"days" keel:"days"`
	BalanceAfterDays   decimal.NullDecimal `json:"balanceAfterDays" keel:"balance_after_days"`
	AutoApproved       bool                `json:"autoApproved" keel:"auto_approved"`
	SearchVector       string              `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Version            int64               `json:"version" keel:"version,version"`
	CreatedAt          int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64               `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker      *Worker      `json:"worker,omitempty" keel:"belongs:worker_id"`
	Approver    *tenant.User `json:"approver,omitempty" keel:"belongs:approver_id,noreport"`
	Rejector    *tenant.User `json:"rejector,omitempty" keel:"belongs:rejector_id,noreport"`
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
