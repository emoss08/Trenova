package onboarding

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Onboarding)(nil)
	_ validationframework.TenantedEntity = (*Onboarding)(nil)
)

type Onboarding struct {
	bun.BaseModel `bun:"table:organization_onboarding,alias:oonb" json:"-"`

	ID               pulid.ID      `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID      `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID      `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Status           Status        `json:"status"                  bun:"status,type:VARCHAR(20),notnull"`
	OperationType    OperationType `json:"operationType,omitempty" bun:"operation_type,type:VARCHAR(20),nullzero"`
	SampleDataLoaded bool          `json:"sampleDataLoaded"        bun:"sample_data_loaded,type:BOOLEAN,notnull"`
	CompletedAt      *int64        `json:"completedAt"             bun:"completed_at,type:BIGINT,nullzero"`
	CompletedByID    pulid.ID      `json:"completedById,omitempty" bun:"completed_by_id,type:VARCHAR(100),nullzero"`
	Version          int64         `json:"version"                 bun:"version,type:BIGINT,notnull"`
	CreatedAt        int64         `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64         `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type CompleteParams struct {
	UserID           pulid.ID
	OperationType    OperationType
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

func (o *Onboarding) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("oonb_")
		}
		if o.CreatedAt == 0 {
			o.CreatedAt = now
		}
		o.UpdatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
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
