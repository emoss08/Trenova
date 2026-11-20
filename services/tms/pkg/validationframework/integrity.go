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

// ConditionOp is how a Condition compares its column. The zero value is
// equality with Value.
type ConditionOp string

const (
	ConditionEq ConditionOp = ""
	// ConditionIsNull holds when the column is null; Value is ignored.
	ConditionIsNull ConditionOp = "is_null"
	// ConditionNullOrNotAfterNow holds when the column, a Unix time in
	// seconds, is null or not later than the moment the check runs: an
	// effective date that has already begun. Value is ignored.
	ConditionNullOrNotAfterNow ConditionOp = "null_or_not_after_now"
)

type Condition struct {
	Column string
	Op     ConditionOp
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
