package validationframework

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type FieldCheck struct {
	Column        string
	Value         any
	CaseSensitive bool
}

type Condition struct {
	Column string
	Value  any
}

type UniqueCheck struct {
	Fields []FieldCheck
}

type ReferenceCheck struct {
	Table      string
	Column     string
	ID         pulid.ID
	Conditions []Condition
}

type IntegrityRequest struct {
	Table          string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ExcludeID      pulid.ID
	Unique         []UniqueCheck
	References     []ReferenceCheck
}

func (r *IntegrityRequest) Empty() bool {
	return len(r.Unique) == 0 && len(r.References) == 0
}

type IntegrityResult struct {
	Duplicate []bool
	Exists    []bool
}

type IntegrityChecker interface {
	CheckIntegrity(ctx context.Context, req *IntegrityRequest) (*IntegrityResult, error)
}
