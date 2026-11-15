package assistantturnservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartRefusesAPersonsQuestionPastTheMonthlyMessageLimit(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Check(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   1,
	}).Return(&platformplanports.QuotaDecision{
		Meter:   platformcatalog.MeterAIAssistantMessages,
		Plan:    platformplan.PlanKeyFreeDemo,
		Allowed: false,
		Limit:   25,
		Used:    25,
	}, nil)
	svc := &Service{l: zap.NewNop(), quota: guard, features: plantest.AllFeatures()}

	for _, origin := range []conversationturn.AssistantTurnOrigin{"", conversationturn.AssistantTurnOriginPerson} {
		turn, err := svc.Start(t.Context(), StartRequest{
			ThreadID:   pulid.MustNew("thr_"),
			UserID:     tenant.UserID,
			TenantInfo: tenant,
			Origin:     origin,
		})
		require.Nil(t, turn)
		require.True(t, errortypes.IsQuotaExceededError(err))
	}
}

func TestStartRefusesScheduledTurnsThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:        zap.NewNop(),
		features: plantest.Without(platformcatalog.FeatureAgentAutomation),
	}

	turn, err := svc.Start(t.Context(), StartRequest{
		ThreadID:   pulid.MustNew("thr_"),
		TenantInfo: plantest.Tenant(),
		Origin:     conversationturn.AssistantTurnOriginScheduled,
	})

	require.Nil(t, turn)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestAssertPlanAllowsLetsFollowUpsThrough(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:        zap.NewNop(),
		quota:    platformplanportsmock.NewMockQuotaGuard(t),
		features: plantest.AllFeatures(),
	}

	require.NoError(t, svc.assertPlanAllows(t.Context(), StartRequest{
		TenantInfo: plantest.Tenant(),
		Origin:     conversationturn.AssistantTurnOriginDecisionFollowUp,
	}))
}

func TestStartRefusesEveryTurnWithoutTheDesk(t *testing.T) {
	t.Parallel()

	svc := &Service{
		l:        zap.NewNop(),
		quota:    platformplanportsmock.NewMockQuotaGuard(t),
		features: plantest.BaseOnly(),
	}

	for _, origin := range []conversationturn.AssistantTurnOrigin{
		"",
		conversationturn.AssistantTurnOriginPerson,
		conversationturn.AssistantTurnOriginDecisionFollowUp,
		conversationturn.AssistantTurnOriginScheduled,
		conversationturn.AssistantTurnOriginCompaction,
		conversationturn.AssistantTurnOriginWaitResolved,
	} {
		turn, err := svc.Start(t.Context(), StartRequest{
			ThreadID:   pulid.MustNew("thr_"),
			TenantInfo: plantest.Tenant(),
			Origin:     origin,
		})

		require.Nil(t, turn)
		restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
		require.Truef(t, ok, "%q: expected a plan restriction, got %v", origin, err)
		require.Equal(t, "Trenova AI", restriction.AddOn)
	}
}
