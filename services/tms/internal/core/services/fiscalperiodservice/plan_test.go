package fiscalperiodservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanTransitionProjectsWhatEachTransitionWrites(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		transition Transition
		from       fiscalperiod.Status
		want       fiscalperiod.Status
		check      func(t *testing.T, f *transitionFixture, after *fiscalperiod.FiscalPeriod)
	}{
		{
			name:       "close",
			transition: TransitionClose,
			from:       fiscalperiod.StatusLocked,
			want:       fiscalperiod.StatusClosed,
			check: func(t *testing.T, f *transitionFixture, after *fiscalperiod.FiscalPeriod) {
				t.Helper()
				assert.Equal(t, f.userID, after.ClosedByID)
				require.NotNil(t, after.ClosedAt)
			},
		},
		{
			name:       "lock",
			transition: TransitionLock,
			from:       fiscalperiod.StatusOpen,
			want:       fiscalperiod.StatusLocked,
			check: func(t *testing.T, f *transitionFixture, after *fiscalperiod.FiscalPeriod) {
				t.Helper()
				assert.Equal(t, f.userID, after.LockedByID)
			},
		},
		{
			name:       "unlock",
			transition: TransitionUnlock,
			from:       fiscalperiod.StatusLocked,
			want:       fiscalperiod.StatusOpen,
			check: func(t *testing.T, _ *transitionFixture, after *fiscalperiod.FiscalPeriod) {
				t.Helper()
				assert.Nil(t, after.LockedAt)
			},
		},
		{
			name:       "reopen",
			transition: TransitionReopen,
			from:       fiscalperiod.StatusClosed,
			want:       fiscalperiod.StatusOpen,
			check: func(t *testing.T, f *transitionFixture, after *fiscalperiod.FiscalPeriod) {
				t.Helper()
				assert.Equal(t, "Late vendor invoice", after.ReopenReason)
				assert.Equal(t, f.userID, after.ReopenedByID)
				assert.Nil(t, after.ClosedAt)
			},
		},
		{
			name:       "activate",
			transition: TransitionActivate,
			from:       fiscalperiod.StatusInactive,
			want:       fiscalperiod.StatusOpen,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newTransitionFixture(t, fiscalyear.StatusOpen)
			target := f.period(1, tc.from)
			f.expectState(target)

			plan, err := f.svc.PlanTransition(t.Context(), &TransitionRequest{
				Transition: tc.transition,
				ID:         target.ID,
				TenantInfo: f.tenant,
				Reason:     "  Late vendor invoice ",
			}, f.userID)

			require.NoError(t, err)
			assert.Equal(t, tc.from, plan.Before.Status)
			assert.Equal(t, tc.want, plan.After.Status)
			if tc.check != nil {
				tc.check(t, f, plan.After)
			}
		})
	}
}

func TestPlanTransitionRefusesWhatTheTransitionWould(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)
	target := f.period(1, fiscalperiod.StatusClosed)
	f.expectState(target)

	_, err := f.svc.PlanTransition(t.Context(), &TransitionRequest{
		Transition: TransitionLock,
		ID:         target.ID,
		TenantInfo: f.tenant,
	}, f.userID)

	requireFieldError(t, err, "status", "Only Open fiscal periods can be locked")
}

func TestPlanTransitionRefusesAnUnknownTransition(t *testing.T) {
	t.Parallel()

	f := newTransitionFixture(t, fiscalyear.StatusOpen)

	_, err := f.svc.PlanTransition(t.Context(), &TransitionRequest{
		Transition: Transition("Delete"),
		ID:         f.period(1, fiscalperiod.StatusOpen).ID,
		TenantInfo: f.tenant,
	}, f.userID)

	require.ErrorContains(t, err, "closed, reopened, locked, unlocked or opened")
}
