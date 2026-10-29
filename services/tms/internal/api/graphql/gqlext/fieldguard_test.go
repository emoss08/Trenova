package gqlext

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func loadProtectedFields(t *testing.T) ProtectedFieldMap {
	t.Helper()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	fields, err := ProtectedFields(schema)
	require.NoError(t, err)
	return fields
}

func TestProtectedFieldsAreTheDeclaredSet(t *testing.T) {
	t.Parallel()

	margins := func(fields ...string) map[string]protectedfield.Group {
		out := make(map[string]protectedfield.Group, len(fields))
		for _, field := range fields {
			out[field] = protectedfield.Margins
		}
		return out
	}
	detentionPay := margins("driverPayAmount", "netMargin")

	assert.Equal(t, ProtectedFieldMap{
		"WorkerProfile": {
			"dob":            protectedfield.Personal,
			"licenseNumber":  protectedfield.Personal,
			"twicCardNumber": protectedfield.Personal,
			"lastDrugTest":   protectedfield.Personal,
		},
		"WorkerCredential": {
			"number": protectedfield.Personal,
		},
		"RateAgreement":       margins("marginFloorPercent", "maxPayPercentOfSell"),
		"CostingControl":      margins("targetMarginPercent"),
		"ResolvedCostProfile": margins("targetMarginPercent"),
		"FleetCostSummary": margins(
			"avgMarginPercent",
			"unprofitableCount",
			"totalRevenue",
		),
		"Query":                          margins("shipmentProfitability"),
		"Shipment":                       margins("profitabilityEstimate"),
		"ShipmentProfitabilityAnalytics": margins("avgMarginPct", "unprofitableCount"),
		"TenderOffer":                    margins("counterMarginPercent"),
		"CapacityMatch":                  margins("quote", "marginPercent"),
		"DetentionOccurrence":            detentionPay,
		"FacilityDetentionStat":          detentionPay,
		"CustomerDetentionStat":          detentionPay,
		"DetentionPreviewResult":         detentionPay,
		"DetentionBacktestBucket":        detentionPay,
		"DetentionBacktestResult": margins(
			"proposedDriverPay", "proposedNetMargin", "negativeMarginStops",
		),
	}, loadProtectedFields(t))
}

func TestEveryProtectedTypeSaysWhatStandsBehindIt(t *testing.T) {
	t.Parallel()

	declared := protectedTypes()
	for object := range loadProtectedFields(t) {
		entry, ok := declared[object]
		require.Truef(t, ok,
			"%s has protected fields but no entry in protectedTypes; name the resource that "+
				"classifies them, the entity REST withholds them through, or why only "+
				"GraphQL serves the type", object)
		assert.Truef(t, entry.entity != nil || entry.graphQLOnly != "",
			"%s names neither the entity REST withholds it through nor why only GraphQL "+
				"serves it", object)
	}
	fields := loadProtectedFields(t)
	for object := range declared {
		_, used := fields[object]
		assert.Truef(t, used, "%s is in protectedTypes but has no protected field; remove it",
			object)
	}
}

func TestEveryProtectedFieldIsConfidentialInTheRegistry(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	declared := protectedTypes()
	checked := 0
	for object, fields := range loadProtectedFields(t) {
		resource := declared[object].resource
		if resource == "" {
			continue
		}
		for field := range fields {
			checked++
			assert.Equalf(t, permission.SensitivityConfidential,
				registry.GetFieldSensitivity(resource.String(), field),
				"%s.%s is protected, so %s must classify %q as confidential, which keeps it "+
					"out of agent tools, reports and previews too", object, field, resource, field)
		}
	}
	assert.Positive(t, checked)
}

func TestEveryProtectedFieldIsWithheldByItsEntity(t *testing.T) {
	t.Parallel()

	declared := protectedTypes()
	for object, fields := range loadProtectedFields(t) {
		entry := declared[object]
		if entry.entity == nil {
			continue
		}

		visible := entry.entity()
		visible.WithholdFields(protectedfield.AllGroups())
		withheld := entry.entity()
		withheld.WithholdFields(0)

		visibleJSON := decodeJSON(t, visible)
		withheldJSON := decodeJSON(t, withheld)
		for field := range fields {
			key := field
			if name, renamed := entry.jsonNames[field]; renamed {
				key = name
			}
			assert.NotEmptyf(t, visibleJSON[key],
				"%s.%s was cleared for a reader who holds every group", object, field)
			assert.Emptyf(t, withheldJSON[key],
				"%s.%s was left in place for a reader who holds no group", object, field)
		}
	}
}

func decodeJSON(t *testing.T, value any) map[string]any {
	t.Helper()

	raw, err := sonic.Marshal(value)
	require.NoError(t, err)
	decoded := map[string]any{}
	require.NoError(t, sonic.Unmarshal(raw, &decoded))
	for key, v := range decoded {
		switch typed := v.(type) {
		case float64:
			if typed == 0 {
				decoded[key] = nil
			}
		case string:
			if typed == "" {
				decoded[key] = nil
			}
		}
	}
	return decoded
}

func parseSchema(t *testing.T, sdl string) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{Name: "test.graphqls", Input: `
enum ProtectedFieldGroup { Personal Margins }
directive @protected(group: ProtectedFieldGroup!) on FIELD_DEFINITION
` + sdl})
	require.NoError(t, err)
	return schema
}

func TestProtectedFieldsRefusesANonNullField(t *testing.T) {
	t.Parallel()

	_, err := ProtectedFields(parseSchema(t, `
type Query { worker: Worker }
type Worker { dob: Int! @protected(group: Personal) }
`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Worker.dob")
	assert.Contains(t, err.Error(), "must be nullable")
}

func TestProtectedFieldsIgnoresOtherDirectives(t *testing.T) {
	t.Parallel()

	fields, err := ProtectedFields(parseSchema(t, `
directive @audited(group: ProtectedFieldGroup!) on FIELD_DEFINITION | INPUT_FIELD_DEFINITION
type Query { worker(input: WorkerInput): Worker }
input WorkerInput { dob: Int @audited(group: Personal) }
type Worker { dob: Int! @audited(group: Personal) }
`))

	require.NoError(t, err)
	assert.Empty(t, fields)
}

func TestProtectedFieldsReadsGroupsPerField(t *testing.T) {
	t.Parallel()

	fields, err := ProtectedFields(parseSchema(t, `
type Query { agreement: Agreement }
type Agreement {
  name: String
  floor: Float @protected(group: Margins)
  owner: String @protected(group: Personal)
}
`))

	require.NoError(t, err)
	assert.Equal(t, ProtectedFieldMap{
		"Agreement": {
			"floor": protectedfield.Margins,
			"owner": protectedfield.Personal,
		},
	}, fields)
}

type countingEngine struct {
	mocks.DenyResourcesPermissionEngine

	calls atomic.Int32
}

func (e *countingEngine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.calls.Add(1)
	return e.DenyResourcesPermissionEngine.Check(ctx, req)
}

func newFieldGuard(t *testing.T, engine permissionports.PermissionEngine) *FieldGuardExtension {
	t.Helper()

	guard := NewFieldGuardExtension(FieldGuardParams{PermissionEngine: engine})
	guard.fields = loadProtectedFields(t)
	return guard
}

func operationContext(t *testing.T, guard *FieldGuardExtension) context.Context {
	t.Helper()

	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	var opCtx context.Context
	guard.InterceptOperation(ctx, func(ctx context.Context) gqlgen.ResponseHandler {
		opCtx = ctx
		return nil
	})
	require.NotNil(t, opCtx)
	return opCtx
}

func fieldContext(ctx context.Context, object, field string) context.Context {
	return gqlgen.WithFieldContext(ctx, &gqlgen.FieldContext{
		Object: object,
		Field:  gqlgen.CollectedField{Field: &ast.Field{Name: field}},
	})
}

func resolveThrough(
	t *testing.T,
	guard *FieldGuardExtension,
	ctx context.Context,
	object, field string,
) (any, bool) {
	t.Helper()

	resolved := false
	result, err := guard.InterceptField(
		fieldContext(ctx, object, field),
		func(context.Context) (any, error) {
			resolved = true
			return "stored value", nil
		},
	)
	require.NoError(t, err)
	return result, resolved
}

func TestFieldGuardWithholdsAProtectedFieldFromAReaderWithoutTheGroup(t *testing.T) {
	t.Parallel()

	guard := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedPersonal.String()},
	})
	ctx := operationContext(t, guard)

	result, resolved := resolveThrough(t, guard, ctx, "WorkerProfile", "licenseNumber")

	assert.Nil(t, result)
	assert.False(t, resolved, "a withheld field is never resolved")
}

func TestFieldGuardResolvesAProtectedFieldForAReaderWithTheGroup(t *testing.T) {
	t.Parallel()

	guard := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedMargins.String()},
	})
	ctx := operationContext(t, guard)

	result, resolved := resolveThrough(t, guard, ctx, "WorkerProfile", "dob")
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	result, resolved = resolveThrough(t, guard, ctx, "RateAgreement", "marginFloorPercent")
	assert.Nil(t, result)
	assert.False(t, resolved)
}

func TestFieldGuardLeavesUnprotectedFieldsAlone(t *testing.T) {
	t.Parallel()

	engine := &countingEngine{DenyResourcesPermissionEngine: mocks.DenyResourcesPermissionEngine{
		Denied: []string{
			permission.ResourceProtectedPersonal.String(),
			permission.ResourceProtectedMargins.String(),
		},
	}}
	guard := newFieldGuard(t, engine)
	ctx := operationContext(t, guard)

	result, resolved := resolveThrough(t, guard, ctx, "WorkerProfile", "licenseExpiry")
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	result, resolved = resolveThrough(t, guard, ctx, "Customer", "dob")
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Zero(t, engine.calls.Load(), "an operation that reads no protected field asks nothing")
}

func TestFieldGuardAsksOncePerOperation(t *testing.T) {
	t.Parallel()

	engine := &countingEngine{}
	guard := newFieldGuard(t, engine)
	ctx := operationContext(t, guard)

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			result, err := guard.InterceptField(
				fieldContext(ctx, "WorkerCredential", "number"),
				func(context.Context) (any, error) { return "CDL-1", nil },
			)
			assert.NoError(t, err)
			assert.Equal(t, "CDL-1", result)
		})
	}
	wg.Wait()

	assert.Equal(t, int32(len(protectedfield.Groups())), engine.calls.Load())
}

func TestFieldGuardWithholdsFromAnUnauthenticatedRequest(t *testing.T) {
	t.Parallel()

	guard := newFieldGuard(t, &mocks.AllowAllPermissionEngine{})
	var ctx context.Context
	guard.InterceptOperation(t.Context(), func(opCtx context.Context) gqlgen.ResponseHandler {
		ctx = opCtx
		return nil
	})

	result, resolved := resolveThrough(t, guard, ctx, "WorkerProfile", "dob")

	assert.Nil(t, result)
	assert.False(t, resolved)
}

func TestFieldGuardWithholdsWhenTheCheckFails(t *testing.T) {
	t.Parallel()

	guard := newFieldGuard(t, &mocks.FailingPermissionEngine{
		Failing: []string{permission.ResourceProtectedPersonal.String()},
		Err:     assert.AnError,
	})
	ctx := operationContext(t, guard)

	result, resolved := resolveThrough(t, guard, ctx, "WorkerProfile", "twicCardNumber")

	assert.Nil(t, result)
	assert.False(t, resolved)
}

type fixedCeilings permission.FieldSensitivity

func (c fixedCeilings) For(context.Context, permission.Resource) permission.FieldSensitivity {
	return permission.FieldSensitivity(c)
}

func (fixedCeilings) Reads(context.Context, permission.Resource) bool { return true }

// unreadable is a reader at Restricted who cannot read one resource at all.
type unreadable struct {
	fixedCeilings
	denied permission.Resource
}

func (c unreadable) Reads(_ context.Context, resource permission.Resource) bool {
	return resource != c.denied
}

func newCeilingGuard(t *testing.T) *FieldGuardExtension {
	t.Helper()

	guard := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{})
	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	guard.resources, err = KindTypes(schema, guard.registry)
	require.NoError(t, err)
	return guard
}

func agentRead(
	t *testing.T,
	guard *FieldGuardExtension,
	level permission.FieldSensitivity,
) (context.Context, *gqlctx.FieldCeiling) {
	t.Helper()

	ceiling := gqlctx.NewFieldCeiling(fixedCeilings(level))
	return gqlctx.WithFieldCeiling(operationContext(t, guard), ceiling), ceiling
}

type fieldStep struct {
	object, field string
	nonNull       bool
	authorizes    string
}

func nestedFieldContext(ctx context.Context, steps ...fieldStep) context.Context {
	for _, step := range steps {
		def := &ast.FieldDefinition{Type: &ast.Type{NamedType: "String", NonNull: step.nonNull}}
		if step.authorizes != "" {
			def.Directives = ast.DirectiveList{{
				Name: authorizesDirective,
				Arguments: ast.ArgumentList{{
					Name:  kindResourceArgument,
					Value: &ast.Value{Kind: ast.StringValue, Raw: step.authorizes},
				}},
			}}
		}
		ctx = gqlgen.WithFieldContext(ctx, &gqlgen.FieldContext{
			Object: step.object,
			Field: gqlgen.CollectedField{Field: &ast.Field{
				Name:       step.field,
				Alias:      step.field,
				Definition: def,
			}},
		})
	}
	return ctx
}

func resolveNested(
	t *testing.T,
	guard *FieldGuardExtension,
	ctx context.Context,
	steps ...fieldStep,
) (any, bool) {
	t.Helper()

	resolved := false
	result, err := guard.InterceptField(nestedFieldContext(ctx, steps...),
		func(context.Context) (any, error) {
			resolved = true
			return "stored value", nil
		})
	require.NoError(t, err)
	return result, resolved
}

func TestFieldGuardMapsTypesToTheirResources(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)

	assert.Equal(t, permission.ResourceWorker, guard.resources["Worker"])
	assert.Equal(t, permission.ResourceCustomer, guard.resources["Customer"])
	assert.NotContains(t, guard.resources, "WorkerProfile",
		"a profile has no resource of its own; it belongs to its worker")
	assert.NotContains(t, guard.resources, "CustomerConnection")
}

func TestFieldGuardLeavesAScreenReadWithNoCeilingAlone(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx := operationContext(t, guard)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)

	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
}

func TestFieldGuardWithholdsAFieldAboveTheAgentCeiling(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityInternal)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)

	assert.Nil(t, result)
	assert.False(t, resolved, "a withheld nullable field is never resolved")
	assert.Equal(t, []string{"worker.email"}, ceiling.Withheld())
	assert.Empty(t, ceiling.Stripped())
}

func TestFieldGuardShowsAFieldWithinTheAgentCeiling(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityRestricted)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "email"},
	)

	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Empty(t, ceiling.Withheld())
}

func TestFieldGuardGatesAProfileFieldAsItsWorkers(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityRestricted)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "workers"},
		fieldStep{object: "WorkerConnection", field: "edges"},
		fieldStep{object: "WorkerEdge", field: "node"},
		fieldStep{object: "Worker", field: "profile"},
		fieldStep{object: "WorkerProfile", field: "medicalExaminerNpi"},
	)

	assert.Nil(t, result)
	assert.False(t, resolved, "confidential is never shown, whatever the ceiling")
	assert.Equal(t, []string{"workers.edges.node.profile.medicalExaminerNpi"}, ceiling.Withheld())
}

func TestFieldGuardResolvesAWithheldNonNullFieldAndMarksItForStripping(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityInternal)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "email", nonNull: true},
	)

	assert.Equal(t, "stored value", result, "nulling a non-null field would null its parent")
	assert.True(t, resolved)
	require.Len(t, ceiling.Stripped(), 1)
	assert.Equal(t, "worker.email", ceiling.Stripped()[0].String())
}

func TestFieldGuardReducesANestedUserToItsName(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityRestricted)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "customer"},
		fieldStep{object: "Customer", field: "createdBy"},
		fieldStep{object: "User", field: "emailAddress"},
	)
	assert.Nil(t, result)
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "customer"},
		fieldStep{object: "Customer", field: "createdBy"},
		fieldStep{object: "User", field: "name"},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Equal(t, []string{"customer.createdBy.emailAddress"}, ceiling.Withheld())
}

func TestFieldGuardWithholdsAProtectedFieldFromAnAgentReadAndSaysSo(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	guard.engine = &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedMargins.String()},
	}
	ctx, ceiling := agentRead(t, guard, permission.SensitivityRestricted)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "rateAgreement"},
		fieldStep{object: "RateAgreement", field: "marginFloorPercent"},
	)

	assert.Nil(t, result)
	assert.False(t, resolved)
	assert.Equal(t, []string{"rateAgreement.marginFloorPercent"}, ceiling.Withheld())
}

func TestFieldGuardNeverWithholdsARecordsID(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ctx, ceiling := agentRead(t, guard, permission.SensitivityPublic)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "worker"},
		fieldStep{object: "Worker", field: "id", nonNull: true},
	)

	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Empty(t, ceiling.Withheld())
}

// A tractor's driver is a worker: a reader who may read tractors but not
// workers sees the driver's reference view (their name) and nothing else.
func TestFieldGuardShowsOnlyTheReferenceViewOfARecordTheReaderCannotRead(t *testing.T) {
	t.Parallel()

	guard := newCeilingGuard(t)
	ceiling := gqlctx.NewFieldCeiling(unreadable{
		fixedCeilings: fixedCeilings(permission.SensitivityRestricted),
		denied:        permission.ResourceWorker,
	})
	ctx := gqlctx.WithFieldCeiling(operationContext(t, guard), ceiling)

	result, resolved := resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "tractors"},
		fieldStep{object: "Tractor", field: "primaryWorker"},
		fieldStep{object: "Worker", field: "wholeName"},
	)
	assert.Equal(t, "stored value", result, "a driver's name is in the worker reference view")
	assert.True(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "tractors"},
		fieldStep{object: "Tractor", field: "primaryWorker"},
		fieldStep{object: "Worker", field: "email"},
	)
	assert.Nil(t, result)
	assert.False(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "tractors"},
		fieldStep{object: "Tractor", field: "primaryWorker"},
		fieldStep{object: "Worker", field: "id", nonNull: true},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)

	result, resolved = resolveNested(t, guard, ctx,
		fieldStep{object: "Query", field: "tractors"},
		fieldStep{object: "Tractor", field: "code"},
	)
	assert.Equal(t, "stored value", result)
	assert.True(t, resolved)
	assert.Equal(t, []string{"tractors.primaryWorker.email"}, ceiling.Withheld())
}
