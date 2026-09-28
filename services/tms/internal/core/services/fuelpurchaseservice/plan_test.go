package fuelpurchaseservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanCreatePurchase_ResolvesWhatCreateWouldSaveAndSavesNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	purchase := h.newPurchase()
	planned, err := h.svc.PlanCreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         purchase,
		JurisdictionCode: "tx",
		UserID:           h.userID,
	})
	require.NoError(t, err)
	assert.False(t, planned.JurisdictionID.IsNil())
	assert.Equal(t, fuelpurchase.PurchaseSourceManual, planned.Source)
	assert.Equal(t, "100.000", planned.Gallons.StringFixed(3))
	assert.Empty(t, h.repo.allPurchases())
	assert.True(t, purchase.JurisdictionID.IsNil(), "the caller's purchase is not changed")

	_, err = h.svc.PlanCreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "Narnia",
		UserID:           h.userID,
	})
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "jurisdictionId", fieldErr.Field)
}

func TestPlanUpdatePurchase_ShowsTheChangeWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "TX",
		UserID:           h.userID,
	})
	require.NoError(t, err)

	edited := *created
	edited.Quantity = decimal.RequireFromString("200")
	change, err := h.svc.PlanUpdatePurchase(t.Context(), &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: h.tenant,
		Purchase:   &edited,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, "100.000", change.Before.Gallons.StringFixed(3))
	assert.Equal(t, "200.000", change.After.Gallons.StringFixed(3))

	stored := h.repo.allPurchases()
	require.Len(t, stored, 1)
	assert.Equal(t, created.Version, stored[0].Version)
	assert.Equal(t, "100.000", stored[0].Gallons.StringFixed(3))

	flipped := *created
	flipped.Source = fuelpurchase.PurchaseSourceCardImport
	_, err = h.svc.PlanUpdatePurchase(t.Context(), &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: h.tenant,
		Purchase:   &flipped,
		UserID:     h.userID,
	})
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "source", fieldErr.Field)
}

func TestPlanDeletePurchase_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "TX",
		UserID:           h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.PlanDeletePurchase(t.Context(), &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: h.tenant,
		ID:         created.ID,
		Version:    created.Version + 1,
		UserID:     h.userID,
	})
	require.Error(t, err)

	planned, err := h.svc.PlanDeletePurchase(t.Context(), &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: h.tenant,
		ID:         created.ID,
		Version:    created.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, planned.ID)
	assert.Len(t, h.repo.allPurchases(), 1)
}

func TestPlanAssignCard_ActivatesASuspendedCardItAssigns(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	input := h.newCard("4411")
	input.Status = fuelpurchase.CardStatusSuspended
	card, err := h.svc.CreateCard(t.Context(), input, h.userID)
	require.NoError(t, err)

	tractorID := h.tractorA.ID
	change, err := h.svc.PlanAssignCard(t.Context(), &fuelpurchaseservice.AssignCardRequest{
		TenantInfo:        h.tenant,
		ID:                card.ID,
		Version:           card.Version,
		AssignedTractorID: &tractorID,
		UserID:            h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.CardStatusSuspended, change.Before.Status)
	assert.Equal(t, fuelpurchase.CardStatusActive, change.After.Status)
	require.NotNil(t, change.After.AssignedTractorID)
	assert.Equal(t, tractorID, *change.After.AssignedTractorID)

	stored, err := h.svc.GetCard(t.Context(), h.tenant, card.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.AssignedTractorID)
	assert.Equal(t, fuelpurchase.CardStatusSuspended, stored.Status)
}

func TestPlanCommit_ListsThePurchasesAndCommitsNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n"+
		"07/15/2026,TRC-002,5678,OK,ULSD,80,3.799,303.92,T-2\n")

	req := &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	}
	plan, err := h.svc.PlanCommit(t.Context(), req)
	require.NoError(t, err)
	assert.Len(t, plan.Purchases, 2)
	assert.Equal(t, batch.ID, plan.Batch.ID)
	assert.Zero(t, h.repo.commits)
	assert.Empty(t, h.repo.allPurchases())

	_, err = h.svc.Commit(t.Context(), req)
	require.NoError(t, err)

	committed := h.repo.batch(batch.ID)
	_, err = h.svc.PlanCommit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    committed.Version,
		UserID:     h.userID,
	})
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "status", fieldErr.Field)
}

func TestPlanDiscard_ShowsTheDiscardAndRefusesACommittedImport(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	change, err := h.svc.PlanDiscard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.ImportStatusPending, change.Before.Status)
	assert.Equal(t, fuelpurchase.ImportStatusDiscarded, change.After.Status)
	assert.Equal(t, fuelpurchase.ImportStatusPending, h.repo.batch(batch.ID).Status)

	committedBatch := h.createImport(t)
	staged := h.stage(t, committedBatch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")
	committed, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    committedBatch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	_, err = h.svc.PlanDiscard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    committedBatch.ID,
		Version:    committed.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
}

func TestPlanResolveRows_CountsWhatWouldResolveAndPostsNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	sync := syncQueued(t, h, feedRow("C-1001", "TRC-UNKNOWN", "4411"))
	h.registerTractor("TRC-UNKNOWN")
	batch := h.repo.batch(sync.BatchID)

	plan, err := h.svc.PlanResolveRows(t.Context(), &fuelpurchaseservice.ResolveRowsRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Reviewed)
	assert.Equal(t, 1, plan.Resolved)
	assert.Zero(t, plan.Queued)
	assert.Empty(t, h.repo.allPurchases())
	assert.Equal(t, batch.Version, h.repo.batch(batch.ID).Version)
}
