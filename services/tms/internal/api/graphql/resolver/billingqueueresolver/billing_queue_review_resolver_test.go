package billingqueueresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeReviewService struct {
	billingqueueports.BillingQueueReviewService
	activity *billingqueueports.ListBillingQueueActivityRequest
}

func (f *fakeReviewService) ListActivity(
	_ context.Context,
	req *billingqueueports.ListBillingQueueActivityRequest,
) (*billingqueueports.BillingQueueEventPage, error) {
	f.activity = req
	return &billingqueueports.BillingQueueEventPage{}, nil
}

func TestBillingQueueItem_ReadsTheItemWithItsShipmentDetailsAndReview(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	service := billingqueueportsmock.NewMockBillingQueueService(t)
	r := &QueryResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		BillingQueueService: service,
	}}
	itemID := pulid.MustNew("bqi_")
	review := &billingqueue.Review{NeedsCount: 2, Blocker: billingqueue.BlockerIssue}
	payerShare := &billingqueue.PayerShare{IsSplit: true}

	var captured *billingqueueports.GetBillingQueueItemByIDRequest
	service.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *billingqueueports.GetBillingQueueItemByIDRequest,
		) (*billingqueue.BillingQueueItem, error) {
			captured = req
			return &billingqueue.BillingQueueItem{
				ID:                itemID,
				Number:            "BQ-1",
				PacketDocumentIDs: []pulid.ID{pulid.MustNew("doc_")},
				AgentReads:        []billingqueue.AgentRead{{Kind: billingqueue.AgentReadTerms}},
				Review:            review,
				PayerShare:        payerShare,
			}, nil
		}).
		Once()

	item, err := r.BillingQueueItem(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		itemID.String(),
	)

	require.NoError(t, err)
	assert.Equal(t, permission.ResourceBillingQueue.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpRead, engine.Request.Operation)
	assert.True(t, captured.ExpandShipmentDetails)
	assert.Equal(t, itemID, captured.ItemID)
	assert.Equal(t, "BQ-1", item.Number)
	assert.Same(t, review, item.Review)
	assert.Same(t, payerShare, item.PayerShare)
	assert.Len(t, item.PacketDocumentIds, 1)
	require.Len(t, item.AgentReads, 1)
	assert.Equal(t, billingqueue.AgentReadTerms, item.AgentReads[0].Kind)
	assert.Empty(t, item.DetentionHolds)
}

func TestReassignBillingQueueCharge_MapsTheAllocationsAndReturnsEveryItem(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.RecordingPermissionEngine{}
	service := billingqueueportsmock.NewMockBillingQueueService(t)
	r := &MutationResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		BillingQueueService: service,
	}}
	itemID := pulid.MustNew("bqi_")
	payerID := pulid.MustNew("cus_")
	chargeID := pulid.MustNew("adc_")
	createdID := pulid.MustNew("bqi_")
	percent := "40"
	additionalChargeID := chargeID.String()

	var captured *billingqueueports.ReassignChargeRequest
	service.EXPECT().
		ReassignCharge(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *billingqueueports.ReassignChargeRequest,
			_ *permissionports.RequestActor,
		) (*billingqueueports.ReassignChargeResult, error) {
			captured = req
			return &billingqueueports.ReassignChargeResult{
				Item:           &billingqueue.BillingQueueItem{ID: itemID},
				Items:          []*billingqueue.BillingQueueItem{{ID: itemID}, {ID: createdID}},
				CreatedItemIDs: []pulid.ID{createdID},
			}, nil
		}).
		Once()

	result, err := r.ReassignBillingQueueCharge(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		itemID.String(),
		gqlmodel.BillingQueueReassignChargeInput{
			ChargeKind:         shipmentvocab.ChargeAllocationKindAccessorial,
			AdditionalChargeID: &additionalChargeID,
			Allocations: []*gqlmodel.ChargeAllocationInput{{
				BillToCustomerID: payerID.String(),
				Method:           shipmentvocab.ChargeAllocationMethodPercent,
				Percent:          &percent,
			}},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, permission.OpUpdate, engine.Request.Operation)
	assert.Equal(t, chargeID, captured.AdditionalChargeID)
	require.Len(t, captured.Allocations, 1)
	assert.Equal(t, payerID, captured.Allocations[0].BillToCustomerID)
	assert.Equal(
		t,
		shipmentvocab.ChargeAllocationKindAccessorial,
		captured.Allocations[0].ChargeKind,
	)
	assert.True(t, decimal.NewFromInt(40).Equal(captured.Allocations[0].Percent.Decimal))
	assert.Equal(t, itemID.String(), result.Item.ID)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, []string{createdID.String()}, result.CreatedItemIds)
	assert.Empty(t, result.CanceledItemIds)
}

func TestUpdateBillingQueueCharges_RefusedWithoutBillingQueueUpdate(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	r := &MutationResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		BillingQueueService: billingqueueportsmock.NewMockBillingQueueService(t),
	}}

	_, err := r.UpdateBillingQueueCharges(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("bqi_").String(),
		gqlmodel.BillingQueueUpdateChargesInput{},
	)

	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceBillingQueue,
		permission.OpUpdate,
	)
}

func TestBillingQueueItemActivity_CapsTheLimitAndContinuesBeforeTheCursor(t *testing.T) {
	t.Parallel()

	review := &fakeReviewService{}
	r := &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		BillingQueueReviewService: review,
	}}
	beforeID := pulid.MustNew("bqe_")
	limit := 1000

	_, err := r.BillingQueueItemActivity(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("bqi_").String(),
		&limit,
		&gqlmodel.BillingQueueActivityCursorInput{At: 1_789_000_000, ID: beforeID.String()},
	)

	require.NoError(t, err)
	assert.Equal(t, maxActivityLimit, review.activity.Limit)
	assert.Equal(t, int64(1_789_000_000), review.activity.BeforeAt)
	assert.Equal(t, beforeID, review.activity.BeforeID)
}
