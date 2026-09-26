package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramFiscalPeriodID    = "fiscalPeriodId"
	maxFiscalReopenReason  = 1000
	fiscalPeriodSupplier   = "The fiscal period, from list_fiscal_periods. Never guess one."
	fiscalPeriodRecordKind = "fiscal period"
)

var fiscalPeriodFields = []string{
	fieldStatus,
	"lockedAt",
	"closedAt",
	"reopenedAt",
	"reopenReason",
}

var fiscalPeriodDates = map[string]assistantartifact.DisplayType{
	"lockedAt":   assistantartifact.DisplayDate,
	"closedAt":   assistantartifact.DisplayDate,
	"reopenedAt": assistantartifact.DisplayDate,
}

type fiscalPeriodKeeper interface {
	PlanTransition(
		ctx context.Context,
		req *fiscalperiodservice.TransitionRequest,
		userID pulid.ID,
	) (*fiscalperiodservice.TransitionPlan, error)
	Transition(
		ctx context.Context,
		req *fiscalperiodservice.TransitionRequest,
		userID pulid.ID,
	) (*fiscalperiod.FiscalPeriod, error)
}

var _ fiscalPeriodKeeper = (*fiscalperiodservice.Service)(nil)

type fiscalPeriodMove struct {
	name        string
	transition  fiscalperiodservice.Transition
	operation   permission.Operation
	description string
	rationale   string
	outcome     string
	needsReason bool
}

func closeFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "close_fiscal_period",
		transition: fiscalperiodservice.TransitionClose,
		operation:  permission.OpClose,
		description: "Propose closing a fiscal period once get_fiscal_close_blockers shows " +
			"nothing in the way, so nothing more posts in it. Earlier periods must be " +
			"closed first. A person always decides, and reopening it takes a reason.",
		rationale: "Ends posting to a period of the books a person signs off; only a " +
			"person closes one.",
		outcome: "nothing more posts in it until someone reopens it",
	}
}

func lockFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "lock_fiscal_period",
		transition: fiscalperiodservice.TransitionLock,
		operation:  permission.OpLock,
		description: "Propose locking an open fiscal period while its books are " +
			"reviewed for close, the step before close_fiscal_period. " +
			"unlock_fiscal_period undoes it. A person always decides.",
		rationale: "Changes which dates the ledger takes postings for; only a person " +
			"locks a period.",
		outcome: "it is locked for review ahead of its close",
	}
}

func unlockFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "unlock_fiscal_period",
		transition: fiscalperiodservice.TransitionUnlock,
		operation:  permission.OpUnlock,
		description: "Propose unlocking a locked fiscal period so it is open again, when " +
			"postings must still land in it before it closes. A person always decides.",
		rationale: "Changes which dates the ledger takes postings for; only a person " +
			"unlocks a period.",
		outcome: "it is open for posting again",
	}
}

func reopenFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "reopen_fiscal_period",
		transition: fiscalperiodservice.TransitionReopen,
		operation:  permission.OpReopen,
		description: "Propose reopening a closed fiscal period, with the reason, so a " +
			"correction can post in it. Later periods must be reopened first and a " +
			"closed fiscal year never is. A person always decides.",
		rationale: "Lets postings land again in books a person closed; only a person " +
			"reopens a period.",
		outcome:     "postings land in it again until it is closed again",
		needsReason: true,
	}
}

func openFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "open_fiscal_period",
		transition: fiscalperiodservice.TransitionActivate,
		operation:  permission.OpActivate,
		description: "Propose opening an inactive fiscal period so postings dated in it " +
			"are accepted. Periods open in order, never in a closed fiscal year. A person " +
			"always decides.",
		rationale: "Changes which dates the ledger takes postings for; only a person " +
			"opens a period.",
		outcome: "postings dated in it are accepted",
	}
}

func (m *fiscalPeriodMove) properties() map[string]any {
	properties := map[string]any{
		paramFiscalPeriodID: stringProperty(fiscalPeriodSupplier, 0),
	}
	if m.needsReason {
		properties[paramReason] = stringProperty("Why the period must take postings again: "+
			"the correction and the evidence for it.", maxFiscalReopenReason)
	}

	return properties
}

func (m *fiscalPeriodMove) required() []string {
	if m.needsReason {
		return []string{paramFiscalPeriodID, paramReason}
	}

	return []string{paramFiscalPeriodID}
}

func (m *fiscalPeriodMove) request(
	params *serviceports.ToolExecuteParams,
) (*fiscalperiodservice.TransitionRequest, error) {
	id, err := requirePulid(params.Params, paramFiscalPeriodID)
	if err != nil {
		return nil, err
	}
	req := &fiscalperiodservice.TransitionRequest{
		Transition: m.transition,
		ID:         id,
		TenantInfo: tenantFrom(*params),
	}
	if m.needsReason {
		if req.Reason, err = requireBoundedText(
			params.Params,
			paramReason,
			maxFiscalReopenReason,
		); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (m *fiscalPeriodMove) render(
	_ *fiscalperiodservice.TransitionRequest,
	plan *fiscalperiodservice.TransitionPlan,
) (*agent.ToolPreview, error) {
	before := plan.Before
	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceFiscalPeriod,
			ID:       before.ID,
			Label:    before.Name,
			Version:  pinnedVersion(before.Version),
		},
		before,
		plan.After,
		toolpreview.Only(fiscalPeriodFields...),
		toolpreview.Volatile("lockedAt", "closedAt", "reopenedAt"),
		toolpreview.Types(fiscalPeriodDates),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would %s %s %s (%s to %s, now %s); %s.",
		fiscalMoveVerb(m.transition),
		fiscalPeriodRecordKind,
		before.Name,
		dayLabel(before.StartDate),
		dayLabel(before.EndDate),
		before.Status,
		m.outcome,
	), change), nil
}

func fiscalMoveVerb(transition fiscalperiodservice.Transition) string {
	switch transition {
	case fiscalperiodservice.TransitionClose:
		return "close"
	case fiscalperiodservice.TransitionReopen:
		return "reopen"
	case fiscalperiodservice.TransitionLock:
		return "lock"
	case fiscalperiodservice.TransitionUnlock:
		return "unlock"
	case fiscalperiodservice.TransitionActivate:
		return "open"
	default:
		return string(transition)
	}
}

func newFiscalPeriodTool(
	periods fiscalPeriodKeeper,
	move *fiscalPeriodMove,
) serviceports.AgentTool {
	return newReceivableTool(ledgerMoneySpec(&receivableSpec{
		name:        move.name,
		description: move.description,
		resource:    permission.ResourceFiscalPeriod,
		operation:   move.operation,
		rationale:   move.rationale,
		properties:  move.properties(),
		required:    move.required(),
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramFiscalPeriodID, permission.ResourceFiscalPeriod)
		},
	}), receivablePlan[*fiscalperiodservice.TransitionRequest, *fiscalperiodservice.TransitionPlan]{
		request: move.request,
		plan: func(
			ctx context.Context,
			req *fiscalperiodservice.TransitionRequest,
			params *serviceports.ToolExecuteParams,
		) (*fiscalperiodservice.TransitionPlan, error) {
			return periods.PlanTransition(ctx, req, params.Actor.UserID)
		},
		refused: func(*fiscalperiodservice.TransitionRequest) string {
			return fmt.Sprintf("Would %s a %s.", fiscalMoveVerb(move.transition),
				fiscalPeriodRecordKind)
		},
		render: move.render,
		run: func(
			ctx context.Context,
			req *fiscalperiodservice.TransitionRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := periods.Transition(ctx, req, params.Actor.UserID)

			return nil, err
		},
	})
}

func provideCloseFiscalPeriodTool(periods *fiscalperiodservice.Service) serviceports.AgentTool {
	return newFiscalPeriodTool(periods, closeFiscalPeriodMove())
}

func provideLockFiscalPeriodTool(periods *fiscalperiodservice.Service) serviceports.AgentTool {
	return newFiscalPeriodTool(periods, lockFiscalPeriodMove())
}

func provideUnlockFiscalPeriodTool(periods *fiscalperiodservice.Service) serviceports.AgentTool {
	return newFiscalPeriodTool(periods, unlockFiscalPeriodMove())
}

func provideReopenFiscalPeriodTool(periods *fiscalperiodservice.Service) serviceports.AgentTool {
	return newFiscalPeriodTool(periods, reopenFiscalPeriodMove())
}

func provideOpenFiscalPeriodTool(periods *fiscalperiodservice.Service) serviceports.AgentTool {
	return newFiscalPeriodTool(periods, openFiscalPeriodMove())
}
