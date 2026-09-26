package driversettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func planFixture(status driversettlement.Status) *driversettlement.Settlement {
	earningID := pulid.MustNew("dstll_")
	return &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		SettlementNumber: "DS-100",
		Status:           status,
		CurrencyCode:     "USD",
		Lines: []*driversettlement.SettlementLine{
			{
				ID:          earningID,
				Category:    driversettlement.LineCategoryEarning,
				Description: "Linehaul",
				AmountMinor: 150000,
			},
			{
				ID:          pulid.MustNew("dstll_"),
				Category:    driversettlement.LineCategoryDeduction,
				Description: "Insurance",
				AmountMinor: -5000,
			},
		},
	}
}

func TestPlanTransitions_FollowTheSettlementLifecycle(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(1_790_000_000)

	cases := []struct {
		name   string
		from   driversettlement.Status
		plan   func(*driversettlement.Settlement) error
		to     driversettlement.Status
		refuse bool
	}{
		{
			name: "submit a draft",
			from: driversettlement.StatusDraft,
			plan: func(s *driversettlement.Settlement) error { return PlanSubmit(s, userID, now) },
			to:   driversettlement.StatusPendingApproval,
		},
		{
			name:   "submit twice",
			from:   driversettlement.StatusPendingApproval,
			plan:   func(s *driversettlement.Settlement) error { return PlanSubmit(s, userID, now) },
			refuse: true,
		},
		{
			name: "approve a pending one",
			from: driversettlement.StatusPendingApproval,
			plan: func(s *driversettlement.Settlement) error { return PlanApprove(s, userID, now) },
			to:   driversettlement.StatusApproved,
		},
		{
			name:   "approve a draft",
			from:   driversettlement.StatusDraft,
			plan:   func(s *driversettlement.Settlement) error { return PlanApprove(s, userID, now) },
			refuse: true,
		},
		{
			name: "reject a pending one",
			from: driversettlement.StatusPendingApproval,
			plan: func(s *driversettlement.Settlement) error { return PlanReject(s, "Wrong miles") },
			to:   driversettlement.StatusDraft,
		},
		{
			name:   "reject without a reason",
			from:   driversettlement.StatusPendingApproval,
			plan:   func(s *driversettlement.Settlement) error { return PlanReject(s, "") },
			refuse: true,
		},
		{
			name: "post an approved one",
			from: driversettlement.StatusApproved,
			plan: func(s *driversettlement.Settlement) error { return PlanPost(s, userID, now) },
			to:   driversettlement.StatusPosted,
		},
		{
			name: "mark a posted one paid",
			from: driversettlement.StatusPosted,
			plan: func(s *driversettlement.Settlement) error {
				return PlanMarkPaid(s, &MarkPaidInput{
					PaymentMethod: "ACH",
					PaidAt:        now,
					UserID:        userID,
				})
			},
			to: driversettlement.StatusPaid,
		},
		{
			name: "mark paid without a method",
			from: driversettlement.StatusPosted,
			plan: func(s *driversettlement.Settlement) error {
				return PlanMarkPaid(s, &MarkPaidInput{PaidAt: now, UserID: userID})
			},
			refuse: true,
		},
		{
			name: "void a posted one",
			from: driversettlement.StatusPosted,
			plan: func(s *driversettlement.Settlement) error {
				return PlanVoid(s, "Duplicate", userID, now)
			},
			to: driversettlement.StatusVoided,
		},
		{
			name: "void without a reason",
			from: driversettlement.StatusPosted,
			plan: func(s *driversettlement.Settlement) error {
				return PlanVoid(s, "", userID, now)
			},
			refuse: true,
		},
		{
			name:   "recalculate an approved one",
			from:   driversettlement.StatusApproved,
			plan:   PlanRecalculate,
			refuse: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entity := planFixture(tc.from)
			err := tc.plan(entity)
			if tc.refuse {
				require.Error(t, err)
				assert.True(t, errortypes.IsError(err))
				assert.Equal(t, tc.from, entity.Status)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.to, entity.Status)
		})
	}
}

func TestPlanReject_KeepsTheReasonInTheNotes(t *testing.T) {
	t.Parallel()

	entity := planFixture(driversettlement.StatusPendingApproval)
	entity.Notes = "Checked by payroll"
	require.NoError(t, PlanReject(entity, "Missing a stop"))

	assert.Equal(t, "Checked by payroll\nRejected: Missing a stop", entity.Notes)
	assert.Nil(t, entity.SubmittedAt)
}

func TestPlanAddAdjustment_AddsALineAndFlagsIt(t *testing.T) {
	t.Parallel()

	entity := planFixture(driversettlement.StatusDraft)
	entity.SyncTotals()
	require.NoError(t, PlanAddAdjustment(entity, &AdjustmentLineInput{
		Description: "Detention pay",
		AmountMinor: 7500,
	}))

	require.Len(t, entity.Lines, 3)
	assert.Equal(t, driversettlement.LineCategoryAdjustment, entity.Lines[2].Category)
	assert.Equal(t, int64(157500), entity.GrossEarningsMinor)
	assert.Equal(t, int64(152500), entity.NetPayMinor)
	assert.True(t, entity.HasExceptions)

	refused := planFixture(driversettlement.StatusApproved)
	require.Error(t, PlanAddAdjustment(refused, &AdjustmentLineInput{
		Description: "Late",
		AmountMinor: 100,
	}))
	require.Error(t, PlanAddAdjustment(entity, &AdjustmentLineInput{Description: "Zero"}))
}

func TestPlanRemoveAdjustment_RemovesOnlyAManualLine(t *testing.T) {
	t.Parallel()

	entity := planFixture(driversettlement.StatusDraft)
	require.NoError(t, PlanAddAdjustment(entity, &AdjustmentLineInput{
		Description: "Bonus",
		AmountMinor: 1000,
	}))
	adjustmentID := pulid.MustNew("dstll_")
	entity.Lines[2].ID = adjustmentID

	require.Error(t, PlanRemoveAdjustment(entity, entity.Lines[0].ID))
	require.Error(t, PlanRemoveAdjustment(entity, pulid.MustNew("dstll_")))

	require.NoError(t, PlanRemoveAdjustment(entity, adjustmentID))
	assert.Len(t, entity.Lines, 2)
	assert.False(t, entity.HasExceptions)
	assert.Equal(t, int64(145000), entity.NetPayMinor)
}

func TestCloneSettlement_LeavesTheOriginalUntouched(t *testing.T) {
	t.Parallel()

	entity := planFixture(driversettlement.StatusDraft)
	clone := CloneSettlement(entity)
	require.NoError(t, PlanAddAdjustment(clone, &AdjustmentLineInput{
		Description: "Bonus",
		AmountMinor: 1000,
	}))
	clone.Lines[0].Description = "Changed"

	assert.Len(t, entity.Lines, 2)
	assert.Equal(t, "Linehaul", entity.Lines[0].Description)
	assert.Empty(t, entity.Exceptions)
}

func TestMergeRebuilt_KeepsManualAdjustments(t *testing.T) {
	t.Parallel()

	entity := planFixture(driversettlement.StatusDraft)
	require.NoError(t, PlanAddAdjustment(entity, &AdjustmentLineInput{
		Description: "Bonus",
		AmountMinor: 1000,
	}))
	rebuilt := &driversettlement.Settlement{
		CurrencyCode:  "USD",
		ShipmentCount: 3,
		Lines: []*driversettlement.SettlementLine{{
			Category:    driversettlement.LineCategoryEarning,
			Description: "Linehaul",
			AmountMinor: 200000,
		}},
	}

	MergeRebuilt(entity, rebuilt)

	require.Len(t, entity.Lines, 2)
	assert.Equal(t, driversettlement.LineCategoryAdjustment, entity.Lines[1].Category)
	assert.True(t, entity.Lines[1].ID.IsNil())
	assert.Equal(t, 3, entity.ShipmentCount)
	assert.Equal(t, int64(201000), entity.NetPayMinor)
	assert.True(t, entity.HasExceptions)
}
