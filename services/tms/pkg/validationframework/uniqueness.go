package validationframework

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type UniquenessChecker interface {
	CheckUniqueness(ctx context.Context, req *UniquenessRequest) (bool, error)
}

type UniquenessRequest struct {
	TableName      string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ExcludeID      pulid.ID
	ScopeFields    []FieldCheck
	Fields         []FieldCheck
}

type FieldCheck struct {
	Column        string
	Value         any
	CaseSensitive bool
}
