package workeremploymentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func terminationInput() gqlmodel.RecordWorkerEmploymentEventInput {
	reason := "Resigned"
	return gqlmodel.RecordWorkerEmploymentEventInput{
		WorkerID:    pulid.MustNew("wrk_").String(),
		Kind:        employment.EmploymentEventTerminated,
		EffectiveAt: 1_800_000_000,
		Reason:      &reason,
	}
}

func readOnlyEngine() *resolvertest.GrantingPermissionEngine {
	return &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceWorkerEmploymentEvent.String() + "|" + string(permission.OpRead): true,
	}}
}

// The preview reads what recording would change, including pay and time off,
// so it needs the write's own permission: read alone is refused.
func TestPreviewWorkerEmploymentEvent_NeedsCreate(t *testing.T) {
	t.Parallel()

	engine := readOnlyEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	preview, err := (&QueryResolver{deps}).PreviewWorkerEmploymentEvent(ctx, terminationInput())

	assert.Nil(t, preview)
	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceWorkerEmploymentEvent,
		permission.OpCreate,
	)
}

func TestWorkerEmploymentEventOptions_NeedsRead(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	options, err := (&QueryResolver{deps}).WorkerEmploymentEventOptions(
		ctx,
		pulid.MustNew("wrk_").String(),
	)

	assert.Nil(t, options)
	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceWorkerEmploymentEvent,
		permission.OpRead,
	)
}

func TestRecordWorkerEmploymentEvent_NeedsCreate(t *testing.T) {
	t.Parallel()

	engine := readOnlyEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	result, err := (&MutationResolver{deps}).RecordWorkerEmploymentEvent(ctx, terminationInput())

	assert.Nil(t, result)
	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceWorkerEmploymentEvent,
		permission.OpCreate,
	)
}

func TestRecordRequest_MapsTheSameInputForPreviewAndRecord(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	manager := pulid.MustNew("usr_").String()
	cleared := ""
	input := terminationInput()
	input.Kind = employment.EmploymentEventTransferred
	input.ManagerID = &manager
	input.FleetCodeID = &cleared

	req, err := recordRequest(authCtx, input)
	require.NoError(t, err)

	assert.Equal(t, authCtx.UserID, req.UserID)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	require.NotNil(t, req.ManagerID)
	assert.Equal(t, manager, req.ManagerID.String())
	require.NotNil(t, req.FleetCodeID)
	assert.True(t, req.FleetCodeID.IsNil(), "an empty fleet code clears the fleet")
	assert.Equal(t, "Resigned", req.Reason)

	bad := "not-an-id"
	input.ManagerID = &bad
	_, err = recordRequest(authCtx, input)
	require.Error(t, err)
}

func TestWorkerEmploymentEffectDays_RendersTheDecimalOrNull(t *testing.T) {
	t.Parallel()

	resolver := &WorkerEmploymentEffectResolver{}
	days := decimal.NewFromFloat(6.5)

	rendered, err := resolver.Days(t.Context(), &employment.Effect{Days: &days})
	require.NoError(t, err)
	require.NotNil(t, rendered)
	assert.Equal(t, "6.5", *rendered)

	none, err := resolver.Days(t.Context(), &employment.Effect{})
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestPreviewValues_AreSortedByKey(t *testing.T) {
	t.Parallel()

	resolver := &WorkerEmploymentEventPreviewResolver{}
	preview := &workeremploymentservice.Preview{
		ToValues: map[string]string{"status": "Inactive", "terminationDate": "1800000000"},
	}

	values, err := resolver.ToValues(t.Context(), preview)
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.Equal(t, "status", values[0].Key)
	assert.Equal(t, "terminationDate", values[1].Key)
}
