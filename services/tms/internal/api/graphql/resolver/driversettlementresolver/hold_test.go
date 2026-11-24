package driversettlementresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoldingAndReleasingNeedTheHoldPermissionNotApprove(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		"driver_settlement|read":    true,
		"driver_settlement|update":  true,
		"driver_settlement|approve": true,
	}}
	mutation := &MutationResolver{Deps: &Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
	}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())
	settlementID := pulid.MustNew("dstl_").String()

	_, err := mutation.HoldDriverSettlement(ctx, gqlmodel.HoldDriverSettlementInput{
		SettlementID: settlementID,
		Reason:       driversettlement.HoldReasonRateDispute,
	})
	require.Error(t, err)
	_, err = mutation.ReleaseDriverSettlementHold(ctx, settlementID)
	require.Error(t, err)

	require.Len(t, engine.Requests, 2)
	for _, req := range engine.Requests {
		assert.Equal(t, permission.ResourceDriverSettlement.String(), req.Resource)
		assert.Equal(t, permission.OpHold, req.Operation)
	}
}

func TestTheHoldReasonIsNullUnlessTheSettlementIsHeld(t *testing.T) {
	t.Parallel()

	resolver := &DriverSettlementResolver{Deps: &Deps{}}
	reason, err := resolver.HoldReason(context.Background(), &driversettlement.Settlement{})
	require.NoError(t, err)
	assert.Nil(t, reason)

	at := int64(1)
	reason, err = resolver.HoldReason(context.Background(), &driversettlement.Settlement{
		HoldReason: driversettlement.HoldReasonDeductionReview,
		HeldAt:     &at,
	})
	require.NoError(t, err)
	require.NotNil(t, reason)
	assert.Equal(t, driversettlement.HoldReasonDeductionReview, *reason)
}

func TestAHeldBlockerSaysOnHoldWithItsReasonAndNote(t *testing.T) {
	t.Parallel()

	rows := blockersToModel(context.Background(), []*settlementreleaseports.Settlement{
		{
			ID: pulid.MustNew("dstl_"), WorkerID: pulid.MustNew("wrk_"),
			Status:     driversettlement.StatusApproved,
			Block:      driversettlement.ReleaseBlockHeld,
			HoldReason: driversettlement.HoldReasonRateDispute,
			HoldNote:   "Waiting on the corrected rate confirmation",
		},
		{
			ID: pulid.MustNew("dstl_"), WorkerID: pulid.MustNew("wrk_"),
			Status:          driversettlement.StatusApproved,
			Block:           driversettlement.ReleaseBlockDisputeOpen,
			DisputeCategory: driversettlement.DisputeCategoryIncorrectRate,
		},
	})

	require.Len(t, rows, 2)
	assert.Equal(t, "On hold", rows[0].Reason)
	assert.Empty(t, rows[0].Message)
	require.NotNil(t, rows[0].HoldReason)
	assert.Equal(t, "Rate dispute", *rows[0].HoldReason)
	require.NotNil(t, rows[0].HoldNote)
	assert.Equal(t, "Waiting on the corrected rate confirmation", *rows[0].HoldNote)
	assert.Nil(t, rows[1].HoldReason, "only a held settlement carries a hold")
}
