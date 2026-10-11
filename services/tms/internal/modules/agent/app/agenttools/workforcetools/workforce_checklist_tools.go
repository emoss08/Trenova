package workforcetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramChecklistID = "checklistId"

const paramChecklistItemID = "checklistItemId"

const paramChecklistTemplate = "templateId"

const paramStartedAt = wfFieldStartedAt

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
	common.FieldStatus, wfFieldCompletedAt, "completedById", common.FieldNote, paramEvidenceDocument,
}

type ChecklistKeeper interface {
	PlanStart(
		ctx context.Context,
		req *workerchecklistservice.StartRequest,
	) (*workerchecklistservice.StartPlan, error)
	Start(
		ctx context.Context,
		req *workerchecklistservice.StartRequest,
	) (*worker.WorkerChecklist, error)
	PlanCompleteItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	CompleteItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*worker.WorkerChecklist, error)
	PlanSkipItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	SkipItem(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*worker.WorkerChecklist, error)
	PlanMarkItemNotApplicable(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	MarkItemNotApplicable(
		ctx context.Context,
		req *workerchecklistservice.ItemRequest,
	) (*worker.WorkerChecklist, error)
	PlanReopenItem(
		ctx context.Context,
		req *workerchecklistservice.ReopenItemRequest,
	) (*workerchecklistservice.ItemPlan, error)
	ReopenItem(
		ctx context.Context,
		req *workerchecklistservice.ReopenItemRequest,
	) (*worker.WorkerChecklist, error)
	PlanCancel(
		ctx context.Context,
		req *workerchecklistservice.CancelRequest,
	) (*workerchecklistservice.ChecklistChange, error)
	Cancel(
		ctx context.Context,
		req *workerchecklistservice.CancelRequest,
	) (*worker.WorkerChecklist, error)
}

func ChecklistToolProviders() []any {
	return []any{
		provideStartWorkerChecklistTool,
		provideUpdateWorkerChecklistItemTool,
		provideCancelWorkerChecklistTool,
	}
}

func checklistRecord(checklist *worker.WorkerChecklist) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerChecklist, checklist.ID, checklist.Name,
		checklist.Version)
}

func newStartWorkerChecklistTool(checklists ChecklistKeeper) agentports.AgentTool {
	spec := withSchema(wfSpec(
		"start_worker_checklist",
		"Start a checklist for a worker from one of the organization's hand-started "+
			"templates, such as an annual file audit. Onboarding and offboarding checklists "+
			"start from the hire or the termination and are refused here. A template already "+
			"open for the worker is not started twice.",
		"Opens a checklist on the worker inside Trenova; nothing is sent, and "+
			"cancel_worker_checklist cancels it.",
		permission.ResourceWorkerChecklist,
		permission.OpCreate,
	), map[string]any{
		common.ParamWorkerID: common.WorkerProperty(),
		paramChecklistTemplate: agenttoolschema.RecordIDText(
			permission.ResourceWorkerChecklistTemplate,
			"The template, from list_worker_checklists. Never "+
				"guess one.",
		),
		paramStartedAt: agenttoolschema.Date("The day it starts, which its items fall due from. " +
			"Defaults to today."),
	}, common.ParamWorkerID, paramChecklistTemplate)

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerchecklistservice.StartRequest, *workerchecklistservice.StartPlan,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workerchecklistservice.StartRequest, error) {
			workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
			if err != nil {
				return nil, err
			}
			templateID, err := common.RequirePulid(params.Params, paramChecklistTemplate)
			if err != nil {
				return nil, err
			}
			started, err := common.OptionalScheduleDay(params.Params, paramStartedAt)
			if err != nil {
				return nil, err
			}
			req := &workerchecklistservice.StartRequest{
				TenantInfo: common.TenantFrom(*params),
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
			_ *agentports.ToolExecuteParams,
		) (*workerchecklistservice.StartPlan, error) {
			return checklists.PlanStart(ctx, req)
		},
		Refused: func(*workerchecklistservice.StartRequest) string {
			return "Would start a checklist for the worker."
		},
		Render: func(
			_ *workerchecklistservice.StartRequest,
			plan *workerchecklistservice.StartPlan,
		) (*agent.ToolPreview, error) {
			if plan.Existing {
				return toolpreview.Build(fmt.Sprintf(
					"%s is already open for the worker; nothing new would start.",
					plan.Checklist.Name)), nil
			}
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerChecklist, pulid.Nil, plan.Checklist.Name, 0),
				plan.Checklist, wfOptions(wfFieldWorkerID, paramChecklistTemplate, "name",
					common.FieldKind, common.FieldStatus, wfFieldStartedAt, "dueAt")...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would start %s with %d item(s).",
				plan.Checklist.Name, len(plan.Checklist.Items)), change), nil
		},
		Run: func(
			ctx context.Context,
			req *workerchecklistservice.StartRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			started, err := checklists.Start(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("started", kindChecklist, paramChecklistID, started.ID,
				started.WorkerID), nil
		},
	})
}

type checklistItemChange struct {
	move   checklistItemMove
	item   *workerchecklistservice.ItemRequest
	reopen *workerchecklistservice.ReopenItemRequest
}

func checklistItemChangeFrom(params *agentports.ToolExecuteParams) (*checklistItemChange, error) {
	id, err := common.RequirePulid(params.Params, paramChecklistItemID)
	if err != nil {
		return nil, err
	}
	move, err := common.RequireEnum(params.Params, paramItemMove, checklistItemMoves.Values)
	if err != nil {
		return nil, err
	}
	note, err := common.BoundedText(params.Params, common.FieldNote, wfNoteChars)
	if err != nil {
		return nil, err
	}
	evidence, err := common.OptionalID(params.Params, paramEvidenceDocument)
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
			TenantInfo: common.TenantFrom(*params),
			UserID:     params.Actor.UserID,
		}}, nil
	}
	return &checklistItemChange{move: move, item: &workerchecklistservice.ItemRequest{
		ID:                 id,
		TenantInfo:         common.TenantFrom(*params),
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
	return nil, common.ErrUnknownValue(paramItemMove, string(c.move), checklistItemMoves.Names())
}

func (c *checklistItemChange) run(
	ctx context.Context,
	checklists ChecklistKeeper,
) (*worker.WorkerChecklist, error) {
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
	return nil, common.ErrUnknownValue(paramItemMove, string(c.move), checklistItemMoves.Names())
}

func newUpdateWorkerChecklistItemTool(checklists ChecklistKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"update_worker_checklist_item",
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
			"The item, from list_worker_checklists. Never guess "+
				"one.",
			permission.KindChecklistItem,
		),
		paramItemMove: agenttoolschema.Enum("Complete, Skip, NotApplicable or Reopen.",
			checklistItemMoves),
		common.FieldNote: common.StringProperty("What was done, or why it was skipped or does not apply. "+
			"Required to skip or mark not applicable.", wfNoteChars),
		paramEvidenceDocument: agenttoolschema.RecordIDText(
			permission.ResourceDocument,
			"For Complete: a document filed on the worker "+
				"that shows it done, from search_documents.",
		),
	}, paramChecklistItemID, paramItemMove), paramChecklistItemID,
		permission.ResourceWorkerChecklist)

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*checklistItemChange, *workerchecklistservice.ItemPlan,
	]{
		Request: checklistItemChangeFrom,
		Plan: func(
			ctx context.Context,
			change *checklistItemChange,
			_ *agentports.ToolExecuteParams,
		) (*workerchecklistservice.ItemPlan, error) {
			return change.plan(ctx, checklists)
		},
		Refused: func(change *checklistItemChange) string {
			return fmt.Sprintf("Would %s a checklist item.", checklistMoveVerb(change.move))
		},
		Render: func(
			change *checklistItemChange,
			plan *workerchecklistservice.ItemPlan,
		) (*agent.ToolPreview, error) {
			item := plan.Item.Before
			options := append(wfOptions(checklistItemFields...),
				toolpreview.Volatile(wfFieldCompletedAt))
			recorded, err := toolpreview.Changed(
				wfRecord(permission.ResourceWorkerChecklist, item.ID, item.Label, item.Version),
				plan.Item.Before, plan.Item.After, options...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would %s %q on %s.",
				checklistMoveVerb(change.move), item.Label, plan.Checklist.Name), recorded), nil
		},
		Run: func(
			ctx context.Context,
			change *checklistItemChange,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			checklist, err := change.run(ctx, checklists)
			if err != nil {
				return nil, err
			}
			result := wfResult(checklistMovePast(change.move), kindChecklistItem,
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

func newCancelWorkerChecklistTool(checklists ChecklistKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"cancel_worker_checklist",
		"Cancel an open checklist that no longer applies, with the reason kept. Its items "+
			"stay as they were.",
		"Cancels a checklist inside Trenova; nothing is sent, and a cancelled checklist is "+
			"started again from its template.",
		permission.ResourceWorkerChecklist,
		permission.OpCancel,
	), map[string]any{
		paramChecklistID: agenttoolschema.RecordIDText(
			permission.ResourceWorkerChecklist,
			"The checklist, from list_worker_checklists. Never "+
				"guess one.",
		),
		common.FieldReason: common.StringProperty("Why it no longer applies.", wfShortChars),
	}, paramChecklistID, common.FieldReason), paramChecklistID, permission.ResourceWorkerChecklist)
	spec.SearchTerms = []string{"drop checklist", "no longer applies"}
	spec.MaxTier = agent.TierPropose

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerchecklistservice.CancelRequest, *workerchecklistservice.ChecklistChange,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workerchecklistservice.CancelRequest, error) {
			id, err := common.RequirePulid(params.Params, paramChecklistID)
			if err != nil {
				return nil, err
			}
			reason, err := common.RequireBoundedText(params.Params, common.FieldReason, wfShortChars)
			if err != nil {
				return nil, err
			}
			return &workerchecklistservice.CancelRequest{
				ID:         id,
				TenantInfo: common.TenantFrom(*params),
				Reason:     reason,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerchecklistservice.CancelRequest,
			_ *agentports.ToolExecuteParams,
		) (*workerchecklistservice.ChecklistChange, error) {
			return checklists.PlanCancel(ctx, req)
		},
		Refused: func(*workerchecklistservice.CancelRequest) string {
			return "Would cancel a checklist."
		},
		Render: func(
			_ *workerchecklistservice.CancelRequest,
			change *workerchecklistservice.ChecklistChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(checklistRecord(change.Before), change.Before,
				change.After, append(wfOptions(common.FieldStatus, "cancelledAt", "cancelReason"),
					toolpreview.Volatile("cancelledAt"))...)
			if err != nil {
				return nil, err
			}
			summary := "Would cancel " + change.Before.Name + "."
			if change.Before.Status != worker.ChecklistStatusOpen {
				summary = fmt.Sprintf("%s is already %s; nothing would change.",
					change.Before.Name, strings.ToLower(string(change.Before.Status)))
			}
			return toolpreview.Build(summary, recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerchecklistservice.CancelRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			cancelled, err := checklists.Cancel(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("cancelled", kindChecklist, paramChecklistID, cancelled.ID,
				cancelled.WorkerID), nil
		},
	})
}

func provideStartWorkerChecklistTool(s *workerchecklistservice.Service) agentports.AgentTool {
	return newStartWorkerChecklistTool(s)
}

func provideUpdateWorkerChecklistItemTool(
	s *workerchecklistservice.Service,
) agentports.AgentTool {
	return newUpdateWorkerChecklistItemTool(s)
}

func provideCancelWorkerChecklistTool(s *workerchecklistservice.Service) agentports.AgentTool {
	return newCancelWorkerChecklistTool(s)
}
