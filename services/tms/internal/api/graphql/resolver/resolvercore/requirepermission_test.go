package resolvercore

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decidedEngine struct {
	permissionports.PermissionEngine

	result permissionports.PermissionCheckResult
	asked  int
}

func (e *decidedEngine) Check(
	context.Context,
	*permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.asked++
	result := e.result

	return &result, nil
}

func offPlanResult() permissionports.PermissionCheckResult {
	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)

	return permissionports.PermissionCheckResult{
		Reason:     permissionports.ReasonNotOnPlan,
		PlanDenial: &denial,
	}
}

func personContext(ctx context.Context) context.Context {
	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func TestRequirePermission_AnOffPlanResourceIsAPlanRestriction(t *testing.T) {
	t.Parallel()

	core := &Core{PermissionEngine: &decidedEngine{result: offPlanResult()}}

	got, err := core.RequirePermission(
		personContext(t.Context()),
		permission.ResourceGeneralLedgerAccount,
		permission.OpRead,
	)

	assert.Nil(t, got)
	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, errortypes.ErrPlanRestricted, restriction.GetCode())
	assert.Equal(t, "tms.accounting", restriction.Feature)
	assert.Equal(t, "Accounting & Cash", restriction.AddOn)
	assert.False(t, errortypes.IsAuthorizationError(err))
}

func TestRequirePermission_AnOrdinaryDenialIsStillAnAuthorizationError(t *testing.T) {
	t.Parallel()

	core := &Core{PermissionEngine: &decidedEngine{
		result: permissionports.PermissionCheckResult{Reason: "no_permission"},
	}}

	got, err := core.RequirePermission(
		personContext(t.Context()),
		permission.ResourceShipment,
		permission.OpDelete,
	)

	assert.Nil(t, got)
	require.True(t, errortypes.IsAuthorizationError(err), "got %v", err)
	assert.False(t, errortypes.IsPlanRestrictionError(err))
	assert.Equal(
		t,
		"You don't have permission to perform this action: shipment delete",
		err.Error(),
	)
}

func TestRequirePermission_AMemoisedPlanDenialKeepsItsReason(t *testing.T) {
	t.Parallel()

	engine := &decidedEngine{result: offPlanResult()}
	core := &Core{PermissionEngine: engine}
	ctx := gqlctx.WithPermissionMemo(personContext(t.Context()), gqlctx.NewPermissionMemo())

	for range 2 {
		_, err := core.RequirePermission(
			ctx,
			permission.ResourceGeneralLedgerAccount,
			permission.OpRead,
		)
		assert.True(t, errortypes.IsPlanRestrictionError(err), "got %v", err)
	}
	assert.Equal(t, 1, engine.asked)
}
