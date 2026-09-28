package accountingmappingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanConfirmShowsTheConfirmationWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	plan, err := h.svc.PlanConfirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		Items:      shown(ar),
		Source:     accountingsync.MappingSourceAgent,
	})

	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, accountingsync.MappingStateProposed, plan.Changes[0].Before.State)
	assert.Equal(t, accountingsync.MappingStateConfirmed, plan.Changes[0].After.State)
	assert.Equal(t, accountingsync.MappingSourceAgent, plan.Changes[0].After.Source)
	assert.Equal(t, accountingsync.MappingStateProposed,
		h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR).State)
	assert.Empty(t, h.audit.entries)
}

func TestPlanRejectShowsTheRejectionWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	change, err := h.svc.PlanReject(t.Context(), &services.AccountingMappingActionRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		ID:         cus.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "50", change.Before.ExternalID)
	assert.Equal(t, accountingsync.MappingStateUnmatched, change.After.State)
	assert.Contains(t, change.After.Signals.RejectedExternalIDs, "50")
	assert.NotContains(t, change.Before.Signals.RejectedExternalIDs, "50")
	assert.Equal(t, accountingsync.MappingStateProposed,
		h.object(t, accountingsync.TargetCustomer, h.customerID).State)
}
