package assistantservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
)

// budgetNearShare is the share of a month's budget past which the Desk warns
// that it is nearly spent.
const budgetNearShare = 0.9

// ThreadBudget says where a conversation's agent stands against its caps, so
// the Desk can warn before a cap stops a reply rather than only after. Reading
// the thread first scopes it to the person: someone else's conversation is
// not found.
func (s *Service) ThreadBudget(
	ctx context.Context,
	req repositories.GetThreadRequest,
) (*services.ThreadBudget, error) {
	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil {
		return nil, err
	}
	out := &services.ThreadBudget{}
	if out.Person, err = s.personAllowance(ctx, req.UserID, req.TenantInfo, time.Now().UTC()); err != nil {
		return nil, err
	}
	if s.budgets == nil {
		return out, nil
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	status, err := s.budgets.Status(ctx, definition)
	if err != nil {
		return nil, err
	}

	out.AgentName = definition.Name
	out.SpentUSD = status.SpentUSD
	out.MonthStart = status.MonthStart
	out.ResetsAt = time.Unix(status.MonthStart, 0).UTC().AddDate(0, 1, 0).Unix()
	out.RunsToday = status.RunsToday
	out.DailyRunLimit = status.DailyRunLimit
	if status.MonthlyBudget != nil {
		out.LimitUSD = *status.MonthlyBudget
		out.Share = budgetShare(status.SpentUSD, *status.MonthlyBudget)
		out.Near = out.Share >= budgetNearShare
	}

	return out, nil
}

// budgetShare is spent over limit, nought when either does not parse or the
// limit is nought.
func budgetShare(spent, limit string) float64 {
	spentUSD, err := decimal.NewFromString(spent)
	if err != nil {
		return 0
	}
	limitUSD, err := decimal.NewFromString(limit)
	if err != nil || !limitUSD.IsPositive() {
		return 0
	}

	share, _ := spentUSD.Div(limitUSD).Float64()
	return share
}
