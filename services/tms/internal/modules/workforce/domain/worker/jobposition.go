package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	_ validationframework.TenantedEntity = (*JobPosition)(nil)
)

// Job titles the roster is counted by, and the reporting lines between them
type JobPosition struct {
	keel.Table `keel:"job_positions,report=Parties"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:jpos_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Status              domaintypes.Status `json:"status"              keel:"status"`
	Code                string             `json:"code"                keel:"code"`
	Title               string             `json:"title"               keel:"title"`
	Description         string             `json:"description"         keel:"description,nullzero"`
	Department          JobDepartment      `json:"department"          keel:"department"`
	FLSAExempt          bool               `json:"flsaExempt"          keel:"flsa_exempt,label=Exempt from Overtime"`
	IsDrivingPosition   bool               `json:"isDrivingPosition"   keel:"is_driving_position,label=Driving Position"`
	ReportsToPositionID pulid.ID           `json:"reportsToPositionId" keel:"reports_to_position_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	ReportsTo *JobPosition `json:"reportsTo,omitempty" keel:"belongs:reports_to_position_id"`
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

func (p *JobPosition) BeforeCreate(_ context.Context) error {
	if p.Status == "" {
		p.Status = domaintypes.StatusActive
	}
	if p.Department == "" {
		p.Department = DepartmentOperations
	}

	return nil
}
