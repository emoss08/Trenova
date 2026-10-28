package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolListWatchtowerItems = "exception_list_watchtower_items"
	paramExceptionAssignee  = "assigneeId"
	paramExceptionNote      = "note"
	paramSnoozeUntil        = "until"
	paramDueAt              = "dueAt"
	paramResolutionCode     = "resolutionCode"
	paramCommentBody        = "body"
	maxExceptionNoteChars   = watchtower.MaxNoteLength
)

var resolutionCodes = agenttoolschema.Source(
	"watchtower.resolutionCode",
	watchtower.SelectableResolutionCodes(),
)

type exceptionView struct {
	Status         string `json:"status"`
	AssigneeID     string `json:"assigneeId,omitempty"`
	DueAt          *int64 `json:"dueAt,omitempty"`
	SnoozedUntil   *int64 `json:"snoozedUntil,omitempty"`
	ResolutionCode string `json:"resolutionCode,omitempty"`
	ResolutionNote string `json:"resolutionNote,omitempty"`
	ResolvedAt     *int64 `json:"resolvedAt,omitempty"`
}

func exceptionViewOf(item *watchtower.Item) *exceptionView {
	view := &exceptionView{
		Status:         item.Status.String(),
		DueAt:          item.DueAt,
		SnoozedUntil:   item.SnoozedUntil,
		ResolutionNote: item.ResolutionNote,
		ResolvedAt:     item.ResolvedAt,
	}
	if item.AssigneeID.IsNotNil() {
		view.AssigneeID = item.AssigneeID.String()
	}
	if item.ResolutionCode != "" {
		view.ResolutionCode = item.ResolutionCode.Label()
	}
	return view
}

func exceptionItemProperty() map[string]any {
	return agenttoolschema.OfKinds(toolparams.StringProperty("The exception, from "+
		"exception_list_watchtower_items. Never guess one.", 0), permission.KindWatchtowerItem)
}

func exceptionNoteProperty(what string) map[string]any {
	return toolparams.StringProperty(what, maxExceptionNoteChars)
}

func exceptionTargetFrom(
	params *agenttoolports.ToolExecuteParams,
) (watchtowerports.ExceptionTarget, error) {
	id, err := toolparams.RequirePulid(params.Params, paramWatchtowerItemID)
	if err != nil {
		return watchtowerports.ExceptionTarget{}, err
	}
	return watchtowerports.ExceptionTarget{
		TenantInfo: toolparams.TenantFrom(*params),
		ItemID:     id,
	}, nil
}

type exceptionChangeSpec[R any] struct {
	spec    *toolspec.ReceivableSpec
	request func(params *agenttoolports.ToolExecuteParams) (R, error)
	target  func(req R) *watchtowerports.ExceptionTarget
	apply   func(
		ctx context.Context,
		req R,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	summary func(before, after *watchtower.Item) string
	action  string
}

func newExceptionChangeTool[R any](
	exceptions watchtowerports.ExceptionService,
	cfg *exceptionChangeSpec[R],
) agenttoolports.AgentTool {
	cfg.spec.Resource = permission.ResourceWatchtower
	cfg.spec.Operation = permission.OpUpdate
	cfg.spec.Egress = agent.EgressInternal
	cfg.spec.Target = func(params map[string]any) (agenttoolports.ToolTarget, bool) {
		return toolparams.TargetOf(params, paramWatchtowerItemID, permission.ResourceWatchtower)
	}
	plan := func(
		ctx context.Context,
		req R,
		params *agenttoolports.ToolExecuteParams,
	) (*kernelports.RecordChange[watchtower.Item], error) {
		target := cfg.target(req)
		before, err := exceptions.GetException(ctx, watchtowerports.GetWatchtowerItemRequest{
			ID:         target.ItemID,
			TenantInfo: target.TenantInfo,
		}, params.Actor)
		if err != nil {
			return nil, err
		}
		if before.SourceKind.OverseesAgents() {
			return nil, errortypes.NewValidationError(paramWatchtowerItemID, errortypes.ErrInvalid,
				"An exception about an agent's own work stays for a person to work")
		}
		target.DryRun = true
		after, err := cfg.apply(ctx, req, params.Actor)
		target.DryRun = false
		if err != nil {
			return nil, err
		}
		return &kernelports.RecordChange[watchtower.Item]{Before: before, After: after}, nil
	}

	return toolspec.NewReceivableTool(
		cfg.spec,
		toolspec.ReceivablePlan[R, *kernelports.RecordChange[watchtower.Item]]{
			Request: cfg.request,
			Settle: func(ctx context.Context, req R, params *agenttoolports.ToolExecuteParams) error {
				_, err := plan(ctx, req, params)
				return err
			},
			Plan: plan,
			Refused: func(R) string {
				return "Would change an exception on the watchtower."
			},
			Render: func(_ R, change *kernelports.RecordChange[watchtower.Item]) (*agent.ToolPreview, error) {
				built, err := toolpreview.Changed(toolpreview.Record{
					Resource: permission.ResourceWatchtower,
					ID:       change.Before.ID,
					Label:    change.Before.Title,
					Version:  changepreview.PinnedVersion(change.Before.Version),
				}, exceptionViewOf(change.Before), exceptionViewOf(change.After),
					toolpreview.Volatile("resolvedAt"))
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(cfg.summary(change.Before, change.After), built), nil
			},
			Run: func(
				ctx context.Context,
				req R,
				params *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				target := cfg.target(req)
				target.DryRun = false
				saved, err := cfg.apply(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return &agent.ToolExecutionResult{
					Action: cfg.action,
					Kind:   "exception",
					Name:   saved.Title,
					IDs:    map[string]string{paramWatchtowerItemID: saved.ID.String()},
				}, nil
			},
		},
	)
}

func NewAssignExceptionTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.AssignExceptionRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_assign",
				Recipe: []string{toolListWatchtowerItems, "exception_assign"},
				Description: "Give an exception on the watchtower to a person, or to nobody, so " +
					"someone owns getting it dealt with. The new owner must be able to see the record " +
					"behind it; they are told it is theirs.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale: "Changes who owns an exception inside Trenova and notifies that person; " +
					"the record behind it is unchanged.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
					paramExceptionAssignee: agenttoolschema.RecordIDText(permission.ResourceUser,
						"The person who should own it: the person who asked, or an assigneeId "+
							"exception_list_watchtower_items shows. Omit to leave it with nobody. Never guess one."),
					paramExceptionNote: exceptionNoteProperty(
						"Why it goes to them, kept in its history.",
					),
				},
				Required:    []string{paramWatchtowerItemID},
				SearchTerms: []string{"assign exception", "give to", "owner", "reassign alert"},
			},
			request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.AssignExceptionRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				req := &watchtowerports.AssignExceptionRequest{
					ExceptionTarget: target,
					Note:            toolparams.OptionalString(params.Params, paramExceptionNote),
				}
				assignee, err := toolparams.OptionalPulidParam(
					params.Params,
					paramExceptionAssignee,
				)
				if err != nil {
					return nil, err
				}
				if assignee != nil {
					req.AssigneeID = *assignee
				}
				return req, nil
			},
			target: func(req *watchtowerports.AssignExceptionRequest) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.AssignExceptionRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.AssignException(ctx, req, actor)
			},
			summary: func(before, after *watchtower.Item) string {
				if after.AssigneeID.IsNil() {
					return fmt.Sprintf("Would leave %q with nobody.", before.Title)
				}
				return fmt.Sprintf(
					"Would give %q to a new owner, who is told it is theirs.",
					before.Title,
				)
			},
			action: "assigned",
		},
	)
}

func NewSnoozeExceptionTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.SnoozeExceptionRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_snooze",
				Recipe: []string{toolListWatchtowerItems, "exception_snooze"},
				Description: "Put an exception aside until a time when nothing can be done before " +
					"then, such as a receiver that opens at 7. It wakes on its own and its owner is told.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale:   "Hides an exception from the open queue until a time; nothing outside Trenova changes.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
					paramSnoozeUntil: agenttoolschema.LocalDateTime(
						"When it should come back, at most 30 days out."),
					paramExceptionNote: exceptionNoteProperty(
						"Why it can wait, kept in its history.",
					),
				},
				Required:    []string{paramWatchtowerItemID, paramSnoozeUntil},
				SearchTerms: []string{"snooze exception", "remind me later", "put aside"},
			},
			request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.SnoozeExceptionRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				until, err := toolparams.RequireLocalTime(params.Params, paramSnoozeUntil,
					timeutils.LoadLocation(params.Timezone))
				if err != nil {
					return nil, err
				}
				return &watchtowerports.SnoozeExceptionRequest{
					ExceptionTarget: target,
					Until:           until,
					Note:            toolparams.OptionalString(params.Params, paramExceptionNote),
				}, nil
			},
			target: func(req *watchtowerports.SnoozeExceptionRequest) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.SnoozeExceptionRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.SnoozeException(ctx, req, actor)
			},
			summary: func(before, _ *watchtower.Item) string {
				return fmt.Sprintf("Would snooze %q; it comes back on its own.", before.Title)
			},
			action: "snoozed",
		},
	)
}

func NewWakeExceptionTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.UnsnoozeExceptionRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_wake",
				Recipe: []string{toolListWatchtowerItems, "exception_wake"},
				Description: "Bring a snoozed exception back to the open queue now, before the " +
					"time exception_snooze set, when whatever it was waiting for has happened " +
					"early. Only a snoozed exception can be woken; exception_list_watchtower_items shows " +
					"which are snoozed and until when.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale: "Puts a snoozed exception back in the open queue inside Trenova; " +
					"nothing outside changes and it can be snoozed again.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
				},
				Required:    []string{paramWatchtowerItemID},
				SearchTerms: []string{"wake exception", "unsnooze", "bring back"},
			},
			request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*watchtowerports.UnsnoozeExceptionRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				return &watchtowerports.UnsnoozeExceptionRequest{ExceptionTarget: target}, nil
			},
			target: func(
				req *watchtowerports.UnsnoozeExceptionRequest,
			) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.UnsnoozeExceptionRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.UnsnoozeException(ctx, req, actor)
			},
			summary: func(before, _ *watchtower.Item) string {
				return fmt.Sprintf("Would bring %q back to the open queue now.", before.Title)
			},
			action: "woken",
		},
	)
}

func NewSetExceptionDueTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.SetExceptionDueRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_set_due",
				Recipe: []string{toolListWatchtowerItems, "exception_set_due"},
				Description: "Move when an exception must be dealt with, or clear its due time. An " +
					"exception past its due time escalates to its owner and the escalation contact.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale:   "Changes when an exception escalates inside Trenova; nothing outside changes.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
					paramDueAt: agenttoolschema.LocalDateTime(
						"The new due time, at most 90 days out. Omit to clear it."),
					paramExceptionNote: exceptionNoteProperty("Why it moved, kept in its history."),
				},
				Required:    []string{paramWatchtowerItemID},
				SearchTerms: []string{"exception due", "extend deadline", "due time"},
			},
			request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.SetExceptionDueRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				dueAt, err := toolparams.OptionalLocalTime(params.Params, paramDueAt,
					timeutils.LoadLocation(params.Timezone))
				if err != nil {
					return nil, err
				}
				return &watchtowerports.SetExceptionDueRequest{
					ExceptionTarget: target,
					DueAt:           dueAt,
					Note:            toolparams.OptionalString(params.Params, paramExceptionNote),
				}, nil
			},
			target: func(req *watchtowerports.SetExceptionDueRequest) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.SetExceptionDueRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.SetExceptionDue(ctx, req, actor)
			},
			summary: func(before, after *watchtower.Item) string {
				if after.DueAt == nil {
					return fmt.Sprintf("Would clear the due time on %q.", before.Title)
				}
				return fmt.Sprintf("Would move when %q is due.", before.Title)
			},
			action: "rescheduled",
		},
	)
}

func NewResolveExceptionTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.ResolveExceptionRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_resolve",
				Recipe: []string{toolListWatchtowerItems, "exception_resolve"},
				Description: "Close an exception with the reason it is closed. The resolution holds " +
					"while the record behind it still shows the problem; it reopens only after the " +
					"problem clears and comes back. Other needs a note.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale: "Closes an exception inside Trenova with an audited reason; the record " +
					"behind it is unchanged.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
					paramResolutionCode: agenttoolschema.Enum(
						"Why it is closed.",
						resolutionCodes,
					),
					paramExceptionNote: exceptionNoteProperty(
						"How it was resolved. Required when the code is Other."),
				},
				Required:    []string{paramWatchtowerItemID, paramResolutionCode},
				SearchTerms: []string{"resolve exception", "close exception", "mark handled"},
			},
			request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.ResolveExceptionRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				raw, err := toolparams.RequireString(params.Params, paramResolutionCode)
				if err != nil {
					return nil, err
				}
				code := watchtower.ResolutionCode(raw)
				if !code.IsSelectable() {
					return nil, errortypes.NewValidationError(
						paramResolutionCode,
						errortypes.ErrInvalid,
						"Choose one of the listed resolution codes",
					)
				}
				return &watchtowerports.ResolveExceptionRequest{
					ExceptionTarget: target,
					Code:            code,
					Note:            toolparams.OptionalString(params.Params, paramExceptionNote),
				}, nil
			},
			target: func(req *watchtowerports.ResolveExceptionRequest) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.ResolveExceptionRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.ResolveException(ctx, req, actor)
			},
			summary: func(before, after *watchtower.Item) string {
				return fmt.Sprintf(
					"Would resolve %q as %s.",
					before.Title,
					after.ResolutionCode.Label(),
				)
			},
			action: "resolved",
		},
	)
}

func NewReopenExceptionTool(exceptions watchtowerports.ExceptionService) agenttoolports.AgentTool {
	return newExceptionChangeTool(
		exceptions,
		&exceptionChangeSpec[*watchtowerports.ReopenExceptionRequest]{
			spec: &toolspec.ReceivableSpec{
				Name:   "exception_reopen",
				Recipe: []string{toolListWatchtowerItems, "exception_reopen"},
				Description: "Open a resolved exception again when it was closed too soon. It gets a " +
					"fresh due time and keeps its owner.",
				DefaultTier: agent.TierPropose,
				MaxTier:     agent.TierActWithApproval,
				Reversible:  true,
				Rationale:   "Puts an exception back on the open queue inside Trenova; nothing outside changes.",
				Properties: map[string]any{
					paramWatchtowerItemID: exceptionItemProperty(),
					paramExceptionNote: exceptionNoteProperty(
						"Why it is open again, kept in its history.",
					),
				},
				Required:    []string{paramWatchtowerItemID},
				SearchTerms: []string{"reopen exception", "not resolved"},
			},
			request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.ReopenExceptionRequest, error) {
				target, err := exceptionTargetFrom(params)
				if err != nil {
					return nil, err
				}
				return &watchtowerports.ReopenExceptionRequest{
					ExceptionTarget: target,
					Note:            toolparams.OptionalString(params.Params, paramExceptionNote),
				}, nil
			},
			target: func(req *watchtowerports.ReopenExceptionRequest) *watchtowerports.ExceptionTarget {
				return &req.ExceptionTarget
			},
			apply: func(
				ctx context.Context,
				req *watchtowerports.ReopenExceptionRequest,
				actor *permissionports.RequestActor,
			) (*watchtower.Item, error) {
				return exceptions.ReopenException(ctx, req, actor)
			},
			summary: func(before, _ *watchtower.Item) string {
				return fmt.Sprintf("Would reopen %q with a fresh due time.", before.Title)
			},
			action: "reopened",
		},
	)
}

func NewCommentOnExceptionTool(
	exceptions watchtowerports.ExceptionService,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:   "exception_add_comment",
		Recipe: []string{toolListWatchtowerItems, "exception_add_comment"},
		Description: "Add a note to an exception's history, such as what the carrier said. " +
			"Its owner is told. The history is never edited.",
		Resource:    permission.ResourceWatchtower,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale:   "Adds a line to an exception's history inside Trenova and tells its owner.",
		Properties: map[string]any{
			paramWatchtowerItemID: exceptionItemProperty(),
			paramCommentBody: toolparams.StringProperty("The comment, in plain words.",
				watchtower.MaxCommentLength),
		},
		Required:    []string{paramWatchtowerItemID, paramCommentBody},
		SearchTerms: []string{"comment on exception", "note on alert", "exception history"},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramWatchtowerItemID, permission.ResourceWatchtower)
		},
	}, toolspec.ReceivablePlan[*watchtowerports.CommentOnExceptionRequest, *watchtower.Event]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*watchtowerports.CommentOnExceptionRequest, error) {
			target, err := exceptionTargetFrom(params)
			if err != nil {
				return nil, err
			}
			body, err := toolparams.RequireString(params.Params, paramCommentBody)
			if err != nil {
				return nil, err
			}
			return &watchtowerports.CommentOnExceptionRequest{
				ExceptionTarget: target,
				Body:            body,
			}, nil
		},
		Settle: func(
			ctx context.Context,
			req *watchtowerports.CommentOnExceptionRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			_, err := previewComment(ctx, exceptions, req, params)
			return err
		},
		Plan: func(
			ctx context.Context,
			req *watchtowerports.CommentOnExceptionRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*watchtower.Event, error) {
			return previewComment(ctx, exceptions, req, params)
		},
		Refused: func(*watchtowerports.CommentOnExceptionRequest) string {
			return "Would add a comment to an exception."
		},
		Render: func(
			_ *watchtowerports.CommentOnExceptionRequest,
			event *watchtower.Event,
		) (*agent.ToolPreview, error) {
			created, err := toolpreview.Create(toolpreview.Record{
				Resource: permission.ResourceWatchtower,
				ID:       event.ItemID,
				Label:    "Exception comment",
			}, &struct {
				Comment string `json:"comment"`
			}{Comment: event.Note})
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(
				"Would add a comment to the exception's history.",
				created,
			), nil
		},
		Run: func(
			ctx context.Context,
			req *watchtowerports.CommentOnExceptionRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req.DryRun = false
			event, err := exceptions.CommentOnException(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			return &agent.ToolExecutionResult{
				Action: "commented",
				Kind:   "exception",
				IDs:    map[string]string{paramWatchtowerItemID: event.ItemID.String()},
			}, nil
		},
	})
}

func previewComment(
	ctx context.Context,
	exceptions watchtowerports.ExceptionService,
	req *watchtowerports.CommentOnExceptionRequest,
	params *agenttoolports.ToolExecuteParams,
) (*watchtower.Event, error) {
	req.DryRun = true
	event, err := exceptions.CommentOnException(ctx, req, params.Actor)
	req.DryRun = false
	return event, err
}
