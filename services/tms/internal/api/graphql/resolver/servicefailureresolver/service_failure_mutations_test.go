package servicefailureresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/servicefailureportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServiceFailureResolvers_CheckTheRESTPermissions(t *testing.T) {
	t.Parallel()

	failureID := pulid.MustNew("sf_").String()
	lifecycle := gqlmodel.ServiceFailureLifecycleInput{Version: 1}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"serviceFailureEdi214Payload",
			permission.OpExport,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ServiceFailureEDI214Payload(ctx, failureID)
				return err
			},
		},
		{
			"serviceFailureEdi214Readiness",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ServiceFailureEDI214Readiness(ctx, failureID, nil)
				return err
			},
		},
		{
			"serviceFailureEdi214Status",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).ServiceFailureEDI214Status(ctx, failureID)
				return err
			},
		},
		{"updateServiceFailure", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateServiceFailure(
				ctx,
				failureID,
				gqlmodel.UpdateServiceFailureInput{Version: 1},
			)
			return err
		}},
		{"reviewServiceFailure", permission.OpApprove, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ReviewServiceFailure(ctx, failureID, lifecycle)
			return err
		}},
		{"resolveServiceFailure", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).ResolveServiceFailure(ctx, failureID, lifecycle)
			return err
		}},
		{"voidServiceFailure", permission.OpArchive, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).VoidServiceFailure(ctx, failureID, lifecycle)
			return err
		}},
		{
			"evaluateShipmentServiceFailures",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).EvaluateShipmentServiceFailures(
					ctx,
					pulid.MustNew("shp_").String(),
					nil,
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceServiceFailure,
				call.operation,
			)
		})
	}
}

func TestUpdateServiceFailure_TakesTheShipmentFromTheStoredFailure(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	failureID := pulid.MustNew("sf_")
	shipmentID := pulid.MustNew("shp_")
	reasonCodeID := pulid.MustNew("sfrc_")
	stored := &servicefailure.ServiceFailure{ID: failureID, ShipmentID: shipmentID}
	updated := &servicefailure.ServiceFailure{ID: failureID, ShipmentID: shipmentID, Notes: "Late"}

	service := servicefailureportsmock.NewMockServiceFailureService(t)
	service.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *servicefailureports.GetServiceFailureByIDRequest) bool {
			return req.ID == failureID && req.TenantInfo.OrgID == authCtx.OrganizationID
		})).
		Return(stored, nil)
	service.EXPECT().
		Update(
			mock.Anything,
			mock.MatchedBy(func(req *servicefailureports.UpdateServiceFailureRequest) bool {
				return req.ID == failureID &&
					req.ShipmentID == shipmentID &&
					req.ReasonCodeID == reasonCodeID &&
					req.Notes == "Late" &&
					req.X12StatusCodeOverride == "SD" &&
					req.Version == 4 &&
					req.TenantInfo.BuID == authCtx.BusinessUnitID
			}),
			mock.Anything,
		).
		Return(updated, nil)

	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		ServiceFailureSvc: service,
	}
	reasonRaw := reasonCodeID.String()
	notes, override := "Late", "SD"
	result, err := (&MutationResolver{deps}).UpdateServiceFailure(
		resolvertest.AuthedContext(t, authCtx),
		failureID.String(),
		gqlmodel.UpdateServiceFailureInput{
			ReasonCodeID:          &reasonRaw,
			Notes:                 &notes,
			X12StatusCodeOverride: &override,
			Version:               4,
		},
	)
	require.NoError(t, err)
	assert.Same(t, updated, result)
}

func TestLifecycleRequestFromInput_LeavesAnAbsentReasonCodeUnset(t *testing.T) {
	t.Parallel()

	current := &servicefailure.ServiceFailure{
		ID:         pulid.MustNew("sf_"),
		ShipmentID: pulid.MustNew("shp_"),
	}
	notes := "Carrier broke down"

	req, err := lifecycleRequestFromInput(
		current,
		&gqlmodel.ServiceFailureLifecycleInput{Notes: &notes, Version: 2},
		resolvercore.TenantInfo(resolvertest.UserAuthContext()),
	)
	require.NoError(t, err)

	assert.Equal(t, current.ID, req.ID)
	assert.Equal(t, current.ShipmentID, req.ShipmentID)
	assert.True(t, req.ReasonCodeID.IsNil())
	assert.Equal(t, "Carrier broke down", req.Notes)
	assert.Equal(t, int64(2), req.Version)
}

func TestEdi214ReadinessTrigger_PreviewsTheNextStepUnlessOneIsNamed(t *testing.T) {
	t.Parallel()

	resolved := servicefailureports.ServiceFailureEDITriggerResolved
	reviewed := servicefailureports.ServiceFailureEDITriggerReviewed

	trigger, status := edi214ReadinessTrigger(nil, servicefailure.StatusOpen)
	assert.Equal(t, reviewed, trigger)
	assert.Equal(t, servicefailure.StatusReviewed, status)

	trigger, status = edi214ReadinessTrigger(nil, servicefailure.StatusReviewed)
	assert.Equal(t, resolved, trigger)
	assert.Equal(t, servicefailure.StatusResolved, status)

	trigger, status = edi214ReadinessTrigger(&resolved, servicefailure.StatusOpen)
	assert.Equal(t, resolved, trigger)
	assert.Equal(t, servicefailure.StatusResolved, status)

	trigger, status = edi214ReadinessTrigger(&reviewed, servicefailure.StatusReviewed)
	assert.Equal(t, reviewed, trigger)
	assert.Equal(t, servicefailure.StatusReviewed, status)
}
