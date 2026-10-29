package turnlimits

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentbudgetports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Limits struct {
	Budgets       agentbudgetports.AgentBudgetService
	Conversations conversationports.ConversationRepository
	AgentControls tenantports.AgentControlRepository
	Quota         platformplanports.QuotaGuard
}

// PersonAllowance is how many questions one person has asked this month
// against what their organization allows. Nil when nobody is limited.
func (s *Limits) PersonAllowance(
	ctx context.Context,
	userID pulid.ID,
	tenant pagination.TenantInfo,
	now time.Time,
) (*assistantports.PersonAllowance, error) {
	if s.AgentControls == nil || userID.IsNil() {
		return nil, nil //nolint:nilnil // a nil allowance is the contract for nobody being limited
	}
	control, err := s.AgentControls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if control.PersonMonthlyMessages <= 0 {
		return nil, nil //nolint:nilnil // a nil allowance is the contract for nobody being limited
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	used, err := s.Conversations.CountQuestionsSince(
		ctx,
		conversationports.CountQuestionsSinceRequest{
			TenantInfo: tenant,
			UserID:     userID,
			Since:      monthStart.Unix(),
		},
	)
	if err != nil {
		return nil, err
	}

	return &assistantports.PersonAllowance{
		Used:     used,
		Limit:    control.PersonMonthlyMessages,
		ResetsAt: monthStart.AddDate(0, 1, 0).Unix(),
	}, nil
}

// AssertWithinAllowance refuses a question once the person has asked as many
// as their organization allows this month, saying how many and when the
// allowance refreshes.
func (s *Limits) AssertWithinAllowance(
	ctx context.Context,
	userID pulid.ID,
	tenant pagination.TenantInfo,
) error {
	allowance, err := s.PersonAllowance(ctx, userID, tenant, time.Now().UTC())
	if err != nil || allowance == nil || allowance.Used < allowance.Limit {
		return err
	}

	refused := errortypes.NewBusinessError(
		"You've used your AI allowance for this month ({0} of {1} questions). It refreshes on the first of next month.",
		allowance.Used,
		allowance.Limit,
	)
	refused.Params = map[string]string{
		"code":     "person_allowance",
		"used":     strconv.Itoa(allowance.Used),
		"limit":    strconv.Itoa(allowance.Limit),
		"resetsAt": strconv.FormatInt(allowance.ResetsAt, 10),
	}

	return refused
}

func (s *Limits) AssertWithinPlan(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := quotaservice.Preflight(ctx, s.Quota, &platformplanports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}); err != nil {
		return err
	}

	return quotaservice.Preflight(ctx, s.Quota, &platformplanports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	})
}
