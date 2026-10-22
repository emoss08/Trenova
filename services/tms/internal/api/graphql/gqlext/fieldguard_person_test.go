package gqlext

import (
	"context"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// personGuard is the guard on the screen's path, for a person whose role
// cannot read the given kinds.
func personGuard(t *testing.T, denied ...permission.Resource) (*FieldGuardExtension, context.Context) {
	t.Helper()

	guard := newCeilingGuard(t)
	names := make([]string, 0, len(denied))
	for _, kind := range denied {
		names = append(names, kind.String())
	}
	guard.engine = &mocks.DenyResourcesPermissionEngine{Denied: names}

	return guard, operationContext(t, guard)
}

func withheldPaths(ctx context.Context) []string {
	holder, _ := ctx.Value(fieldVisibilityKey{}).(*fieldVisibility)
	if holder == nil {
		return nil
	}
	return holder.withheldPaths()
}

var tractorDriver = []fieldStep{
	{object: "Query", field: "tractors"},
	{object: "TractorConnection", field: "edges"},
	{object: "TractorEdge", field: "node"},
	{object: "Tractor", field: "primaryWorker"},
}

func steps(base []fieldStep, more ...fieldStep) []fieldStep {
	return append(append([]fieldStep{}, base...), more...)
}

func TestPersonRead_ALinkedRecordShowsItsReferenceView(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker)

	result, resolved := resolveNested(t, guard, ctx,
		steps(tractorDriver, fieldStep{object: "Worker", field: "wholeName", nonNull: true})...)
	assert.Equal(t, "stored value", result, "a driver's name is in the worker reference view")
	assert.True(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		steps(tractorDriver, fieldStep{object: "Worker", field: "email"})...)
	assert.Nil(t, result, "a driver's email needs worker read")
	assert.False(t, resolved, "a withheld field is never resolved")

	result, resolved = resolveNested(t, guard, ctx,
		steps(tractorDriver, fieldStep{object: "Worker", field: "id", nonNull: true})...)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	assert.Equal(t, []string{"tractors.edges.node.primaryWorker.email"}, withheldPaths(ctx))
}

func TestPersonRead_TheKindsReadShowsTheWholeLinkedRecord(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t)

	result, resolved := resolveNested(t, guard, ctx,
		steps(tractorDriver, fieldStep{object: "Worker", field: "email"})...)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Empty(t, withheldPaths(ctx))
}

func TestPersonRead_RefusesANonNullFieldOutsideTheView(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker)

	resolved := false
	_, err := guard.InterceptField(
		nestedFieldContext(ctx, steps(tractorDriver,
			fieldStep{object: "Worker", field: "addressLine1", nonNull: true})...),
		func(context.Context) (any, error) {
			resolved = true
			return "stored value", nil
		})
	require.Error(t, err, "an operation must declare @requiresRead for a non-view field")
	assert.Contains(t, err.Error(), "@requiresRead")
	assert.False(t, resolved)
}

func TestPersonRead_TheRecordARootReturnsIsNotGatedByKind(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)
	assert.Equal(t, "stored value", result, "the root resolver authorized the record")
	assert.True(t, resolved)
}

func TestPersonRead_ASelfScopedRootIsNotGatedByKind(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker, permission.ResourceDriverSettlement)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "mySettlements"},
		fieldStep{object: "DriverSettlement", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)
	assert.Equal(t, "stored value", result, "everything under a self-scoped root is the caller's own")
	assert.True(t, resolved)
}

func TestPersonRead_ARootThatAuthorizesAKindReturnsItsOwnRecords(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "workerOverview", authorizes: "worker"},
		fieldStep{object: "WorkerOverview", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "workerOverview"},
		fieldStep{object: "WorkerOverview", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)
	assert.Nil(t, result, "without the declaration the worker is a linked record")
	assert.False(t, resolved)
}

func TestPersonRead_APartOfALinkedRecordFollowsThatRecordsRead(t *testing.T) {
	t.Parallel()

	moveStatus := []fieldStep{
		{object: "Query", field: "invoice"},
		{object: "Invoice", field: "shipment"},
		{object: "Shipment", field: "moves"},
		{object: "ShipmentMove", field: "status"},
	}

	guard, ctx := personGuard(t, permission.ResourceShipment)
	result, resolved := resolveNested(t, guard, ctx, moveStatus...)
	assert.Nil(t, result, "a move is part of its shipment, which the reader cannot read")
	assert.False(t, resolved)

	guard, ctx = personGuard(t, permission.ResourceShipmentMove)
	result, resolved = resolveNested(t, guard, ctx, moveStatus...)
	assert.Equal(t, "stored value", result, "the shipment's read reaches its moves")
	assert.True(t, resolved)
}

func TestPersonRead_ALeaveCaseAWorkerCarriesNeedsLeaveRead(t *testing.T) {
	t.Parallel()

	leaveField := func(field string) []fieldStep {
		return []fieldStep{
			{object: "Query", field: "worker"},
			{object: "Worker", field: "leaveCases"},
			{object: "WorkerLeaveCase", field: field},
		}
	}
	credentialExpiry := []fieldStep{
		{object: "Query", field: "worker"},
		{object: "Worker", field: "credentials"},
		{object: "WorkerCredential", field: "expiresAt"},
	}
	leaveGuard := func(denied ...permission.Resource) (*FieldGuardExtension, context.Context) {
		guard, ctx := personGuard(t, denied...)
		guard.resources["WorkerLeaveCase"] = permission.ResourceWorkerLeave
		return guard, ctx
	}

	guard, ctx := leaveGuard(permission.ResourceWorkerLeave, permission.ResourceWorkerCredential)
	for _, field := range []string{"reason", "leaveType", "notes", "certificationStatus"} {
		result, resolved := resolveNested(t, guard, ctx, leaveField(field)...)
		assert.Nil(t, result, "a worker reader without worker_leave read sees no %s", field)
		assert.False(t, resolved, field)
	}
	result, resolved := resolveNested(t, guard, ctx, leaveField("id")...)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	result, resolved = resolveNested(t, guard, ctx, credentialExpiry...)
	assert.Equal(t, "stored value", result, "a credential is still part of its worker")
	assert.True(t, resolved)
	assert.Contains(t, withheldPaths(ctx), "worker.leaveCases.reason")

	guard, ctx = leaveGuard(permission.ResourceWorker)
	result, resolved = resolveNested(t, guard, ctx, leaveField("reason")...)
	assert.Equal(t, "stored value", result, "worker_leave read shows the case")
	assert.True(t, resolved)
}

func TestPersonRead_AnObjectInTheViewShowsWithIt(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceLocation)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "distanceOverrides"},
		fieldStep{object: "DistanceOverride", field: "originLocation"},
		fieldStep{object: "Location", field: "state"},
		fieldStep{object: "UsState", field: "abbreviation"},
	)
	assert.Equal(t, "stored value", result, "a location's state is in its reference view")
	assert.True(t, resolved)
}

func TestPersonRead_AProjectionIsGatedAsItsKind(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceAccessorialCharge)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "shipment"},
		fieldStep{object: "Shipment", field: "additionalCharges"},
		fieldStep{object: "ShipmentAdditionalCharge", field: "accessorialCharge"},
		fieldStep{object: "ShipmentAccessorialCharge", field: "amount"},
	)
	assert.Nil(t, result, "an accessorial charge's price needs its own read")
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "shipment"},
		fieldStep{object: "Shipment", field: "additionalCharges"},
		fieldStep{object: "ShipmentAdditionalCharge", field: "accessorialCharge"},
		fieldStep{object: "ShipmentAccessorialCharge", field: "code"},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
}

func TestPersonRead_TheResponseNamesWhatWasWithheld(t *testing.T) {
	t.Parallel()

	guard, ctx := personGuard(t, permission.ResourceWorker)
	resolveNested(t, guard, ctx, steps(tractorDriver, fieldStep{object: "Worker", field: "email"})...)

	resp := guard.InterceptResponse(ctx, func(context.Context) *gqlgen.Response {
		return &gqlgen.Response{}
	})
	require.NotNil(t, resp)
	assert.Equal(t, []string{"tractors.edges.node.primaryWorker.email"},
		resp.Extensions[withheldExtension])

	guard, ctx = personGuard(t)
	resp = guard.InterceptResponse(ctx, func(context.Context) *gqlgen.Response {
		return &gqlgen.Response{}
	})
	assert.NotContains(t, resp.Extensions, withheldExtension, "nothing withheld, nothing named")
}

func TestPersonRead_AnInjuryCaseAWorkerCarriesNeedsInjuryRead(t *testing.T) {
	t.Parallel()

	injuryField := func(field string) []fieldStep {
		return []fieldStep{
			{object: "Query", field: "worker"},
			{object: "Worker", field: "injuries"},
			{object: "WorkerInjury", field: field},
		}
	}

	guard, ctx := personGuard(t, permission.ResourceWorkerInjury)
	for _, field := range []string{"bodyPart", "harmfulAgent"} {
		result, resolved := resolveNested(t, guard, ctx, injuryField(field)...)
		assert.Nil(t, result, "a worker reader without worker_injury read sees no %s", field)
		assert.False(t, resolved, field)
	}
	for _, field := range []string{"id", "caseNumber", "status"} {
		result, resolved := resolveNested(t, guard, ctx, injuryField(field)...)
		assert.Equal(t, "stored value", result, "%s is in the injury's reference view", field)
		assert.True(t, resolved, field)
	}
	assert.Contains(t, withheldPaths(ctx), "worker.injuries.bodyPart")

	guard, ctx = personGuard(t, permission.ResourceWorker)
	result, resolved := resolveNested(t, guard, ctx, injuryField("bodyPart")...)
	assert.Equal(t, "stored value", result, "worker_injury read shows the case")
	assert.True(t, resolved)
}
