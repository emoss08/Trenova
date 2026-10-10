package assistantturnservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
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
	svc := &Service{l: zap.NewNop(), quota: guard}

	for _, origin := range []conversation.AssistantTurnOrigin{"", conversation.AssistantTurnOriginPerson} {
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

	svc := &Service{l: zap.NewNop(), plans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}

	turn, err := svc.Start(t.Context(), StartRequest{
		ThreadID:   pulid.MustNew("thr_"),
		TenantInfo: plantest.Tenant(),
		Origin:     conversation.AssistantTurnOriginScheduled,
	})

	require.Nil(t, turn)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestAssertPlanAllowsLetsFollowUpsThrough(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), quota: platformplanportsmock.NewMockQuotaGuard(t), plans: platformplanportsmock.NewMockPlanService(t)}

	require.NoError(t, svc.assertPlanAllows(t.Context(), StartRequest{
		TenantInfo: plantest.Tenant(),
		Origin:     conversation.AssistantTurnOriginDecisionFollowUp,
	}))
}
