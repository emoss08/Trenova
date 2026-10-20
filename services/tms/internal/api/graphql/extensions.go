package graphql

import "github.com/emoss08/trenova/internal/api/graphql/gqlext"

type (
	CostBudgetParams       = gqlext.CostBudgetParams
	CostBudgetExtension    = gqlext.CostBudgetExtension
	FieldGuardParams       = gqlext.FieldGuardParams
	FieldGuardExtension    = gqlext.FieldGuardExtension
	ObservabilityParams    = gqlext.ObservabilityParams
	ObservabilityExtension = gqlext.ObservabilityExtension
	ReadOnlyParams         = gqlext.ReadOnlyParams
	ReadOnlyExtension      = gqlext.ReadOnlyExtension
	ProtectedFieldMap      = gqlext.ProtectedFieldMap
)

const CostBudgetErrorCode = gqlext.CostBudgetErrorCode

var (
	NewCostBudgetExtension    = gqlext.NewCostBudgetExtension
	NewFieldGuardExtension    = gqlext.NewFieldGuardExtension
	NewObservabilityExtension = gqlext.NewObservabilityExtension
	NewReadOnlyExtension      = gqlext.NewReadOnlyExtension
	ProtectedFields           = gqlext.ProtectedFields
)
