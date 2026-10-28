package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramFiscalPeriodID = "fiscalPeriodId"

const maxFiscalReopenReason = 1000

const fiscalPeriodSupplier = "The fiscal period, from ledger_list_fiscal_periods. Never guess one."

const fiscalPeriodRecordKind = "fiscal period"

var fiscalPeriodFields = []string{
	common.FieldStatus,
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

type FiscalPeriodKeeper interface {
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

type fiscalPeriodMove struct {
	name        string
	recipe      []string
	transition  fiscalperiodservice.Transition
	operation   permission.Operation
	description string
	rationale   string
	outcome     string
	needsReason bool
}

func closeFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name: "ledger_close_fiscal_period",
		recipe: []string{
			"ledger_list_fiscal_periods",
			"ledger_get_fiscal_close_blockers",
			"ledger_close_fiscal_period",
		},
		transition: fiscalperiodservice.TransitionClose,
		operation:  permission.OpClose,
		description: "Propose closing a fiscal period once ledger_get_fiscal_close_blockers shows " +
			"nothing in the way, so nothing more posts in it. Earlier periods must be " +
			"closed first. A person always decides, and reopening it takes a reason.",
		rationale: "Ends posting to a period of the books a person signs off; only a " +
			"person closes one.",
		outcome: "nothing more posts in it until someone reopens it",
	}
}

func lockFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name: "ledger_lock_fiscal_period",
		recipe: []string{
			"ledger_list_fiscal_periods",
			"ledger_get_fiscal_close_blockers",
			"ledger_lock_fiscal_period",
		},
		transition: fiscalperiodservice.TransitionLock,
		operation:  permission.OpLock,
		description: "Propose locking an open fiscal period while its books are " +
			"reviewed for close, the step before ledger_close_fiscal_period. " +
			"ledger_unlock_fiscal_period undoes it. A person always decides.",
		rationale: "Changes which dates the ledger takes postings for; only a person " +
			"locks a period.",
		outcome: "it is locked for review ahead of its close",
	}
}

func unlockFiscalPeriodMove() *fiscalPeriodMove {
	return &fiscalPeriodMove{
		name:       "ledger_unlock_fiscal_period",
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
		name:       "ledger_reopen_fiscal_period",
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
		name:       "ledger_open_fiscal_period",
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
		paramFiscalPeriodID: agenttoolschema.RecordIDText(permission.ResourceFiscalPeriod,
			fiscalPeriodSupplier),
	}
	if m.needsReason {
		properties[common.ParamReason] = common.StringProperty(
			"Why the period must take postings again: "+
				"the correction and the evidence for it.",
			maxFiscalReopenReason,
		)
	}

	return properties
}

func (m *fiscalPeriodMove) required() []string {
	if m.needsReason {
		return []string{paramFiscalPeriodID, common.ParamReason}
	}

	return []string{paramFiscalPeriodID}
}

func (m *fiscalPeriodMove) request(
	params *agenttoolports.ToolExecuteParams,
) (*fiscalperiodservice.TransitionRequest, error) {
	id, err := common.RequirePulid(params.Params, paramFiscalPeriodID)
	if err != nil {
		return nil, err
	}
	req := &fiscalperiodservice.TransitionRequest{
		Transition: m.transition,
		ID:         id,
		TenantInfo: common.TenantFrom(*params),
	}
	if m.needsReason {
		if req.Reason, err = common.RequireBoundedText(
			params.Params,
			common.ParamReason,
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
			Version:  common.PinnedVersion(before.Version),
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
		common.DayLabel(before.StartDate),
		common.DayLabel(before.EndDate),
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
	periods FiscalPeriodKeeper,
	move *fiscalPeriodMove,
) agenttoolports.AgentTool {
	return common.NewReceivableTool(common.LedgerMoneySpec(&common.ReceivableSpec{
		Name:        move.name,
		Recipe:      move.recipe,
		Description: move.description,
		Resource:    permission.ResourceFiscalPeriod,
		Operation:   move.operation,
		Rationale:   move.rationale,
		Properties:  move.properties(),
		Required:    move.required(),
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return common.TargetOf(params, paramFiscalPeriodID, permission.ResourceFiscalPeriod)
		},
	}), common.ReceivablePlan[*fiscalperiodservice.TransitionRequest, *fiscalperiodservice.TransitionPlan]{
		Request: move.request,
		Plan: func(
			ctx context.Context,
			req *fiscalperiodservice.TransitionRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*fiscalperiodservice.TransitionPlan, error) {
			return periods.PlanTransition(ctx, req, params.Actor.UserID)
		},
		Refused: func(*fiscalperiodservice.TransitionRequest) string {
			return fmt.Sprintf("Would %s a %s.", fiscalMoveVerb(move.transition),
				fiscalPeriodRecordKind)
		},
		Render: move.render,
		Run: func(
			ctx context.Context,
			req *fiscalperiodservice.TransitionRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := periods.Transition(ctx, req, params.Actor.UserID)

			return nil, err
		},
	})
}

func ProvideCloseFiscalPeriodTool(periods *fiscalperiodservice.Service) agenttoolports.AgentTool {
	return newFiscalPeriodTool(periods, closeFiscalPeriodMove())
}

func ProvideLockFiscalPeriodTool(periods *fiscalperiodservice.Service) agenttoolports.AgentTool {
	return newFiscalPeriodTool(periods, lockFiscalPeriodMove())
}

func ProvideUnlockFiscalPeriodTool(periods *fiscalperiodservice.Service) agenttoolports.AgentTool {
	return newFiscalPeriodTool(periods, unlockFiscalPeriodMove())
}

func ProvideReopenFiscalPeriodTool(periods *fiscalperiodservice.Service) agenttoolports.AgentTool {
	return newFiscalPeriodTool(periods, reopenFiscalPeriodMove())
}

func ProvideOpenFiscalPeriodTool(periods *fiscalperiodservice.Service) agenttoolports.AgentTool {
	return newFiscalPeriodTool(periods, openFiscalPeriodMove())
}
