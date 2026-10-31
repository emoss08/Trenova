package turnlimits

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type controls struct {
	tenantports.AgentControlRepository

	monthly int
}

func (c controls) GetOrCreate(context.Context, pagination.TenantInfo) (*agentcontrol.AgentControl, error) {
	return &agentcontrol.AgentControl{PersonMonthlyMessages: c.monthly}, nil
}

type deskQuestions struct {
	conversationports.ConversationRepository

	count int
}

func (d deskQuestions) CountQuestionsSince(
	context.Context,
	conversationports.CountQuestionsSinceRequest,
) (int, error) {
	return d.count, nil
}

type paletteQuestions struct {
	count  int
	asked  pagination.TenantInfo
	since  int64
	called bool
}

func (p *paletteQuestions) CountUnkeptSince(
	_ context.Context,
	tenant pagination.TenantInfo,
	since int64,
) (int, error) {
	p.called = true
	p.asked = tenant
	p.since = since

	return p.count, nil
}

func TestPersonAllowance_CountsPaletteQuestionsBesideDeskMessages(t *testing.T) {
	t.Parallel()

	palette := &paletteQuestions{count: 4}
	limits := &Limits{
		Conversations:    deskQuestions{count: 6},
		AgentControls:    controls{monthly: 20},
		PaletteQuestions: palette,
	}
	userID := pulid.MustNew("usr_")
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

	allowance, err := limits.PersonAllowance(t.Context(), userID, plantest.Tenant(), now)
	require.NoError(t, err)

	assert.Equal(t, 10, allowance.Used)
	assert.Equal(t, 20, allowance.Limit)
	assert.Equal(t, userID, palette.asked.UserID, "the person's own questions are counted")
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix(), palette.since)
}

func TestAssertMayAsk_RefusesOnceTheMonthsQuestionsAreUsed(t *testing.T) {
	t.Parallel()

	limits := &Limits{
		Conversations:    deskQuestions{count: 15},
		AgentControls:    controls{monthly: 20},
		PaletteQuestions: &paletteQuestions{count: 5},
	}

	err := limits.AssertMayAsk(t.Context(), pulid.MustNew("usr_"), plantest.Tenant())

	var refused *errortypes.BusinessError
	require.ErrorAs(t, err, &refused)
	assert.Equal(t, "person_allowance", refused.Params["code"])
	assert.Equal(t, "20", refused.Params["used"])
}
