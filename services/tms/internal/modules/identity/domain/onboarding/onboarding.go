package onboarding

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Onboarding)(nil)
)

type Onboarding struct {
	keel.Table `keel:"organization_onboarding"`

	ID               pulid.ID             `json:"id" keel:"id,pk,autoid:oonb_"`
	BusinessUnitID   pulid.ID             `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID             `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status           Status               `json:"status" keel:"status"`
	OperationType    tenant.OperationType `json:"operationType,omitempty" keel:"operation_type,nullzero"`
	SampleDataLoaded bool                 `json:"sampleDataLoaded" keel:"sample_data_loaded"`
	CompletedAt      *int64               `json:"completedAt" keel:"completed_at"`
	CompletedByID    pulid.ID             `json:"completedById,omitempty" keel:"completed_by_id,nullzero"`
	Version          int64                `json:"version" keel:"version,version"`
	CreatedAt        int64                `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64                `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type CompleteParams struct {
	UserID           pulid.ID
	OperationType    tenant.OperationType
	SampleDataLoaded bool
	CompletedAt      int64
}

func NewPending(organizationID, businessUnitID pulid.ID) *Onboarding {
	return &Onboarding{
		OrganizationID: organizationID,
		BusinessUnitID: businessUnitID,
		Status:         StatusPending,
	}
}

func (o *Onboarding) IsCompleted() bool {
	return o.Status == StatusCompleted
}

func (o *Onboarding) Complete(params CompleteParams) {
	completedAt := params.CompletedAt
	if completedAt == 0 {
		completedAt = timeutils.NowUnix()
	}

	o.Status = StatusCompleted
	o.OperationType = params.OperationType
	o.SampleDataLoaded = params.SampleDataLoaded
	o.CompletedAt = &completedAt
	o.CompletedByID = params.UserID
}

func (o *Onboarding) Validate(multiErr *errortypes.MultiError) {
	completed := o.Status == StatusCompleted

	multiErr.AddOzzoError(validation.ValidateStruct(
		o,
		validation.Field(&o.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&o.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(
			&o.Status,
			validation.Required.Error("Status is required"),
			validation.By(func(any) error {
				if !o.Status.IsValid() {
					return validation.NewError("validation_invalid", "Status is invalid")
				}
				return nil
			}),
		),
		validation.Field(
			&o.OperationType,
			validation.When(
				completed,
				validation.Required.Error("Operation type is required"),
			),
			validation.By(func(any) error {
				if o.OperationType != "" && !o.OperationType.IsValid() {
					return validation.NewError(
						"validation_invalid",
						"Operation type must be asset, brokerage or both",
					)
				}
				return nil
			}),
		),
		validation.Field(
			&o.CompletedAt,
			validation.When(completed, validation.Required.Error("Completion time is required")),
		),
		validation.Field(
			&o.CompletedByID,
			validation.When(completed, validation.Required.Error("Completed by is required")),
		),
	))
}

func (o *Onboarding) GetID() pulid.ID {
	return o.ID
}

func (o *Onboarding) GetOrganizationID() pulid.ID {
	return o.OrganizationID
}

func (o *Onboarding) GetBusinessUnitID() pulid.ID {
	return o.BusinessUnitID
}

func (o *Onboarding) GetTableName() string {
	return "organization_onboarding"
}
