package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramPTOType      = "type"
	paramPTOStart     = wfFieldStartDate
	paramPTOEnd       = wfFieldEndDate
	paramAmountDays   = "amountDays"
	paramEffectiveOn  = "effectiveDate"
	kindTimeOff       = "time off request"
	kindBalance       = "time off balance"
	maxPTOReasonChars = 255
)

var ptoFields = []string{
	wfFieldWorkerID, paramPTOType, fieldStatus, paramPTOStart, paramPTOEnd, fieldReason,
	"days", "autoApproved", wfFieldApproverID,
}

type ptoRequester interface {
	Get(ctx context.Context, req *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error)
	PlanCreate(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
	Create(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
	PlanUpdate(
		ctx context.Context,
		entity *worker.WorkerPTO,
	) (*serviceports.RecordChange[worker.WorkerPTO], error)
	Update(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
}

type ptoBalanceAdjuster interface {
	PlanAdjust(
		ctx context.Context,
		req *ptoledgerservice.AdjustRequest,
	) (*ptoledgerservice.AdjustPlan, error)
	Adjust(
		ctx context.Context,
		req *ptoledgerservice.AdjustRequest,
	) (*worker.WorkerPTOLedgerEntry, error)
}

var _ ptoBalanceAdjuster = (*ptoledgerservice.Service)(nil)

func ptoToolProviders() []any {
	return []any{
		provideRequestWorkerPTOTool,
		provideUpdateWorkerPTOTool,
		provideAdjustWorkerPTOBalanceTool,
	}
}

func ptoRecord(pto *worker.WorkerPTO) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerPTO, pto.ID,
		fmt.Sprintf("%s from %s", pto.Type, dayText(pto.StartDate)), pto.Version)
}

func ptoDatesProperties() map[string]any {
	return map[string]any{
		paramPTOType:  agenttoolschema.Enum("The kind of time off.", agenttoolschema.PTOTypes),
		paramPTOStart: dayProperty("The first day off."),
		paramPTOEnd:   dayProperty("The last day off."),
		fieldReason: stringProperty("The reason the worker gave. The driver sees it with "+
			"the request.", maxPTOReasonChars),
	}
}

func applyPTODates(pto *worker.WorkerPTO, params map[string]any, required bool) error {
	if kind, given, err := optionalEnum(params, paramPTOType,
		agenttoolschema.PTOTypes.Values); err != nil {
		return err
	} else if given {
		pto.Type = kind
	} else if required {
		return fmt.Errorf("missing required parameter %q", paramPTOType)
	}
	for _, field := range []struct {
		key  string
		dest *int64
	}{
		{paramPTOStart, &pto.StartDate},
		{paramPTOEnd, &pto.EndDate},
	} {
		day, err := optionalScheduleDay(params, field.key)
		switch {
		case err != nil:
			return err
		case day != nil:
			*field.dest = *day
		case required:
			return fmt.Errorf("missing required parameter %q", field.key)
		}
	}
	reason, err := optionalBoundedText(params, fieldReason, maxPTOReasonChars)
	switch {
	case err != nil:
		return err
	case reason != nil:
		pto.Reason = *reason
	case required:
		return fmt.Errorf("missing required parameter %q", fieldReason)
	}
	return nil
}

func newRequestWorkerPTOTool(pto ptoRequester) serviceports.AgentTool {
	properties := ptoDatesProperties()
	properties[paramWorkerID] = workerProperty()
	spec := withSchema(wfSpec(
		"request_worker_pto",
		"File a time-off request for a worker who asked for it, over the days given. It "+
			"counts the days against their policy and refuses days that overlap time off "+
			"already booked, a blackout, or more than the balance allows. Under a policy that "+
			"needs no approval it is approved and booked at once and the driver is told; "+
			"otherwise it waits for approve_worker_pto.",
		"Files time off the driver sees in Dash; under a policy with no approval step it is "+
			"booked and the driver told at once, and cancel_worker_pto withdraws it.",
		permission.ResourceWorkerPTO,
		permission.OpCreate,
	), properties, paramWorkerID, paramPTOType, paramPTOStart, paramPTOEnd, fieldReason)
	spec.egress = agent.EgressDriverVisible
	spec.searchTerms = []string{"vacation", "time off", "PTO request", "day off"}

	return newReportingReceivableTool(spec, receivablePlan[*worker.WorkerPTO, *worker.WorkerPTO]{
		request: func(params *serviceports.ToolExecuteParams) (*worker.WorkerPTO, error) {
			workerID, err := requirePulid(params.Params, paramWorkerID)
			if err != nil {
				return nil, err
			}
			entity := &worker.WorkerPTO{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				WorkerID:       workerID,
				Status:         worker.PTOStatusRequested,
			}
			return entity, applyPTODates(entity, params.Params, true)
		},
		plan: func(
			ctx context.Context,
			entity *worker.WorkerPTO,
			params *serviceports.ToolExecuteParams,
		) (*worker.WorkerPTO, error) {
			return pto.PlanCreate(ctx, entity, params.Actor.UserID)
		},
		refused: func(*worker.WorkerPTO) string { return "Would file a time-off request." },
		render: func(_ *worker.WorkerPTO, planned *worker.WorkerPTO) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerPTO, pulid.Nil,
					ptoRecord(planned).Label, 0),
				planned, wfOptions(ptoFields...)...)
			if err != nil {
				return nil, err
			}
			summary := fmt.Sprintf("Would request %s days of %s from %s to %s.",
				planned.Days.String(), planned.Type, dayText(planned.StartDate),
				dayText(planned.EndDate))
			if planned.AutoApproved {
				summary += " The worker's policy needs no approval, so it is booked at once " +
					"and the driver is told."
			}
			return toolpreview.Build(summary, change), nil
		},
		run: func(
			ctx context.Context,
			entity *worker.WorkerPTO,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := pto.Create(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return wfResult("requested", kindTimeOff, "ptoId", created.ID, created.WorkerID), nil
		},
	})
}

type ptoEdit struct {
	id     pulid.ID
	params *serviceports.ToolExecuteParams
}

func (e *ptoEdit) entity(ctx context.Context, pto ptoRequester) (*worker.WorkerPTO, error) {
	current, err := pto.Get(ctx, &repositories.GetPTOByIDRequest{
		ID:         e.id,
		TenantInfo: tenantFrom(*e.params),
	})
	if err != nil {
		return nil, err
	}
	entity := *current
	entity.Worker = nil
	return &entity, applyPTODates(&entity, e.params.Params, false)
}

func newUpdateWorkerPTOTool(pto ptoRequester) serviceports.AgentTool {
	properties := ptoDatesProperties()
	properties["ptoId"] = idProperty(ptoIDNote)
	spec := targeting(withSchema(wfSpec(
		"update_worker_pto",
		"Change the kind, days or reason of a time-off request still waiting for a "+
			"decision. Give only what changes. A decided request is refused; cancel it and "+
			"file another.",
		"Changes a pending request the driver sees in Dash; it is changed again the same way.",
		permission.ResourceWorkerPTO,
		permission.OpUpdate,
	), properties, "ptoId"), "ptoId", permission.ResourceWorkerPTO)
	spec.searchTerms = []string{
		"edit time off request",
		"edit vacation request",
		"change pto request",
	}
	spec.egress = agent.EgressDriverVisible

	return newReportingReceivableTool(spec, receivablePlan[
		*ptoEdit, *serviceports.RecordChange[worker.WorkerPTO],
	]{
		request: func(params *serviceports.ToolExecuteParams) (*ptoEdit, error) {
			id, err := requirePulid(params.Params, "ptoId")
			if err != nil {
				return nil, err
			}
			return &ptoEdit{id: id, params: params}, nil
		},
		plan: func(
			ctx context.Context,
			edit *ptoEdit,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.RecordChange[worker.WorkerPTO], error) {
			entity, err := edit.entity(ctx, pto)
			if err != nil {
				return nil, err
			}
			return pto.PlanUpdate(ctx, entity)
		},
		refused: func(*ptoEdit) string { return "Would change a time-off request." },
		render: func(
			_ *ptoEdit,
			change *serviceports.RecordChange[worker.WorkerPTO],
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(ptoRecord(change.Before), change.Before,
				change.After, wfOptions(ptoFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would change the time-off request.", recorded), nil
		},
		run: func(
			ctx context.Context,
			edit *ptoEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, pto)
			if err != nil {
				return nil, err
			}
			updated, err := pto.Update(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindTimeOff, "ptoId", updated.ID, updated.WorkerID), nil
		},
	})
}

func ptoAdjustmentFrom(
	params *serviceports.ToolExecuteParams,
) (*ptoledgerservice.AdjustRequest, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := requireEnum(params.Params, paramPTOType, agenttoolschema.PTOTypes.Values)
	if err != nil {
		return nil, err
	}
	amount, present, err := optionalDecimal(params.Params, paramAmountDays)
	if err != nil {
		return nil, err
	}
	if !present || amount.IsZero() {
		return nil, fmt.Errorf("parameter %q must be a number of days other than zero",
			paramAmountDays)
	}
	note, err := requireBoundedText(params.Params, fieldNote, wfShortChars)
	if err != nil {
		return nil, err
	}
	effective, err := optionalScheduleDay(params.Params, paramEffectiveOn)
	if err != nil {
		return nil, err
	}
	req := &ptoledgerservice.AdjustRequest{
		TenantInfo: tenantFrom(*params),
		WorkerID:   workerID,
		PTOType:    kind,
		AmountDays: amount,
		Note:       note,
		UserID:     params.Actor.UserID,
	}
	if effective != nil {
		req.EffectiveAt = *effective
	}
	return req, nil
}

func newAdjustWorkerPTOBalanceTool(ledger ptoBalanceAdjuster) serviceports.AgentTool {
	spec := personOnly(withSchema(wfSpec(
		"adjust_worker_pto_balance",
		"Draft a manual adjustment to a worker's time-off balance, in days, for a person to "+
			"approve. Positive days add and negative days take away, with a note saying why, "+
			"such as a balance carried over from a previous system.",
		"Changes how many paid days a worker holds, which the organization owes and may pay "+
			"out, so a person approves it and it runs as them.",
		permission.ResourceWorkerPTO,
		permission.OpManage,
	), map[string]any{
		paramWorkerID: workerProperty(),
		paramPTOType:  agenttoolschema.Enum("The balance to adjust.", agenttoolschema.PTOTypes),
		paramAmountDays: amountProperty("Days to add, or a negative number of days to take " +
			"away, such as 1.5 or -2."),
		paramEffectiveOn: dayProperty("The day it takes effect. Defaults to today."),
		fieldNote:        stringProperty("Why, kept on the ledger. Required.", wfShortChars),
	}, paramWorkerID, paramPTOType, paramAmountDays, fieldNote))
	spec.searchTerms = []string{"balance adjustment", "pto balance"}
	spec.egress = agent.EgressMoney
	spec.reversible = false

	return newReportingReceivableTool(spec, receivablePlan[
		*ptoledgerservice.AdjustRequest, *ptoledgerservice.AdjustPlan,
	]{
		request: ptoAdjustmentFrom,

		plan: func(
			ctx context.Context,
			req *ptoledgerservice.AdjustRequest,
			_ *serviceports.ToolExecuteParams,
		) (*ptoledgerservice.AdjustPlan, error) {
			return ledger.PlanAdjust(ctx, req)
		},
		refused: func(*ptoledgerservice.AdjustRequest) string {
			return "Would adjust a time-off balance."
		},
		render: func(
			req *ptoledgerservice.AdjustRequest,
			plan *ptoledgerservice.AdjustPlan,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerPTO, pulid.Nil,
					fmt.Sprintf("%s balance adjustment", req.PTOType), 0),
				plan.Entry, append(wfOptions(wfFieldWorkerID, "ptoType", "entryType",
					paramAmountDays, "balanceAfterDays", "effectiveAt", fieldNote),
					toolpreview.Volatile("effectiveAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would adjust the %s balance by %s days, from %s to %s.", req.PTOType,
				req.AmountDays.String(), plan.BalanceBefore.String(),
				plan.BalanceAfter.String()), change), nil
		},
		run: func(
			ctx context.Context,
			req *ptoledgerservice.AdjustRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entry, err := ledger.Adjust(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("adjusted", kindBalance, "ledgerEntryId", entry.ID,
				entry.WorkerID), nil
		},
	})
}

func provideRequestWorkerPTOTool(pto serviceports.WorkerPTOService) serviceports.AgentTool {
	return newRequestWorkerPTOTool(pto)
}

func provideUpdateWorkerPTOTool(pto serviceports.WorkerPTOService) serviceports.AgentTool {
	return newUpdateWorkerPTOTool(pto)
}

func provideAdjustWorkerPTOBalanceTool(ledger *ptoledgerservice.Service) serviceports.AgentTool {
	return newAdjustWorkerPTOBalanceTool(ledger)
}
