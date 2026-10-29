package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamCommentID = "commentId"

const ParamCommentText = "comment"

const ParamResolved = "resolved"

const kindComment = "shipment comment"

const maxCommentChars = 4000

const fieldPinnedAt = "pinnedAt"

const FieldPinnedByID = "pinnedById"

const fieldEditedAt = "editedAt"

const FieldPriority = "priority"

const fieldCommentText = "comment"

const fieldDeletedByID = "deletedById"

const fieldRequiresAck = "requiresAcknowledgment"

const fieldCommentType = "type"

const FieldCommentVisib = "visibility"

var commentUserRefs = map[string]permission.Resource{
	FieldPinnedByID:                 permission.ResourceUser,
	changepreview.FieldResolvedByID: permission.ResourceUser,
	fieldDeletedByID:                permission.ResourceUser,
}

type commentModerator interface {
	GetByID(
		ctx context.Context,
		req *shipmentcommentports.GetShipmentCommentByIDRequest,
	) (*shipment.ShipmentComment, error)
	Pin(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unpin(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Resolve(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unresolve(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		req *shipmentcommentports.UpdateShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *shipmentcommentports.DeleteShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) error
	PreviewPin(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
		pinned bool,
	) (*shipmentcommentports.ShipmentCommentChange, error)
	PreviewResolve(
		ctx context.Context,
		req *shipmentcommentports.ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
		resolved bool,
	) (*shipmentcommentports.ShipmentCommentChange, error)
	PreviewUpdate(
		ctx context.Context,
		req *shipmentcommentports.UpdateShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipmentcommentports.ShipmentCommentChange, error)
	PreviewDelete(
		ctx context.Context,
		req *shipmentcommentports.DeleteShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipmentcommentports.ShipmentCommentDeletion, error)
}

type commentPermissions interface {
	Check(
		ctx context.Context,
		req *permissionports.PermissionCheckRequest,
	) (*permissionports.PermissionCheckResult, error)
}

func commentIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceShipmentComment, "The comment",
		"the recentComments shipment_get lists or the page you are on")
}

func commentRecord(comment *shipment.ShipmentComment) toolpreview.Record {
	label := strings.TrimSpace(comment.Comment)
	if len(label) > 60 {
		label = label[:57] + "..."
	}

	return toolpreview.Record{
		Resource: permission.ResourceShipmentComment,
		ID:       comment.ID,
		Label:    "Comment: " + label,
		Version:  changepreview.PreviewVersion(comment.Version),
	}
}

func commentResult(
	action string,
	comment *shipment.ShipmentComment,
) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{Action: action, Kind: kindComment}
	if comment != nil {
		result.IDs = map[string]string{
			ParamCommentID:             comment.ID.String(),
			toolparams.ParamShipmentID: comment.ShipmentID.String(),
		}
		result.Record = changepreview.RecordOf(
			changepreview.ShipmentRecordEntity,
			comment.ShipmentID,
		)
	}

	return result
}

func toggleCommentRequest(
	params *agenttoolports.ToolExecuteParams,
) (*shipmentcommentports.ToggleShipmentCommentRequest, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	commentID, err := toolparams.RequirePulid(params.Params, ParamCommentID)
	if err != nil {
		return nil, err
	}

	return &shipmentcommentports.ToggleShipmentCommentRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, nil
}

func renderCommentToggle(
	verb string,
	fields []string,
) func(*shipmentcommentports.ToggleShipmentCommentRequest, *shipmentcommentports.ShipmentCommentChange) (*agentpreview.ToolPreview, error) {
	return func(
		_ *shipmentcommentports.ToggleShipmentCommentRequest,
		plan *shipmentcommentports.ShipmentCommentChange,
	) (*agentpreview.ToolPreview, error) {
		change, err := toolpreview.Changed(
			commentRecord(plan.Before),
			plan.Before,
			plan.After,
			toolpreview.Only(fields...),
			toolpreview.Volatile(fieldPinnedAt, changepreview.FieldResolvedAt),
			toolpreview.WithRefs(commentUserRefs),
		)
		if err != nil {
			return nil, err
		}

		return toolpreview.Build(fmt.Sprintf("Would %s the comment on the shipment.", verb),
			change), nil
	}
}

type commentToggleSpec struct {
	name        string
	description string
	operation   permission.Operation
	rationale   string
	verb        string
	action      string
	fields      []string
	pinned      *bool
	resolved    func(params map[string]any) bool
}

func newCommentToggleTool(
	comments commentModerator,
	spec *commentToggleSpec,
) agenttoolports.AgentTool {
	properties := map[string]any{
		toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty(
			"The shipment the comment is on",
		),
		ParamCommentID: commentIDProperty(),
	}
	if spec.resolved != nil {
		properties[ParamResolved] = toolparams.BooleanProperty(
			"True to mark the comment resolved, " +
				"false to reopen it. Defaults to true.",
		)
	}

	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        spec.name,
		Description: spec.description,
		Artifact:    changepreview.ShipmentRecordEntity,
		Resource:    permission.ResourceShipmentComment,
		Operation:   spec.operation,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale:   spec.rationale,
		Properties:  properties,
		Required:    []string{toolparams.ParamShipmentID, ParamCommentID},
		Target:      targetComment,
	}, toolspec.ReceivablePlan[*shipmentcommentports.ToggleShipmentCommentRequest, *shipmentcommentports.ShipmentCommentChange]{
		Request: toggleCommentRequest,
		Plan: func(
			ctx context.Context,
			req *shipmentcommentports.ToggleShipmentCommentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*shipmentcommentports.ShipmentCommentChange, error) {
			if spec.pinned != nil {
				return comments.PreviewPin(ctx, req, params.Actor, *spec.pinned)
			}

			return comments.PreviewResolve(ctx, req, params.Actor, spec.resolved(params.Params))
		},
		Refused: func(*shipmentcommentports.ToggleShipmentCommentRequest) string {
			return "Would " + spec.verb + " the comment on the shipment."
		},
		Render: renderCommentToggle(spec.verb, spec.fields),
		Run: func(
			ctx context.Context,
			req *shipmentcommentports.ToggleShipmentCommentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			var (
				saved *shipment.ShipmentComment
				err   error
			)
			switch {
			case spec.pinned != nil && *spec.pinned:
				saved, err = comments.Pin(ctx, req, params.Actor)
			case spec.pinned != nil:
				saved, err = comments.Unpin(ctx, req, params.Actor)
			case spec.resolved(params.Params):
				saved, err = comments.Resolve(ctx, req, params.Actor)
			default:
				saved, err = comments.Unresolve(ctx, req, params.Actor)
			}
			if err != nil {
				return nil, err
			}

			return commentResult(spec.action, saved), nil
		},
	})
}

func targetComment(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamCommentID, permission.ResourceShipmentComment)
}

func BoolPointer(value bool) *bool { return &value }

func NewPinShipmentCommentTool(comments commentModerator) agenttoolports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "shipment_pin_comment",
		description: "Pin a comment to the top of a shipment's thread so dispatch sees " +
			"it first. Use it for the note that says what the shipment is waiting on or how it " +
			"must be handled. A reply cannot be pinned, and a shipment keeps at most " +
			"twenty-five pinned.",
		operation: permission.OpPin,
		rationale: "Orders a shipment's own comment thread inside Trenova; unpinning " +
			"undoes it and nothing is sent.",
		verb:   "pin",
		action: "pinned",
		fields: []string{fieldPinnedAt, FieldPinnedByID},
		pinned: BoolPointer(true),
	})
}

func NewUnpinShipmentCommentTool(comments commentModerator) agenttoolports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "shipment_unpin_comment",
		description: "Take a pinned comment off the top of a shipment's thread once " +
			"what it said no longer applies.",
		operation: permission.OpUnpin,
		rationale: "Orders a shipment's own comment thread inside Trenova; pinning " +
			"again undoes it and nothing is sent.",
		verb:   "unpin",
		action: "unpinned",
		fields: []string{fieldPinnedAt, FieldPinnedByID},
		pinned: BoolPointer(false),
	})
}

func NewResolveShipmentCommentTool(comments commentModerator) agenttoolports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "shipment_resolve_comment",
		description: "Mark a comment on a shipment resolved once what it asked for is " +
			"done, or reopen one with resolved=false when it is not. Say what settled it " +
			"in a reply with shipment_add_comment first, so the thread shows why.",
		operation: permission.OpResolve,
		rationale: "Marks a shipment's own comment settled inside Trenova; reopening " +
			"undoes it and nothing is sent.",
		verb:   "resolve",
		action: "resolved",
		fields: []string{changepreview.FieldResolvedAt, changepreview.FieldResolvedByID},
		resolved: func(params map[string]any) bool {
			if raw, ok := params[ParamResolved]; ok {
				if value, isBool := raw.(bool); isBool {
					return value
				}
			}

			return true
		},
	})
}

type commentEdit struct {
	get         shipmentcommentports.GetShipmentCommentByIDRequest
	text        *string
	visibility  *shipment.CommentVisibility
	priority    *shipment.CommentPriority
	requiresAck *bool
}

func readCommentEdit(params *agenttoolports.ToolExecuteParams) (*commentEdit, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	commentID, err := toolparams.RequirePulid(params.Params, ParamCommentID)
	if err != nil {
		return nil, err
	}

	edit := &commentEdit{get: shipmentcommentports.GetShipmentCommentByIDRequest{
		CommentID:  commentID,
		ShipmentID: shipmentID,
		TenantInfo: toolparams.TenantFrom(*params),
	}}
	if _, given := params.Params[ParamCommentText]; given {
		text, textErr := toolparams.RequireBoundedText(
			params.Params,
			ParamCommentText,
			maxCommentChars,
		)
		if textErr != nil {
			return nil, textErr
		}
		edit.text = &text
	}
	visibility, given, err := toolparams.OptionalEnum(
		params.Params,
		FieldCommentVisib,
		agenttoolschema.CommentVisibilities.Values,
	)
	if err != nil {
		return nil, err
	}
	if given {
		edit.visibility = &visibility
	}
	priority, given, err := toolparams.OptionalEnum(
		params.Params,
		FieldPriority,
		agenttoolschema.CommentPriorities.Values,
	)
	if err != nil {
		return nil, err
	}
	if given {
		edit.priority = &priority
	}
	if edit.requiresAck, err = toolparams.OptionalBoolPointer(
		params.Params,
		fieldRequiresAck,
	); err != nil {
		return nil, err
	}
	if edit.text == nil && edit.visibility == nil && edit.priority == nil &&
		edit.requiresAck == nil {
		return nil, toolparams.ErrNothingToChange
	}

	return edit, nil
}

func (e *commentEdit) entity(
	current *shipment.ShipmentComment,
	actor *permissionports.RequestActor,
) *shipment.ShipmentComment {
	next := &shipment.ShipmentComment{
		ID:                     current.ID,
		BusinessUnitID:         current.BusinessUnitID,
		OrganizationID:         current.OrganizationID,
		ShipmentID:             current.ShipmentID,
		UserID:                 actor.UserID,
		Comment:                current.Comment,
		Body:                   current.Body,
		Type:                   current.Type,
		Visibility:             current.Visibility,
		Priority:               current.Priority,
		RequiresAcknowledgment: current.RequiresAcknowledgment,
		Version:                current.Version,
		MentionedUserIDs:       make([]pulid.ID, 0, len(current.MentionedUsers)),
		AttachmentDocumentIDs:  make([]pulid.ID, 0, len(current.Attachments)),
	}
	for _, mention := range current.MentionedUsers {
		if mention != nil {
			next.MentionedUserIDs = append(next.MentionedUserIDs, mention.MentionedUserID)
		}
	}
	for _, attachment := range current.Attachments {
		if attachment != nil {
			next.AttachmentDocumentIDs = append(next.AttachmentDocumentIDs, attachment.DocumentID)
		}
	}
	if e.text != nil {
		next.Comment = *e.text
		next.Body = nil
	}
	if e.visibility != nil {
		next.Visibility = *e.visibility
		next.Type = commentType(*e.visibility)
	}
	if e.priority != nil {
		next.Priority = *e.priority
	}
	if e.requiresAck != nil {
		next.RequiresAcknowledgment = *e.requiresAck
	}

	return next
}

func classifyCommentEditVisibility(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	return shipmentpreview.ClassifyCommentVisibility(params)
}

type commentEditPlan struct {
	change *shipmentcommentports.ShipmentCommentChange
}

func moderatorFor(
	ctx context.Context,
	permissions commentPermissions,
	current *shipment.ShipmentComment,
	actor *permissionports.RequestActor,
) (bool, error) {
	if current.UserID == actor.UserID {
		return false, nil
	}
	if permissions == nil {
		return false, nil
	}

	result, err := permissions.Check(
		ctx,
		actor.PermissionCheck(permission.ResourceShipmentComment, permission.OpManage),
	)
	if err != nil {
		return false, fmt.Errorf("authorize comment moderation: %w", err)
	}

	return result.Allowed, nil
}

func NewEditShipmentCommentTool(
	comments commentModerator,
	permissions commentPermissions,
) agenttoolports.AgentTool {
	base := toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "shipment_update_comment",
		SearchTerms: []string{"fix comment", "reword comment", "typo"},
		Artifact:    changepreview.ShipmentRecordEntity,
		Description: "Change the text, visibility, priority or acknowledgment request " +
			"of a comment on a shipment. Send only what changes. Your own comments and " +
			"those the person may moderate can be edited; a note widened past the " +
			"organization's staff is read by whoever it is widened to, so only do it when " +
			"asked.",
		Resource:    permission.ResourceShipmentComment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Edits a shipment's own comment inside Trenova; one made visible to " +
			"a customer or a driver is read by them, so its visibility argument decides.",
		Properties: map[string]any{
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty(
				"The shipment the comment is on",
			),
			ParamCommentID: commentIDProperty(),
			ParamCommentText: toolparams.StringProperty(
				"The new text, in full.",
				maxCommentChars,
			),
			FieldCommentVisib: agenttoolschema.Enum("Who sees it. Only widen it to a customer "+
				"or a driver when the person asked.", agenttoolschema.CommentVisibilities),
			FieldPriority: agenttoolschema.Enum("How urgently dispatch should read it.",
				agenttoolschema.CommentPriorities),
			fieldRequiresAck: toolparams.BooleanProperty("Whether the people it mentions must " +
				"acknowledge it."),
		},
		Required: []string{toolparams.ParamShipmentID, ParamCommentID},
		Target:   targetComment,
	}, toolspec.ReceivablePlan[*commentEdit, *commentEditPlan]{
		Request: readCommentEdit,
		Plan: func(
			ctx context.Context,
			edit *commentEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*commentEditPlan, error) {
			req, err := commentUpdateRequest(ctx, comments, permissions, edit, params.Actor)
			if err != nil {
				return nil, err
			}
			change, err := comments.PreviewUpdate(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return &commentEditPlan{change: change}, nil
		},
		Refused: func(*commentEdit) string { return "Would edit the comment on the shipment." },
		Render:  renderCommentEdit,
		Run: func(
			ctx context.Context,
			edit *commentEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := commentUpdateRequest(ctx, comments, permissions, edit, params.Actor)
			if err != nil {
				return nil, err
			}
			saved, err := comments.Update(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return commentResult("edited", saved), nil
		},
	})

	return toolspec.ClassifiedTool[*commentEdit, *commentEditPlan]{
		ReportingReceivableTool: base,
		Egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		Classify: classifyCommentEditVisibility,
	}
}

func renderCommentEdit(_ *commentEdit, plan *commentEditPlan) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(
		commentRecord(plan.change.Before),
		plan.change.Before,
		plan.change.After,
		toolpreview.Only(fieldCommentText, fieldCommentType, FieldCommentVisib,
			FieldPriority, fieldRequiresAck, fieldEditedAt),
		toolpreview.Volatile(fieldEditedAt),
	)
	if err != nil {
		return nil, err
	}
	summary := "Would edit the comment on the shipment."
	if audience := commentAudience(plan.change.After.Visibility); audience != "" &&
		plan.change.After.Visibility != plan.change.Before.Visibility {
		summary += " " + audience
	}

	return toolpreview.Build(summary, change), nil
}

func commentUpdateRequest(
	ctx context.Context,
	comments commentModerator,
	permissions commentPermissions,
	edit *commentEdit,
	actor *permissionports.RequestActor,
) (*shipmentcommentports.UpdateShipmentCommentRequest, error) {
	current, err := comments.GetByID(ctx, &edit.get)
	if err != nil {
		return nil, err
	}
	moderator, err := moderatorFor(ctx, permissions, current, actor)
	if err != nil {
		return nil, err
	}

	return &shipmentcommentports.UpdateShipmentCommentRequest{
		Entity:      edit.entity(current, actor),
		AsModerator: moderator,
	}, nil
}

type commentDeletion struct {
	get shipmentcommentports.GetShipmentCommentByIDRequest
}

func NewDeleteShipmentCommentTool(
	comments commentModerator,
	permissions commentPermissions,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_delete_comment",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Delete a comment from a shipment's thread: your own, or another " +
			"person's when the person you act for may moderate comments. One with replies " +
			"is left as a placeholder so the replies keep their place. Prefer " +
			"shipment_resolve_comment for a note that was right at the time.",
		Resource:    permission.ResourceShipmentComment,
		Operation:   permission.OpDelete,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Removes a note from a shipment's own thread inside Trenova; the text " +
			"is gone, so a person confirms it.",
		Properties: map[string]any{
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty(
				"The shipment the comment is on",
			),
			ParamCommentID: commentIDProperty(),
		},
		Required: []string{toolparams.ParamShipmentID, ParamCommentID},
		Target:   targetComment,
	}, toolspec.ReceivablePlan[*commentDeletion, *shipmentcommentports.ShipmentCommentDeletion]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*commentDeletion, error) {
			toggle, err := toggleCommentRequest(params)
			if err != nil {
				return nil, err
			}

			return &commentDeletion{get: shipmentcommentports.GetShipmentCommentByIDRequest{
				CommentID:  toggle.CommentID,
				ShipmentID: toggle.ShipmentID,
				TenantInfo: toggle.TenantInfo,
			}}, nil
		},
		Plan: func(
			ctx context.Context,
			deletion *commentDeletion,
			params *agenttoolports.ToolExecuteParams,
		) (*shipmentcommentports.ShipmentCommentDeletion, error) {
			req, err := commentDeleteRequest(ctx, comments, permissions, deletion, params.Actor)
			if err != nil {
				return nil, err
			}

			return comments.PreviewDelete(ctx, req, params.Actor)
		},
		Refused: func(*commentDeletion) string { return "Would delete the comment." },
		Render: func(
			_ *commentDeletion,
			plan *shipmentcommentports.ShipmentCommentDeletion,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Delete(commentRecord(plan.Comment), plan.Comment,
				toolpreview.Only(fieldCommentText, FieldCommentVisib, FieldPriority))
			if err != nil {
				return nil, err
			}
			summary := "Would delete the comment from the shipment's thread."
			if plan.Tombstoned {
				summary = "Would delete the comment's text, leaving a placeholder because " +
					"it has replies."
			}

			return toolpreview.Build(summary, change), nil
		},
		Run: func(
			ctx context.Context,
			deletion *commentDeletion,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := commentDeleteRequest(ctx, comments, permissions, deletion, params.Actor)
			if err != nil {
				return nil, err
			}
			if err = comments.Delete(ctx, req, params.Actor); err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "deleted",
				Kind:   kindComment,
				IDs: map[string]string{
					ParamCommentID:             deletion.get.CommentID.String(),
					toolparams.ParamShipmentID: deletion.get.ShipmentID.String(),
				},
			}, nil
		},
	})
}

func commentDeleteRequest(
	ctx context.Context,
	comments commentModerator,
	permissions commentPermissions,
	deletion *commentDeletion,
	actor *permissionports.RequestActor,
) (*shipmentcommentports.DeleteShipmentCommentRequest, error) {
	current, err := comments.GetByID(ctx, &deletion.get)
	if err != nil {
		return nil, err
	}
	moderator, err := moderatorFor(ctx, permissions, current, actor)
	if err != nil {
		return nil, err
	}

	return &shipmentcommentports.DeleteShipmentCommentRequest{
		TenantInfo:  deletion.get.TenantInfo,
		ShipmentID:  deletion.get.ShipmentID,
		CommentID:   deletion.get.CommentID,
		AsModerator: moderator,
	}, nil
}
