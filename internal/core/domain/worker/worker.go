package worker

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Worker)(nil)
	_ domain.Validatable        = (*Worker)(nil)
	_ infra.PostgresSearchable  = (*Worker)(nil)
)

type Worker struct {
	bun.BaseModel `bun:"table:workers,alias:wrk" json:"-"`

	// Primary identifiers
	ID             pulid.ID  `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	StateID        pulid.ID  `json:"stateId" bun:"state_id,type:VARCHAR(100),notnull"`
	FleetCodeID    *pulid.ID `json:"fleetCodeId" bun:"fleet_code_id,type:VARCHAR(100)"`

	// Core Fields
	Status            domain.Status `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Type              WorkerType    `json:"type" bun:"type,type:worker_type_enum,notnull,default:'Employee'"`
	ProfilePicURL     string        `json:"profilePicUrl" bun:"profile_pic_url,type:VARCHAR(255)"`
	FirstName         string        `json:"firstName" bun:"first_name,type:VARCHAR(100),notnull"`
	LastName          string        `json:"lastName" bun:"last_name,type:VARCHAR(100),notnull"`
	WholeName         string        `json:"wholeName" bun:"whole_name,type:VARCHAR(201),scanonly"`
	AddressLine1      string        `json:"addressLine1" bun:"address_line1,type:VARCHAR(150),notnull"`
	AddressLine2      string        `json:"addressLine2" bun:"address_line2,type:VARCHAR(150)"`
	City              string        `json:"city" bun:"city,type:VARCHAR(100),notnull"`
	PostalCode        string        `json:"postalCode" bun:"postal_code,type:us_postal_code,notnull"`
	Gender            domain.Gender `json:"gender" bun:"gender,type:gender_enum,notnull"`
	CanBeAssigned     bool          `json:"canBeAssigned" bun:"can_be_assigned,type:BOOLEAN,notnull,default:false"`
	AssignmentBlocked string        `json:"assignmentBlocked,omitempty" bun:"assignment_blocked,type:VARCHAR(255)"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	State        *usstate.UsState           `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id"`
	Profile      *WorkerProfile             `json:"profile,omitempty" bun:"rel:has-one,join:id=worker_id"`
	PTO          []*WorkerPTO               `json:"pto,omitempty" bun:"rel:has-many,join:id=worker_id"`
}

// Validation
func (w *Worker) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, w,
		// Type is required and must be either Employee or Contractor
		validation.Field(&w.Type,
			validation.Required.Error("Type is required"),
			validation.In(WorkerTypeEmployee, WorkerTypeContractor).Error("Type must be either Employee or Contractor"),
		),

		// First Name cannot contain spaces and numbers
		validation.Field(&w.FirstName,
			validation.Required.Error("First Name is required"),
			is.Alpha.Error("First Name cannot contain spaces and numbers"),
		),

		// Last Name cannot contain spaces and numbers
		validation.Field(&w.LastName,
			validation.Required.Error("Last Name is required"),
			is.Alpha.Error("Last Name cannot contain spaces and numbers"),
		),

		// Gender is required
		validation.Field(&w.Gender,
			validation.Required.Error("Gender is required"),
			validation.In(
				domain.GenderMale,
				domain.GenderFemale,
			).Error("Gender must be either Male or Female"),
		),

		// Address Information is required
		validation.Field(&w.AddressLine1,
			validation.Required.Error("Address Line 1 is required"),
		),
		validation.Field(&w.City,
			validation.Required.Error("City is required"),
		),
		validation.Field(&w.PostalCode,
			validation.Required.Error("Postal Code is required"),
			validation.By(domain.ValidatePostalCode),
		),
		validation.Field(&w.StateID,
			validation.Required.Error("State is required"),
		),

		// Status is required
		validation.Field(&w.Status,
			validation.Required.Error("Status is required"),
			validation.In(domain.StatusActive, domain.StatusInactive).Error("Status must be either Active or Inactive"),
		),

		// Ensure their is a profile for the worker
		validation.Field(&w.Profile, validation.Required.Error("Worker profile must be provided.")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (w *Worker) GetTableName() string {
	return "workers"
}

// Search Configuration
func (w *Worker) GetID() string {
	return w.ID.String()
}

func (w *Worker) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "wrk",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "first_name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:   "last_name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:       "status",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}

// misc
func (w *Worker) FullName() string {
	return fmt.Sprintf("%s %s", w.FirstName, w.LastName)
}

func (w *Worker) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID == "" {
			w.ID = pulid.MustNew("wrk_")
		}

		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}
