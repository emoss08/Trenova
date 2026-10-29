package timeofftools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ptopreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPTOType = "type"

const paramPTOStart = workforcekit.FieldStartDate

const paramPTOEnd = workforcekit.FieldEndDate

const paramAmountDays = "amountDays"

const paramEffectiveOn = "effectiveDate"

const kindTimeOff = "time off request"

const kindBalance = "time off balance"

const maxPTOReasonChars = 255

var ptoFields = []string{
	workforcekit.FieldWorkerID,
	paramPTOType,
	changepreview.FieldStatus,
	paramPTOStart,
	paramPTOEnd,
	changepreview.FieldReason,
	"days",
	"autoApproved",
	workforcekit.FieldApproverID,
}

type ptoRequester interface {
	Get(ctx context.Context, req *workerptoports.GetPTOByIDRequest) (*worker.WorkerPTO, error)
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
	) (*kernelports.RecordChange[worker.WorkerPTO], error)
	Update(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
}

type PtoBalanceAdjuster interface {
	PlanAdjust(
		ctx context.Context,
		req *ptoledgerservice.AdjustRequest,
	) (*ptoledgerservice.AdjustPlan, error)
	Adjust(
		ctx context.Context,
		req *ptoledgerservice.AdjustRequest,
	) (*pto.WorkerPTOLedgerEntry, error)
}

func PtoToolProviders() []any {
	return []any{
		provideRequestWorkerPTOTool,
		provideUpdateWorkerPTOTool,
		provideAdjustWorkerPTOBalanceTool,
	}
}

func ptoRecord(leave *worker.WorkerPTO) toolpreview.Record {
	return workforcekit.Record(
		permission.ResourceWorkerPTO,
		leave.ID,
		fmt.Sprintf(
			"%s from %s",
			leave.Type,
			changepreview.DayText(leave.StartDate),
		),
		leave.Version,
	)
}

func ptoDatesProperties() map[string]any {
	return map[string]any{
		paramPTOType: agenttoolschema.Enum("The kind of time off.", agenttoolschema.PTOTypes),
		paramPTOStart: agenttoolschema.Date("The first day off. \"Next Friday\" is the coming " +
			"Friday, not the one a week later."),
		paramPTOEnd: agenttoolschema.Date("The last day off."),
		changepreview.FieldReason: toolparams.StringProperty(
			"The reason the worker gave, when one was given; the "+
				"driver sees it with the request. Omit it rather than ask, and the kind is used.",
			maxPTOReasonChars,
		),
	}
}

func applyPTODates(leave *worker.WorkerPTO, params map[string]any, required bool) error {
	if kind, given, err := toolparams.OptionalEnum(params, paramPTOType,
		agenttoolschema.PTOTypes.Values); err != nil {
		return err
	} else if given {
		leave.Type = kind
	} else if required {
		return fmt.Errorf("missing required parameter %q", paramPTOType)
	}
	for _, field := range []struct {
		key  string
		dest *int64
	}{
		{paramPTOStart, &leave.StartDate},
		{paramPTOEnd, &leave.EndDate},
	} {
		day, err := toolparams.OptionalScheduleDay(params, field.key)
		switch {
		case err != nil:
			return err
		case day != nil:
			*field.dest = *day
		case required:
			return fmt.Errorf("missing required parameter %q", field.key)
		}
	}
	reason, err := toolparams.OptionalBoundedText(
		params,
		changepreview.FieldReason,
		maxPTOReasonChars,
	)
	switch {
	case err != nil:
		return err
	case reason != nil:
		leave.Reason = *reason
	case required:
		leave.Reason = string(leave.Type)
	}
	return nil
}

func newRequestWorkerPTOTool(requests ptoRequester) agenttoolports.AgentTool {
	properties := ptoDatesProperties()
	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_request_pto",
		"File a time-off request for a worker who asked for it, over the days given. It "+
			"counts the days against their policy and refuses days that overlap time off "+
			"already booked, a blackout, or more than the balance allows. Under a policy that "+
			"needs no approval it is approved and booked at once and the driver is told; "+
			"otherwise it waits for worker_approve_pto.",
		"Files time off the driver sees in Dash; under a policy with no approval step it is "+
			"booked and the driver told at once, and worker_cancel_pto withdraws it.",
		permission.ResourceWorkerPTO,
		permission.OpCreate,
	), properties, toolparams.ParamWorkerID, paramPTOType, paramPTOStart, paramPTOEnd)
	spec.Egress = agent.EgressDriverVisible
	spec.SearchTerms = []string{"vacation", "time off", "PTO request", "day off"}

	return toolspec.NewReportingReceivableTool(
		spec,
		toolspec.ReceivablePlan[*worker.WorkerPTO, *worker.WorkerPTO]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*worker.WorkerPTO, error) {
				workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
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
			Plan: func(
				ctx context.Context,
				entity *worker.WorkerPTO,
				params *agenttoolports.ToolExecuteParams,
			) (*worker.WorkerPTO, error) {
				return requests.PlanCreate(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*worker.WorkerPTO) string { return "Would file a time-off request." },
			Render: func(_ *worker.WorkerPTO, planned *worker.WorkerPTO) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Create(
					workforcekit.Record(permission.ResourceWorkerPTO, pulid.Nil,
						ptoRecord(planned).Label, 0),
					planned, workforcekit.Options(ptoFields...)...)
				if err != nil {
					return nil, err
				}
				summary := fmt.Sprintf("Would request %s days of %s from %s to %s.",
					planned.Days.String(), planned.Type, changepreview.DayText(planned.StartDate),
					changepreview.DayText(planned.EndDate))
				if planned.AutoApproved {
					summary += " The worker's policy needs no approval, so it is booked at once " +
						"and the driver is told."
				}
				return toolpreview.Build(summary, change), nil
			},
			Run: func(
				ctx context.Context,
				entity *worker.WorkerPTO,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := requests.Create(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result(
					"requested",
					kindTimeOff,
					"ptoId",
					created.ID,
					created.WorkerID,
				), nil
			},
		},
	)
}

type ptoEdit struct {
	id     pulid.ID
	params *agenttoolports.ToolExecuteParams
}

func (e *ptoEdit) entity(ctx context.Context, requests ptoRequester) (*worker.WorkerPTO, error) {
	current, err := requests.Get(ctx, &workerptoports.GetPTOByIDRequest{
		ID:         e.id,
		TenantInfo: toolparams.TenantFrom(*e.params),
	})
	if err != nil {
		return nil, err
	}
	entity := *current
	entity.Worker = nil
	return &entity, applyPTODates(&entity, e.params.Params, false)
}

func newUpdateWorkerPTOTool(requests ptoRequester) agenttoolports.AgentTool {
	properties := ptoDatesProperties()
	properties["ptoId"] = agenttoolschema.RecordIDText(
		permission.ResourceWorkerPTO,
		ptopreview.PtoIDNote,
	)
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_pto",
		"Change the kind, days or reason of a time-off request still waiting for a "+
			"decision. Give only what changes. A decided request is refused; cancel it and "+
			"file another.",
		"Changes a pending request the driver sees in Dash; it is changed again the same way.",
		permission.ResourceWorkerPTO,
		permission.OpUpdate,
	), properties, "ptoId"), "ptoId", permission.ResourceWorkerPTO)
	spec.SearchTerms = []string{
		"edit time off request",
		"edit vacation request",
		"change pto request",
	}
	spec.Egress = agent.EgressDriverVisible

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*ptoEdit, *kernelports.RecordChange[worker.WorkerPTO],
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*ptoEdit, error) {
			id, err := toolparams.RequirePulid(params.Params, "ptoId")
			if err != nil {
				return nil, err
			}
			return &ptoEdit{id: id, params: params}, nil
		},
		Plan: func(
			ctx context.Context,
			edit *ptoEdit,
			_ *agenttoolports.ToolExecuteParams,
		) (*kernelports.RecordChange[worker.WorkerPTO], error) {
			entity, err := edit.entity(ctx, requests)
			if err != nil {
				return nil, err
			}
			return requests.PlanUpdate(ctx, entity)
		},
		Refused: func(*ptoEdit) string { return "Would change a time-off request." },
		Render: func(
			_ *ptoEdit,
			change *kernelports.RecordChange[worker.WorkerPTO],
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(ptoRecord(change.Before), change.Before,
				change.After, workforcekit.Options(ptoFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would change the time-off request.", recorded), nil
		},
		Run: func(
			ctx context.Context,
			edit *ptoEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, requests)
			if err != nil {
				return nil, err
			}
			updated, err := requests.Update(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result(
				"updated",
				kindTimeOff,
				"ptoId",
				updated.ID,
				updated.WorkerID,
			), nil
		},
	})
}

func ptoAdjustmentFrom(
	params *agenttoolports.ToolExecuteParams,
) (*ptoledgerservice.AdjustRequest, error) {
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := toolparams.RequireEnum(
		params.Params,
		paramPTOType,
		agenttoolschema.PTOTypes.Values,
	)
	if err != nil {
		return nil, err
	}
	amount, present, err := toolparams.OptionalDecimal(params.Params, paramAmountDays)
	if err != nil {
		return nil, err
	}
	if !present || amount.IsZero() {
		return nil, fmt.Errorf("parameter %q must be a number of days other than zero",
			paramAmountDays)
	}
	note, err := toolparams.RequireBoundedText(
		params.Params,
		changepreview.FieldNote,
		workforcekit.ShortChars,
	)
	if err != nil {
		return nil, err
	}
	effective, err := toolparams.OptionalScheduleDay(params.Params, paramEffectiveOn)
	if err != nil {
		return nil, err
	}
	req := &ptoledgerservice.AdjustRequest{
		TenantInfo: toolparams.TenantFrom(*params),
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

func newAdjustWorkerPTOBalanceTool(ledger PtoBalanceAdjuster) agenttoolports.AgentTool {
	spec := workforcekit.PersonOnly(workforcekit.WithSchema(workforcekit.Spec(
		"worker_adjust_pto_balance",
		"Draft a manual adjustment to a worker's time-off balance, in days, for a person to "+
			"approve. Positive days add and negative days take away, with a note saying why, "+
			"such as a balance carried over from a previous system.",
		"Changes how many paid days a worker holds, which the organization owes and may pay "+
			"out, so a person approves it and it runs as them.",
		permission.ResourceWorkerPTO,
		permission.OpManage,
	), map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramPTOType: agenttoolschema.Enum(
			"The balance to adjust.",
			agenttoolschema.PTOTypes,
		),
		paramAmountDays: toolparams.AmountProperty(
			"Days to add, or a negative number of days to take " +
				"away, such as 1.5 or -2.",
		),
		paramEffectiveOn: agenttoolschema.Date(
			"The day it takes effect. Defaults to today.",
		),
		changepreview.FieldNote: toolparams.StringProperty(
			"Why, kept on the ledger. Required.",
			workforcekit.ShortChars,
		),
	}, toolparams.ParamWorkerID, paramPTOType, paramAmountDays, changepreview.FieldNote))
	spec.SearchTerms = []string{"balance adjustment", "pto balance"}
	spec.Egress = agent.EgressMoney
	spec.Reversible = false

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*ptoledgerservice.AdjustRequest, *ptoledgerservice.AdjustPlan,
	]{
		Request: ptoAdjustmentFrom,

		Plan: func(
			ctx context.Context,
			req *ptoledgerservice.AdjustRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*ptoledgerservice.AdjustPlan, error) {
			return ledger.PlanAdjust(ctx, req)
		},
		Refused: func(*ptoledgerservice.AdjustRequest) string {
			return "Would adjust a time-off balance."
		},
		Render: func(
			req *ptoledgerservice.AdjustRequest,
			plan *ptoledgerservice.AdjustPlan,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(
				workforcekit.Record(permission.ResourceWorkerPTO, pulid.Nil,
					fmt.Sprintf("%s balance adjustment", req.PTOType), 0),
				plan.Entry,
				append(workforcekit.Options(workforcekit.FieldWorkerID, "ptoType", "entryType",
					paramAmountDays, "balanceAfterDays", "effectiveAt", changepreview.FieldNote),
					toolpreview.Volatile("effectiveAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would adjust the %s balance by %s days, from %s to %s.", req.PTOType,
				req.AmountDays.String(), plan.BalanceBefore.String(),
				plan.BalanceAfter.String()), change), nil
		},
		Run: func(
			ctx context.Context,
			req *ptoledgerservice.AdjustRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entry, err := ledger.Adjust(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("adjusted", kindBalance, "ledgerEntryId", entry.ID,
				entry.WorkerID), nil
		},
	})
}

func provideRequestWorkerPTOTool(service workerptoports.WorkerPTOService) agenttoolports.AgentTool {
	return newRequestWorkerPTOTool(service)
}

func provideUpdateWorkerPTOTool(service workerptoports.WorkerPTOService) agenttoolports.AgentTool {
	return newUpdateWorkerPTOTool(service)
}

func provideAdjustWorkerPTOBalanceTool(ledger *ptoledgerservice.Service) agenttoolports.AgentTool {
	return newAdjustWorkerPTOBalanceTool(ledger)
}
