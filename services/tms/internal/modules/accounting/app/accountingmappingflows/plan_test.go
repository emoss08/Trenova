package accountingmappingflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanConfirmShowsTheConfirmationWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	plan, err := h.svc.decisions.PlanConfirm(
		t.Context(),
		&accountingmappingports.ConfirmAccountingMappingsRequest{
			TenantInfo: h.tenant,
			UserID:     h.tenant.UserID,
			Items:      shown(ar),
			Source:     accountingmapping.MappingSourceAgent,
		},
	)

	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, accountingmapping.MappingStateProposed, plan.Changes[0].Before.State)
	assert.Equal(t, accountingmapping.MappingStateConfirmed, plan.Changes[0].After.State)
	assert.Equal(t, accountingmapping.MappingSourceAgent, plan.Changes[0].After.Source)
	assert.Equal(t, accountingmapping.MappingStateProposed,
		h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR).State)
	assert.Empty(t, h.audit.entries)
}

func TestPlanRejectShowsTheRejectionWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	change, err := h.svc.decisions.PlanReject(
		t.Context(),
		&accountingmappingports.AccountingMappingActionRequest{
			TenantInfo: h.tenant,
			UserID:     h.tenant.UserID,
			ID:         cus.ID,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "50", change.Before.ExternalID)
	assert.Equal(t, accountingmapping.MappingStateUnmatched, change.After.State)
	assert.Contains(t, change.After.Signals.RejectedExternalIDs, "50")
	assert.NotContains(t, change.Before.Signals.RejectedExternalIDs, "50")
	assert.Equal(t, accountingmapping.MappingStateProposed,
		h.object(t, accountingsync.TargetCustomer, h.customerID).State)
}
