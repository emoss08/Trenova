package workforcetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPTOType = "type"

const paramPTOStart = wfFieldStartDate

const paramPTOEnd = wfFieldEndDate

const paramAmountDays = "amountDays"

const paramEffectiveOn = "effectiveDate"

const kindTimeOff = "time off request"

const kindBalance = "time off balance"

const maxPTOReasonChars = 255

var ptoFields = []string{
	wfFieldWorkerID,
	paramPTOType,
	common.FieldStatus,
	paramPTOStart,
	paramPTOEnd,
	common.FieldReason,
	"days",
	"autoApproved",
	wfFieldApproverID,
}

type ptoRequester interface {
	Get(ctx context.Context, req *workerports.GetPTOByIDRequest) (*worker.WorkerPTO, error)
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
	) (*worker.WorkerPTOLedgerEntry, error)
}

func PtoToolProviders() []any {
	return []any{
		provideRequestWorkerPTOTool,
		provideUpdateWorkerPTOTool,
		provideAdjustWorkerPTOBalanceTool,
	}
}

func ptoRecord(pto *worker.WorkerPTO) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerPTO, pto.ID,
		fmt.Sprintf("%s from %s", pto.Type, common.DayText(pto.StartDate)), pto.Version)
}

func ptoDatesProperties() map[string]any {
	return map[string]any{
		paramPTOType: agenttoolschema.Enum("The kind of time off.", agenttoolschema.PTOTypes),
		paramPTOStart: agenttoolschema.Date("The first day off. \"Next Friday\" is the coming " +
			"Friday, not the one a week later."),
		paramPTOEnd: agenttoolschema.Date("The last day off."),
		common.FieldReason: common.StringProperty(
			"The reason the worker gave, when one was given; the "+
				"driver sees it with the request. Omit it rather than ask, and the kind is used.",
			maxPTOReasonChars,
		),
	}
}

func applyPTODates(pto *worker.WorkerPTO, params map[string]any, required bool) error {
	if kind, given, err := common.OptionalEnum(params, paramPTOType,
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
		day, err := common.OptionalScheduleDay(params, field.key)
		switch {
		case err != nil:
			return err
		case day != nil:
			*field.dest = *day
		case required:
			return fmt.Errorf("missing required parameter %q", field.key)
		}
	}
	reason, err := common.OptionalBoundedText(params, common.FieldReason, maxPTOReasonChars)
	switch {
	case err != nil:
		return err
	case reason != nil:
		pto.Reason = *reason
	case required:
		pto.Reason = string(pto.Type)
	}
	return nil
}

func newRequestWorkerPTOTool(pto ptoRequester) agenttoolports.AgentTool {
	properties := ptoDatesProperties()
	properties[common.ParamWorkerID] = common.WorkerProperty()
	spec := withSchema(wfSpec(
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
	), properties, common.ParamWorkerID, paramPTOType, paramPTOStart, paramPTOEnd)
	spec.Egress = agent.EgressDriverVisible
	spec.SearchTerms = []string{"vacation", "time off", "PTO request", "day off"}

	return common.NewReportingReceivableTool(
		spec,
		common.ReceivablePlan[*worker.WorkerPTO, *worker.WorkerPTO]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*worker.WorkerPTO, error) {
				workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
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
				return pto.PlanCreate(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*worker.WorkerPTO) string { return "Would file a time-off request." },
			Render: func(_ *worker.WorkerPTO, planned *worker.WorkerPTO) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerPTO, pulid.Nil,
						ptoRecord(planned).Label, 0),
					planned, wfOptions(ptoFields...)...)
				if err != nil {
					return nil, err
				}
				summary := fmt.Sprintf("Would request %s days of %s from %s to %s.",
					planned.Days.String(), planned.Type, common.DayText(planned.StartDate),
					common.DayText(planned.EndDate))
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
			) (*agent.ToolExecutionResult, error) {
				created, err := pto.Create(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult(
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

func (e *ptoEdit) entity(ctx context.Context, pto ptoRequester) (*worker.WorkerPTO, error) {
	current, err := pto.Get(ctx, &workerports.GetPTOByIDRequest{
		ID:         e.id,
		TenantInfo: common.TenantFrom(*e.params),
	})
	if err != nil {
		return nil, err
	}
	entity := *current
	entity.Worker = nil
	return &entity, applyPTODates(&entity, e.params.Params, false)
}

func newUpdateWorkerPTOTool(pto ptoRequester) agenttoolports.AgentTool {
	properties := ptoDatesProperties()
	properties["ptoId"] = agenttoolschema.RecordIDText(
		permission.ResourceWorkerPTO,
		common.PtoIDNote,
	)
	spec := targeting(withSchema(wfSpec(
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

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*ptoEdit, *kernelports.RecordChange[worker.WorkerPTO],
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*ptoEdit, error) {
			id, err := common.RequirePulid(params.Params, "ptoId")
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
			entity, err := edit.entity(ctx, pto)
			if err != nil {
				return nil, err
			}
			return pto.PlanUpdate(ctx, entity)
		},
		Refused: func(*ptoEdit) string { return "Would change a time-off request." },
		Render: func(
			_ *ptoEdit,
			change *kernelports.RecordChange[worker.WorkerPTO],
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(ptoRecord(change.Before), change.Before,
				change.After, wfOptions(ptoFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would change the time-off request.", recorded), nil
		},
		Run: func(
			ctx context.Context,
			edit *ptoEdit,
			params *agenttoolports.ToolExecuteParams,
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
	params *agenttoolports.ToolExecuteParams,
) (*ptoledgerservice.AdjustRequest, error) {
	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := common.RequireEnum(params.Params, paramPTOType, agenttoolschema.PTOTypes.Values)
	if err != nil {
		return nil, err
	}
	amount, present, err := common.OptionalDecimal(params.Params, paramAmountDays)
	if err != nil {
		return nil, err
	}
	if !present || amount.IsZero() {
		return nil, fmt.Errorf("parameter %q must be a number of days other than zero",
			paramAmountDays)
	}
	note, err := common.RequireBoundedText(params.Params, common.FieldNote, wfShortChars)
	if err != nil {
		return nil, err
	}
	effective, err := common.OptionalScheduleDay(params.Params, paramEffectiveOn)
	if err != nil {
		return nil, err
	}
	req := &ptoledgerservice.AdjustRequest{
		TenantInfo: common.TenantFrom(*params),
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
	spec := personOnly(withSchema(wfSpec(
		"worker_adjust_pto_balance",
		"Draft a manual adjustment to a worker's time-off balance, in days, for a person to "+
			"approve. Positive days add and negative days take away, with a note saying why, "+
			"such as a balance carried over from a previous system.",
		"Changes how many paid days a worker holds, which the organization owes and may pay "+
			"out, so a person approves it and it runs as them.",
		permission.ResourceWorkerPTO,
		permission.OpManage,
	), map[string]any{
		common.ParamWorkerID: common.WorkerProperty(),
		paramPTOType: agenttoolschema.Enum(
			"The balance to adjust.",
			agenttoolschema.PTOTypes,
		),
		paramAmountDays: common.AmountProperty(
			"Days to add, or a negative number of days to take " +
				"away, such as 1.5 or -2.",
		),
		paramEffectiveOn: agenttoolschema.Date("The day it takes effect. Defaults to today."),
		common.FieldNote: common.StringProperty("Why, kept on the ledger. Required.", wfShortChars),
	}, common.ParamWorkerID, paramPTOType, paramAmountDays, common.FieldNote))
	spec.SearchTerms = []string{"balance adjustment", "pto balance"}
	spec.Egress = agent.EgressMoney
	spec.Reversible = false

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
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
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerPTO, pulid.Nil,
					fmt.Sprintf("%s balance adjustment", req.PTOType), 0),
				plan.Entry, append(wfOptions(wfFieldWorkerID, "ptoType", "entryType",
					paramAmountDays, "balanceAfterDays", "effectiveAt", common.FieldNote),
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

func provideRequestWorkerPTOTool(pto workerports.WorkerPTOService) agenttoolports.AgentTool {
	return newRequestWorkerPTOTool(pto)
}

func provideUpdateWorkerPTOTool(pto workerports.WorkerPTOService) agenttoolports.AgentTool {
	return newUpdateWorkerPTOTool(pto)
}

func provideAdjustWorkerPTOBalanceTool(ledger *ptoledgerservice.Service) agenttoolports.AgentTool {
	return newAdjustWorkerPTOBalanceTool(ledger)
}
