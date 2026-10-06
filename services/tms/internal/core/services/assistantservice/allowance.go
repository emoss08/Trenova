package assistantservice

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// personAllowance is how many questions one person has asked this month
// against what their organization allows. Nil when nobody is limited.
func (s *Service) personAllowance(
	ctx context.Context,
	userID pulid.ID,
	tenant pagination.TenantInfo,
	now time.Time,
) (*services.PersonAllowance, error) {
	if s.agentControls == nil || userID.IsNil() {
		return nil, nil
	}
	control, err := s.agentControls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if control.PersonMonthlyMessages <= 0 {
		return nil, nil
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	used, err := s.conversations.CountQuestionsSince(ctx, repositories.CountQuestionsSinceRequest{
		TenantInfo: tenant,
		UserID:     userID,
		Since:      monthStart.Unix(),
	})
	if err != nil {
		return nil, err
	}

	return &services.PersonAllowance{
		Used:     used,
		Limit:    control.PersonMonthlyMessages,
		ResetsAt: monthStart.AddDate(0, 1, 0).Unix(),
	}, nil
}

// assertWithinAllowance refuses a question once the person has asked as many
// as their organization allows this month, saying how many and when the
// allowance refreshes.
func (s *Service) assertWithinAllowance(
	ctx context.Context,
	userID pulid.ID,
	tenant pagination.TenantInfo,
) error {
	allowance, err := s.personAllowance(ctx, userID, tenant, time.Now().UTC())
	if err != nil || allowance == nil || allowance.Used < allowance.Limit {
		return err
	}

	refused := errortypes.NewBusinessError(
		"You've used your AI allowance for this month ({0} of {1} questions). It refreshes on the first of next month.",
		allowance.Used, allowance.Limit,
	)
	refused.Params = map[string]string{
		"code":     "person_allowance",
		"used":     strconv.Itoa(allowance.Used),
		"limit":    strconv.Itoa(allowance.Limit),
		"resetsAt": strconv.FormatInt(allowance.ResetsAt, 10),
	}

	return refused
}
