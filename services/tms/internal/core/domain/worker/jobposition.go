package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var ErrInvalidJobDepartment = errors.New("invalid job department")

// JobDepartment is the part of the business a position sits in. Headcount is
// read by department at least as often as by terminal, which is why it is an
// enum on the position rather than free text somebody spells three ways.
type JobDepartment string

const (
	DepartmentOperations     = JobDepartment("Operations")
	DepartmentSafety         = JobDepartment("Safety")
	DepartmentMaintenance    = JobDepartment("Maintenance")
	DepartmentBilling        = JobDepartment("Billing")
	DepartmentAdministration = JobDepartment("Administration")
	DepartmentSales          = JobDepartment("Sales")
	DepartmentHumanResources = JobDepartment("HumanResources")
	DepartmentExecutive      = JobDepartment("Executive")
	DepartmentOther          = JobDepartment("Other")
)

func (d JobDepartment) String() string { return string(d) }

func (d JobDepartment) IsValid() bool {
	switch d {
	case DepartmentOperations, DepartmentSafety, DepartmentMaintenance, DepartmentBilling,
		DepartmentAdministration, DepartmentSales, DepartmentHumanResources,
		DepartmentExecutive, DepartmentOther:
		return true
	default:
		return false
	}
}

func AllJobDepartments() []JobDepartment {
	return []JobDepartment{
		DepartmentOperations,
		DepartmentSafety,
		DepartmentMaintenance,
		DepartmentBilling,
		DepartmentAdministration,
		DepartmentSales,
		DepartmentHumanResources,
		DepartmentExecutive,
		DepartmentOther,
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*JobPosition)(nil)
	_ validationframework.TenantedEntity = (*JobPosition)(nil)
)

// JobPosition is what somebody does, as distinct from where they do it. A
// fleet code is a terminal; this is the title the headcount is counted by and
// the org chart is drawn from.
type JobPosition struct {
	bun.BaseModel `bun:"table:job_positions,alias:jpos" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Status      domaintypes.Status `json:"status"      bun:"status,type:status_enum,notnull,default:'Active'"`
	Code        string             `json:"code"        bun:"code,type:VARCHAR(20),notnull"`
	Title       string             `json:"title"       bun:"title,type:VARCHAR(100),notnull"`
	Description string             `json:"description" bun:"description,type:TEXT,nullzero"`
	Department  JobDepartment      `json:"department"  bun:"department,type:job_department_enum,notnull,default:'Operations'"`
	// FLSAExempt says the position is exempt from overtime. It lives on the
	// position because that is where the duties test is applied — two people
	// doing the same job are exempt or not together.
	FLSAExempt bool `json:"flsaExempt" bun:"flsa_exempt,type:BOOLEAN,notnull"`
	// IsDrivingPosition separates the roster that needs a CDL from the one that
	// does not, which is the line most compliance rules are drawn along.
	IsDrivingPosition bool `json:"isDrivingPosition" bun:"is_driving_position,type:BOOLEAN,notnull"`
	// ReportsToPositionID is the shape of the org chart. A person's own manager
	// is on the worker, because two people in the same position can report to
	// different managers.
	ReportsToPositionID pulid.ID `json:"reportsToPositionId" bun:"reports_to_position_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	ReportsTo *JobPosition `json:"reportsTo,omitempty" bun:"rel:belongs-to,join:reports_to_position_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *JobPosition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 20).Error("Code cannot exceed 20 characters"),
		),
		validation.Field(&p.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, 100).Error("Title cannot exceed 100 characters"),
		),
		validation.Field(&p.Department,
			validation.Required.Error("Department is required"),
			domainvalidation.ValidEnum[JobDepartment]("Department is not valid"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
	))

	// A position reporting to itself is a cycle of one, and the shortest way to
	// hang an org chart walk.
	if !p.ReportsToPositionID.IsNil() && p.ReportsToPositionID == p.ID {
		multiErr.Add(
			"reportsToPositionId",
			errortypes.ErrInvalid,
			"A position cannot report to itself",
		)
	}
}

// Normalise trims the free text and settles the code on a single spelling, so
// the unique index is not the only thing deciding two codes are the same.
func (p *JobPosition) Normalise() {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
}

func (p *JobPosition) GetID() pulid.ID { return p.ID }

func (p *JobPosition) GetCreatedAt() int64 { return p.CreatedAt }

func (p *JobPosition) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *JobPosition) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *JobPosition) GetTableName() string { return "job_positions" }

func (p *JobPosition) GetResourceType() string { return "job_position" }

func (p *JobPosition) GetResourceID() string { return p.ID.String() }

func (p *JobPosition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("jpos_")
		}
		if p.Status == "" {
			p.Status = domaintypes.StatusActive
		}
		if p.Department == "" {
			p.Department = DepartmentOperations
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
