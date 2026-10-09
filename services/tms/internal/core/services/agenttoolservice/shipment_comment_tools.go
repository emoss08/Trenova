package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramCommentID    = "commentId"
	paramCommentText  = "comment"
	paramResolved     = "resolved"
	kindComment       = "shipment comment"
	maxCommentChars   = 4000
	fieldPinnedAt     = "pinnedAt"
	fieldPinnedByID   = "pinnedById"
	fieldEditedAt     = "editedAt"
	fieldPriority     = "priority"
	fieldCommentText  = "comment"
	fieldDeletedByID  = "deletedById"
	fieldRequiresAck  = "requiresAcknowledgment"
	fieldCommentType  = "type"
	fieldCommentVisib = "visibility"
)

var commentUserRefs = map[string]permission.Resource{
	fieldPinnedByID:   permission.ResourceUser,
	fieldResolvedByID: permission.ResourceUser,
	fieldDeletedByID:  permission.ResourceUser,
}

type commentModerator interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetShipmentCommentByIDRequest,
	) (*shipment.ShipmentComment, error)
	Pin(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unpin(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Resolve(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unresolve(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		req *serviceports.UpdateShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *serviceports.DeleteShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) error
	PreviewPin(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
		pinned bool,
	) (*serviceports.ShipmentCommentChange, error)
	PreviewResolve(
		ctx context.Context,
		req *serviceports.ToggleShipmentCommentRequest,
		actor *serviceports.RequestActor,
		resolved bool,
	) (*serviceports.ShipmentCommentChange, error)
	PreviewUpdate(
		ctx context.Context,
		req *serviceports.UpdateShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ShipmentCommentChange, error)
	PreviewDelete(
		ctx context.Context,
		req *serviceports.DeleteShipmentCommentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ShipmentCommentDeletion, error)
}

type commentPermissions interface {
	Check(
		ctx context.Context,
		req *serviceports.PermissionCheckRequest,
	) (*serviceports.PermissionCheckResult, error)
}

func commentIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceShipmentComment, "The comment",
		"the recentComments get_shipment lists or the page you are on")
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
		Version:  previewVersion(comment.Version),
	}
}

func commentResult(action string, comment *shipment.ShipmentComment) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindComment}
	if comment != nil {
		result.IDs = map[string]string{
			paramCommentID:  comment.ID.String(),
			paramShipmentID: comment.ShipmentID.String(),
		}
		result.Record = recordOf(shipmentRecordEntity, comment.ShipmentID)
	}

	return result
}

func toggleCommentRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ToggleShipmentCommentRequest, error) {
	shipmentID, err := requirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}
	commentID, err := requirePulid(params.Params, paramCommentID)
	if err != nil {
		return nil, err
	}

	return &serviceports.ToggleShipmentCommentRequest{
		TenantInfo: tenantFrom(*params),
		ShipmentID: shipmentID,
		CommentID:  commentID,
	}, nil
}

func renderCommentToggle(
	verb string,
	fields []string,
) func(*serviceports.ToggleShipmentCommentRequest, *serviceports.ShipmentCommentChange) (*agent.ToolPreview, error) {
	return func(
		_ *serviceports.ToggleShipmentCommentRequest,
		plan *serviceports.ShipmentCommentChange,
	) (*agent.ToolPreview, error) {
		change, err := toolpreview.Changed(
			commentRecord(plan.Before),
			plan.Before,
			plan.After,
			toolpreview.Only(fields...),
			toolpreview.Volatile(fieldPinnedAt, fieldResolvedAt),
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
) serviceports.AgentTool {
	properties := map[string]any{
		paramShipmentID: shipmentIDProperty("The shipment the comment is on"),
		paramCommentID:  commentIDProperty(),
	}
	if spec.resolved != nil {
		properties[paramResolved] = booleanProperty("True to mark the comment resolved, " +
			"false to reopen it. Defaults to true.")
	}

	return newReportingReceivableTool(&receivableSpec{
		name:        spec.name,
		description: spec.description,
		artifact:    shipmentRecordEntity,
		resource:    permission.ResourceShipmentComment,
		operation:   spec.operation,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale:   spec.rationale,
		properties:  properties,
		required:    []string{paramShipmentID, paramCommentID},
		target:      targetComment,
	}, receivablePlan[*serviceports.ToggleShipmentCommentRequest, *serviceports.ShipmentCommentChange]{
		request: toggleCommentRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ToggleShipmentCommentRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentCommentChange, error) {
			if spec.pinned != nil {
				return comments.PreviewPin(ctx, req, params.Actor, *spec.pinned)
			}

			return comments.PreviewResolve(ctx, req, params.Actor, spec.resolved(params.Params))
		},
		refused: func(*serviceports.ToggleShipmentCommentRequest) string {
			return "Would " + spec.verb + " the comment on the shipment."
		},
		render: renderCommentToggle(spec.verb, spec.fields),
		run: func(
			ctx context.Context,
			req *serviceports.ToggleShipmentCommentRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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

func targetComment(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramCommentID, permission.ResourceShipmentComment)
}

func boolPointer(value bool) *bool { return &value }

func newPinShipmentCommentTool(comments commentModerator) serviceports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "pin_shipment_comment",
		description: "Pin a comment to the top of a shipment's thread so dispatch sees " +
			"it first. Use it for the note that says what the shipment is waiting on or how it " +
			"must be handled. A reply cannot be pinned, and a shipment keeps at most " +
			"twenty-five pinned.",
		operation: permission.OpPin,
		rationale: "Orders a shipment's own comment thread inside Trenova; unpinning " +
			"undoes it and nothing is sent.",
		verb:   "pin",
		action: "pinned",
		fields: []string{fieldPinnedAt, fieldPinnedByID},
		pinned: boolPointer(true),
	})
}

func newUnpinShipmentCommentTool(comments commentModerator) serviceports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "unpin_shipment_comment",
		description: "Take a pinned comment off the top of a shipment's thread once " +
			"what it said no longer applies.",
		operation: permission.OpUnpin,
		rationale: "Orders a shipment's own comment thread inside Trenova; pinning " +
			"again undoes it and nothing is sent.",
		verb:   "unpin",
		action: "unpinned",
		fields: []string{fieldPinnedAt, fieldPinnedByID},
		pinned: boolPointer(false),
	})
}

func newResolveShipmentCommentTool(comments commentModerator) serviceports.AgentTool {
	return newCommentToggleTool(comments, &commentToggleSpec{
		name: "resolve_shipment_comment",
		description: "Mark a comment on a shipment resolved once what it asked for is " +
			"done, or reopen one with resolved=false when it is not. Say what settled it " +
			"in a reply with add_shipment_comment first, so the thread shows why.",
		operation: permission.OpResolve,
		rationale: "Marks a shipment's own comment settled inside Trenova; reopening " +
			"undoes it and nothing is sent.",
		verb:   "resolve",
		action: "resolved",
		fields: []string{fieldResolvedAt, fieldResolvedByID},
		resolved: func(params map[string]any) bool {
			if raw, ok := params[paramResolved]; ok {
				if value, isBool := raw.(bool); isBool {
					return value
				}
			}

			return true
		},
	})
}

type commentEdit struct {
	get         repositories.GetShipmentCommentByIDRequest
	text        *string
	visibility  *shipment.CommentVisibility
	priority    *shipment.CommentPriority
	requiresAck *bool
}

func readCommentEdit(params *serviceports.ToolExecuteParams) (*commentEdit, error) {
	shipmentID, err := requirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}
	commentID, err := requirePulid(params.Params, paramCommentID)
	if err != nil {
		return nil, err
	}

	edit := &commentEdit{get: repositories.GetShipmentCommentByIDRequest{
		CommentID:  commentID,
		ShipmentID: shipmentID,
		TenantInfo: tenantFrom(*params),
	}}
	if _, given := params.Params[paramCommentText]; given {
		text, textErr := requireBoundedText(params.Params, paramCommentText, maxCommentChars)
		if textErr != nil {
			return nil, textErr
		}
		edit.text = &text
	}
	visibility, given, err := optionalEnum(
		params.Params,
		fieldCommentVisib,
		agenttoolschema.CommentVisibilities.Values,
	)
	if err != nil {
		return nil, err
	}
	if given {
		edit.visibility = &visibility
	}
	priority, given, err := optionalEnum(
		params.Params,
		fieldPriority,
		agenttoolschema.CommentPriorities.Values,
	)
	if err != nil {
		return nil, err
	}
	if given {
		edit.priority = &priority
	}
	if edit.requiresAck, err = optionalBoolPointer(params.Params, fieldRequiresAck); err != nil {
		return nil, err
	}
	if edit.text == nil && edit.visibility == nil && edit.priority == nil &&
		edit.requiresAck == nil {
		return nil, errNothingToChange
	}

	return edit, nil
}

func (e *commentEdit) entity(
	current *shipment.ShipmentComment,
	actor *serviceports.RequestActor,
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
	params serviceports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) serviceports.CallPolicy {
	return classifyCommentVisibility(params)
}

type commentEditPlan struct {
	change *serviceports.ShipmentCommentChange
}

func moderatorFor(
	ctx context.Context,
	permissions commentPermissions,
	current *shipment.ShipmentComment,
	actor *serviceports.RequestActor,
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

func newEditShipmentCommentTool(
	comments commentModerator,
	permissions commentPermissions,
) serviceports.AgentTool {
	base := newReportingReceivableTool(&receivableSpec{
		name:        "edit_shipment_comment",
		searchTerms: []string{"fix comment", "reword comment", "typo"},
		artifact:    shipmentRecordEntity,
		description: "Change the text, visibility, priority or acknowledgment request " +
			"of a comment on a shipment. Send only what changes. Your own comments and " +
			"those the person may moderate can be edited; a note widened past the " +
			"organization's staff is read by whoever it is widened to, so only do it when " +
			"asked.",
		resource:    permission.ResourceShipmentComment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Edits a shipment's own comment inside Trenova; one made visible to " +
			"a customer or a driver is read by them, so its visibility argument decides.",
		properties: map[string]any{
			paramShipmentID:  shipmentIDProperty("The shipment the comment is on"),
			paramCommentID:   commentIDProperty(),
			paramCommentText: stringProperty("The new text, in full.", maxCommentChars),
			fieldCommentVisib: agenttoolschema.Enum("Who sees it. Only widen it to a customer "+
				"or a driver when the person asked.", agenttoolschema.CommentVisibilities),
			fieldPriority: agenttoolschema.Enum("How urgently dispatch should read it.",
				agenttoolschema.CommentPriorities),
			fieldRequiresAck: booleanProperty("Whether the people it mentions must " +
				"acknowledge it."),
		},
		required: []string{paramShipmentID, paramCommentID},
		target:   targetComment,
	}, receivablePlan[*commentEdit, *commentEditPlan]{
		request: readCommentEdit,
		plan: func(
			ctx context.Context,
			edit *commentEdit,
			params *serviceports.ToolExecuteParams,
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
		refused: func(*commentEdit) string { return "Would edit the comment on the shipment." },
		render: func(_ *commentEdit, plan *commentEditPlan) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				commentRecord(plan.change.Before),
				plan.change.Before,
				plan.change.After,
				toolpreview.Only(fieldCommentText, fieldCommentType, fieldCommentVisib,
					fieldPriority, fieldRequiresAck, fieldEditedAt),
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
		},
		run: func(
			ctx context.Context,
			edit *commentEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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

	return classifiedTool[*commentEdit, *commentEditPlan]{
		reportingReceivableTool: base,
		egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		classify: classifyCommentEditVisibility,
	}
}

func commentUpdateRequest(
	ctx context.Context,
	comments commentModerator,
	permissions commentPermissions,
	edit *commentEdit,
	actor *serviceports.RequestActor,
) (*serviceports.UpdateShipmentCommentRequest, error) {
	current, err := comments.GetByID(ctx, &edit.get)
	if err != nil {
		return nil, err
	}
	moderator, err := moderatorFor(ctx, permissions, current, actor)
	if err != nil {
		return nil, err
	}

	return &serviceports.UpdateShipmentCommentRequest{
		Entity:      edit.entity(current, actor),
		AsModerator: moderator,
	}, nil
}

type commentDeletion struct {
	get repositories.GetShipmentCommentByIDRequest
}

func newDeleteShipmentCommentTool(
	comments commentModerator,
	permissions commentPermissions,
) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "delete_shipment_comment",
		artifact: shipmentRecordEntity,
		description: "Delete a comment from a shipment's thread: your own, or another " +
			"person's when the person you act for may moderate comments. One with replies " +
			"is left as a placeholder so the replies keep their place. Prefer " +
			"resolve_shipment_comment for a note that was right at the time.",
		resource:    permission.ResourceShipmentComment,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Removes a note from a shipment's own thread inside Trenova; the text " +
			"is gone, so a person confirms it.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment the comment is on"),
			paramCommentID:  commentIDProperty(),
		},
		required: []string{paramShipmentID, paramCommentID},
		target:   targetComment,
	}, receivablePlan[*commentDeletion, *serviceports.ShipmentCommentDeletion]{
		request: func(params *serviceports.ToolExecuteParams) (*commentDeletion, error) {
			toggle, err := toggleCommentRequest(params)
			if err != nil {
				return nil, err
			}

			return &commentDeletion{get: repositories.GetShipmentCommentByIDRequest{
				CommentID:  toggle.CommentID,
				ShipmentID: toggle.ShipmentID,
				TenantInfo: toggle.TenantInfo,
			}}, nil
		},
		plan: func(
			ctx context.Context,
			deletion *commentDeletion,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentCommentDeletion, error) {
			req, err := commentDeleteRequest(ctx, comments, permissions, deletion, params.Actor)
			if err != nil {
				return nil, err
			}

			return comments.PreviewDelete(ctx, req, params.Actor)
		},
		refused: func(*commentDeletion) string { return "Would delete the comment." },
		render: func(
			_ *commentDeletion,
			plan *serviceports.ShipmentCommentDeletion,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(commentRecord(plan.Comment), plan.Comment,
				toolpreview.Only(fieldCommentText, fieldCommentVisib, fieldPriority))
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
		run: func(
			ctx context.Context,
			deletion *commentDeletion,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := commentDeleteRequest(ctx, comments, permissions, deletion, params.Actor)
			if err != nil {
				return nil, err
			}
			if err = comments.Delete(ctx, req, params.Actor); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "deleted",
				Kind:   kindComment,
				IDs: map[string]string{
					paramCommentID:  deletion.get.CommentID.String(),
					paramShipmentID: deletion.get.ShipmentID.String(),
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
	actor *serviceports.RequestActor,
) (*serviceports.DeleteShipmentCommentRequest, error) {
	current, err := comments.GetByID(ctx, &deletion.get)
	if err != nil {
		return nil, err
	}
	moderator, err := moderatorFor(ctx, permissions, current, actor)
	if err != nil {
		return nil, err
	}

	return &serviceports.DeleteShipmentCommentRequest{
		TenantInfo:  deletion.get.TenantInfo,
		ShipmentID:  deletion.get.ShipmentID,
		CommentID:   deletion.get.CommentID,
		AsModerator: moderator,
	}, nil
}
