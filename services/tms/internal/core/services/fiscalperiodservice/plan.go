package fiscalperiodservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Transition string

const (
	TransitionClose    = Transition("Close")
	TransitionReopen   = Transition("Reopen")
	TransitionLock     = Transition("Lock")
	TransitionUnlock   = Transition("Unlock")
	TransitionActivate = Transition("Activate")
)

type TransitionRequest struct {
	Transition Transition
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Reason     string
}

type TransitionPlan struct {
	Before *fiscalperiod.FiscalPeriod
	After  *fiscalperiod.FiscalPeriod
}

func (s *Service) rulesFor(req *TransitionRequest) (transition, error) {
	switch req.Transition {
	case TransitionClose:
		return s.closeTransition(), nil
	case TransitionReopen:
		return reopenTransition(req.Reason), nil
	case TransitionLock:
		return lockTransition(), nil
	case TransitionUnlock:
		return unlockTransition(), nil
	case TransitionActivate:
		return activateTransition(), nil
	default:
		return transition{}, errortypes.NewValidationError(
			"transition",
			errortypes.ErrInvalid,
			"A fiscal period is closed, reopened, locked, unlocked or opened",
		)
	}
}

func (s *Service) PlanTransition(
	ctx context.Context,
	req *TransitionRequest,
	userID pulid.ID,
) (*TransitionPlan, error) {
	rules, err := s.rulesFor(req)
	if err != nil {
		return nil, err
	}

	state, err := s.loadTransitionState(ctx, req.ID, req.TenantInfo, false)
	if err != nil {
		return nil, err
	}
	if err = rules.validate(ctx, state); err != nil {
		return nil, err
	}

	after := *state.period
	rules.project(&after, userID, timeutils.NowUnix())

	return &TransitionPlan{Before: state.period, After: &after}, nil
}

func (s *Service) Transition(
	ctx context.Context,
	req *TransitionRequest,
	userID pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	switch req.Transition {
	case TransitionClose:
		return s.Close(ctx, repositories.CloseFiscalPeriodRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		}, userID)
	case TransitionReopen:
		return s.Reopen(ctx, repositories.ReopenFiscalPeriodRequest{
			ID:           req.ID,
			TenantInfo:   req.TenantInfo,
			ReopenReason: req.Reason,
		}, userID)
	case TransitionLock:
		return s.Lock(ctx, repositories.LockFiscalPeriodRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		}, userID)
	case TransitionUnlock:
		return s.Unlock(ctx, repositories.UnlockFiscalPeriodRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		}, userID)
	case TransitionActivate:
		return s.Activate(ctx, repositories.ActivateFiscalPeriodRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		}, userID)
	default:
		_, err := s.rulesFor(req)
		return nil, err
	}
}
