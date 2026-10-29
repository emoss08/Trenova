package trainingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramChecklistID = "checklistId"

const paramChecklistItemID = "checklistItemId"

const paramChecklistTemplate = "templateId"

const paramStartedAt = workforcekit.FieldStartedAt

const paramItemMove = "action"

const paramEvidenceDocument = "evidenceDocumentId"

const kindChecklist = "checklist"

const kindChecklistItem = "checklist item"

type checklistItemMove string

const checklistItemComplete = checklistItemMove("Complete")

const checklistItemSkip = checklistItemMove("Skip")

const checklistItemNotApplicable = checklistItemMove("NotApplicable")

const checklistItemReopen = checklistItemMove("Reopen")

var checklistItemMoves = agenttoolschema.Source(
	"workerChecklistItem.agentMove",
	[]checklistItemMove{
		checklistItemComplete,
		checklistItemSkip,
		checklistItemNotApplicable,
		checklistItemReopen,
	},
)

var checklistItemFields = []string{
	changepreview.FieldStatus,
	workforcekit.FieldCompletedAt,
	"completedById",
	changepreview.FieldNote,
	paramEvidenceDocument,
}

type ChecklistKeeper interface {
	PlanStart(
		ctx context.Context,
		req *workerchecklistservice.StartRequest,
	) (*workerchecklistservice.StartPlan, error)
	Start(
		ctx context.Context,
		req *workerchecklistservice.StartRequest,
	) (*workerchecklist.WorkerChecklist, error)
	PlanCompleteItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	CompleteItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklist.WorkerChecklist, error)
	PlanSkipItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	SkipItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklist.WorkerChecklist, error)
	PlanMarkItemNotApplicable(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	MarkItemNotApplicable(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklist.WorkerChecklist, error)
	PlanReopenItem(
		ctx context.Context,
		req *workerchecklistservice.ReopenItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	ReopenItem(
		ctx context.Context,
		req *workerchecklistservice.ReopenItemRequest,
	) (*workerchecklist.WorkerChecklist, error)
	PlanCancel(
		ctx context.Context,
		req *workerchecklistservice.CancelRequest,
	) (*workerchecklistservice.ChecklistChange, error)
	Cancel(
		ctx context.Context,
		req *workerchecklistservice.CancelRequest,
	) (*workerchecklist.WorkerChecklist, error)
}

func ChecklistToolProviders() []any {
	return []any{
		provideStartWorkerChecklistTool,
		provideUpdateWorkerChecklistItemTool,
		provideCancelWorkerChecklistTool,
	}
}

func checklistRecord(checklist *workerchecklist.WorkerChecklist) toolpreview.Record {
	return workforcekit.Record(permission.ResourceWorkerChecklist, checklist.ID, checklist.Name,
		checklist.Version)
}

func newStartWorkerChecklistTool(checklists ChecklistKeeper) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_start_checklist",
		"Start a checklist for a worker from one of the organization's hand-started "+
			"templates, such as an annual file audit. Onboarding and offboarding checklists "+
			"start from the hire or the termination and are refused here. A template already "+
			"open for the worker is not started twice.",
		"Opens a checklist on the worker inside Trenova; nothing is sent, and "+
			"worker_cancel_checklist cancels it.",
		permission.ResourceWorkerChecklist,
		permission.OpCreate,
	), map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramChecklistTemplate: agenttoolschema.RecordIDText(
			permission.ResourceWorkerChecklistTemplate,
			"The template, from worker_list_checklists. Never "+
				"guess one.",
		),
		paramStartedAt: agenttoolschema.Date("The day it starts, which its items fall due from. " +
			"Defaults to today."),
	}, toolparams.ParamWorkerID, paramChecklistTemplate)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerchecklistservice.StartRequest, *workerchecklistservice.StartPlan,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerchecklistservice.StartRequest, error) {
			workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
			if err != nil {
				return nil, err
			}
			templateID, err := toolparams.RequirePulid(params.Params, paramChecklistTemplate)
			if err != nil {
				return nil, err
			}
			started, err := toolparams.OptionalScheduleDay(params.Params, paramStartedAt)
			if err != nil {
				return nil, err
			}
			req := &workerchecklistservice.StartRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				WorkerID:   workerID,
				TemplateID: templateID,
				UserID:     params.Actor.UserID,
			}
			if started != nil {
				req.StartedAt = *started
			}
			return req, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerchecklistservice.StartRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerchecklistservice.StartPlan, error) {
			return checklists.PlanStart(ctx, req)
		},
		Refused: func(*workerchecklistservice.StartRequest) string {
			return "Would start a checklist for the worker."
		},
		Render: func(
			_ *workerchecklistservice.StartRequest,
			plan *workerchecklistservice.StartPlan,
		) (*agentpreview.ToolPreview, error) {
			if plan.Existing {
				return toolpreview.Build(fmt.Sprintf(
					"%s is already open for the worker; nothing new would start.",
					plan.Checklist.Name)), nil
			}
			change, err := toolpreview.Create(
				workforcekit.Record(
					permission.ResourceWorkerChecklist,
					pulid.Nil,
					plan.Checklist.Name,
					0,
				),
				plan.Checklist,
				workforcekit.Options(
					workforcekit.FieldWorkerID,
					paramChecklistTemplate,
					"name",
					changepreview.FieldKind,
					changepreview.FieldStatus,
					workforcekit.FieldStartedAt,
					"dueAt",
				)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would start %s with %d item(s).",
				plan.Checklist.Name, len(plan.Checklist.Items)), change), nil
		},
		Run: func(
			ctx context.Context,
			req *workerchecklistservice.StartRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			started, err := checklists.Start(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("started", kindChecklist, paramChecklistID, started.ID,
				started.WorkerID), nil
		},
	})
}

type checklistItemChange struct {
	move   checklistItemMove
	item   *workerchecklistservice.ItemRequest
	reopen *workerchecklistservice.ReopenItemRequest
}

func checklistItemChangeFrom(
	params *agenttoolports.ToolExecuteParams,
) (*checklistItemChange, error) {
	id, err := toolparams.RequirePulid(params.Params, paramChecklistItemID)
	if err != nil {
		return nil, err
	}
	move, err := toolparams.RequireEnum(params.Params, paramItemMove, checklistItemMoves.Values)
	if err != nil {
		return nil, err
	}
	note, err := toolparams.BoundedText(
		params.Params,
		changepreview.FieldNote,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	evidence, err := toolparams.OptionalID(params.Params, paramEvidenceDocument)
	if err != nil {
		return nil, err
	}
	if move == checklistItemReopen {
		if note != "" || !evidence.IsNil() {
			return nil, errortypes.NewValidationError(paramItemMove, errortypes.ErrInvalid,
				"A reopened item takes no note or evidence")
		}
		return &checklistItemChange{move: move, reopen: &workerchecklistservice.ReopenItemRequest{
			ID:         id,
			TenantInfo: toolparams.TenantFrom(*params),
			UserID:     params.Actor.UserID,
		}}, nil
	}
	return &checklistItemChange{move: move, item: &workerchecklistservice.ItemRequest{
		ID:                 id,
		TenantInfo:         toolparams.TenantFrom(*params),
		Note:               note,
		EvidenceDocumentID: evidence,
		UserID:             params.Actor.UserID,
	}}, nil
}

func (c *checklistItemChange) plan(
	ctx context.Context,
	checklists ChecklistKeeper,
) (*workerchecklistservice.ItemPlan, error) {
	switch c.move {
	case checklistItemComplete:
		return checklists.PlanCompleteItem(ctx, c.item)
	case checklistItemSkip:
		return checklists.PlanSkipItem(ctx, c.item)
	case checklistItemNotApplicable:
		return checklists.PlanMarkItemNotApplicable(ctx, c.item)
	case checklistItemReopen:
		return checklists.PlanReopenItem(ctx, c.reopen)
	}
	return nil, toolparams.ErrUnknownValue(
		paramItemMove,
		string(c.move),
		checklistItemMoves.Names(),
	)
}

func (c *checklistItemChange) run(
	ctx context.Context,
	checklists ChecklistKeeper,
) (*workerchecklist.WorkerChecklist, error) {
	switch c.move {
	case checklistItemComplete:
		return checklists.CompleteItem(ctx, c.item)
	case checklistItemSkip:
		return checklists.SkipItem(ctx, c.item)
	case checklistItemNotApplicable:
		return checklists.MarkItemNotApplicable(ctx, c.item)
	case checklistItemReopen:
		return checklists.ReopenItem(ctx, c.reopen)
	}
	return nil, toolparams.ErrUnknownValue(
		paramItemMove,
		string(c.move),
		checklistItemMoves.Names(),
	)
}

func newUpdateWorkerChecklistItemTool(checklists ChecklistKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_checklist_item",
		"Settle or reopen one item on a worker's open checklist. Complete it when it is "+
			"done, with a document as evidence when there is one; Skip or NotApplicable with "+
			"a note saying why; or Reopen a settled item. A checklist whose required items are all settled closes "+
			"on its own.",
		"Settles or reopens a checklist item inside Trenova; nothing is sent, and a settled "+
			"item is reopened the same way.",
		permission.ResourceWorkerChecklist,
		permission.OpUpdate,
	), map[string]any{
		paramChecklistItemID: agenttoolschema.KindID(
			"The item, from worker_list_checklists. Never guess "+
				"one.",
			recordkind.ChecklistItem,
		),
		paramItemMove: agenttoolschema.Enum("Complete, Skip, NotApplicable or Reopen.",
			checklistItemMoves),
		changepreview.FieldNote: toolparams.StringProperty(
			"What was done, or why it was skipped or does not apply. "+
				"Required to skip or mark not applicable.",
			workforcekit.NoteChars,
		),
		paramEvidenceDocument: agenttoolschema.RecordIDText(
			permission.ResourceDocument,
			"For Complete: a document filed on the worker "+
				"that shows it done, from document_list.",
		),
	}, paramChecklistItemID, paramItemMove), paramChecklistItemID,
		permission.ResourceWorkerChecklist)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*checklistItemChange, *workerchecklistservice.ItemPlan,
	]{
		Request: checklistItemChangeFrom,
		Plan: func(
			ctx context.Context,
			change *checklistItemChange,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerchecklistservice.ItemPlan, error) {
			return change.plan(ctx, checklists)
		},
		Refused: func(change *checklistItemChange) string {
			return fmt.Sprintf("Would %s a checklist item.", checklistMoveVerb(change.move))
		},
		Render: func(
			change *checklistItemChange,
			plan *workerchecklistservice.ItemPlan,
		) (*agentpreview.ToolPreview, error) {
			item := plan.Item.Before
			options := append(workforcekit.Options(checklistItemFields...),
				toolpreview.Volatile(workforcekit.FieldCompletedAt))
			recorded, err := toolpreview.Changed(
				workforcekit.Record(
					permission.ResourceWorkerChecklist,
					item.ID,
					item.Label,
					item.Version,
				),
				plan.Item.Before,
				plan.Item.After,
				options...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would %s %q on %s.",
				checklistMoveVerb(change.move), item.Label, plan.Checklist.Name), recorded), nil
		},
		Run: func(
			ctx context.Context,
			change *checklistItemChange,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			checklist, err := change.run(ctx, checklists)
			if err != nil {
				return nil, err
			}
			result := workforcekit.Result(checklistMovePast(change.move), kindChecklistItem,
				paramChecklistID, checklist.ID, checklist.WorkerID)
			if change.item != nil {
				result.IDs[paramChecklistItemID] = change.item.ID.String()
			} else {
				result.IDs[paramChecklistItemID] = change.reopen.ID.String()
			}
			return result, nil
		},
	})
}

func checklistMoveVerb(move checklistItemMove) string {
	switch move {
	case checklistItemComplete:
		return "complete"
	case checklistItemSkip:
		return "skip"
	case checklistItemNotApplicable:
		return "mark not applicable"
	case checklistItemReopen:
		return "reopen"
	}
	return string(move)
}

func checklistMovePast(move checklistItemMove) string {
	switch move {
	case checklistItemComplete:
		return "completed"
	case checklistItemSkip:
		return "skipped"
	case checklistItemNotApplicable:
		return "marked not applicable"
	case checklistItemReopen:
		return "reopened"
	}
	return string(move)
}

func newCancelWorkerChecklistTool(checklists ChecklistKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_cancel_checklist",
		"Cancel an open checklist that no longer applies, with the reason kept. Its items "+
			"stay as they were.",
		"Cancels a checklist inside Trenova; nothing is sent, and a cancelled checklist is "+
			"started again from its template.",
		permission.ResourceWorkerChecklist,
		permission.OpCancel,
	), map[string]any{
		paramChecklistID: agenttoolschema.RecordIDText(
			permission.ResourceWorkerChecklist,
			"The checklist, from worker_list_checklists. Never "+
				"guess one.",
		),
		changepreview.FieldReason: toolparams.StringProperty(
			"Why it no longer applies.",
			workforcekit.ShortChars,
		),
	}, paramChecklistID, changepreview.FieldReason), paramChecklistID, permission.ResourceWorkerChecklist)
	spec.SearchTerms = []string{"drop checklist", "no longer applies"}
	spec.MaxTier = agent.TierPropose

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerchecklistservice.CancelRequest, *workerchecklistservice.ChecklistChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerchecklistservice.CancelRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramChecklistID)
			if err != nil {
				return nil, err
			}
			reason, err := toolparams.RequireBoundedText(
				params.Params,
				changepreview.FieldReason,
				workforcekit.ShortChars,
			)
			if err != nil {
				return nil, err
			}
			return &workerchecklistservice.CancelRequest{
				ID:         id,
				TenantInfo: toolparams.TenantFrom(*params),
				Reason:     reason,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerchecklistservice.CancelRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerchecklistservice.ChecklistChange, error) {
			return checklists.PlanCancel(ctx, req)
		},
		Refused: func(*workerchecklistservice.CancelRequest) string {
			return "Would cancel a checklist."
		},
		Render: func(
			_ *workerchecklistservice.CancelRequest,
			change *workerchecklistservice.ChecklistChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(
				checklistRecord(change.Before),
				change.Before,
				change.After,
				append(
					workforcekit.Options(changepreview.FieldStatus, "cancelledAt", "cancelReason"),
					toolpreview.Volatile("cancelledAt"),
				)...)
			if err != nil {
				return nil, err
			}
			summary := "Would cancel " + change.Before.Name + "."
			if change.Before.Status != workerchecklist.ChecklistStatusOpen {
				summary = fmt.Sprintf("%s is already %s; nothing would change.",
					change.Before.Name, strings.ToLower(string(change.Before.Status)))
			}
			return toolpreview.Build(summary, recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerchecklistservice.CancelRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			cancelled, err := checklists.Cancel(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("cancelled", kindChecklist, paramChecklistID, cancelled.ID,
				cancelled.WorkerID), nil
		},
	})
}

func provideStartWorkerChecklistTool(s *workerchecklistservice.Service) agenttoolports.AgentTool {
	return newStartWorkerChecklistTool(s)
}

func provideUpdateWorkerChecklistItemTool(
	s *workerchecklistservice.Service,
) agenttoolports.AgentTool {
	return newUpdateWorkerChecklistItemTool(s)
}

func provideCancelWorkerChecklistTool(s *workerchecklistservice.Service) agenttoolports.AgentTool {
	return newCancelWorkerChecklistTool(s)
}
