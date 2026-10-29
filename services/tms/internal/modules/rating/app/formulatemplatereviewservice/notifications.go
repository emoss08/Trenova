package formulatemplatereviewservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	notificationSource = "formula_template_service"
	dataLink           = "link"

	eventTemplateSubmitted         = "formula_template.submitted"
	eventTemplateReviewRequested   = "formula_template.review_requested"
	eventTemplateApproved          = "formula_template.approved"
	eventTemplateRejected          = "formula_template.rejected"
	eventTemplateChangesRequested  = "formula_template.changes_requested"
	eventTemplateSubmissionExpired = "formula_template.submission_expired"
)

func templateStudioLink(templateID pulid.ID) string {
	return "/billing/configuration-files/formula-templates/" + templateID.String() + "/edit"
}

// templateReviewLink opens the studio on the approve step, with the diff and
// impact already in view, so a reviewer lands on the decision rather than on
// the editor.
func templateReviewLink(templateID pulid.ID) string {
	return templateStudioLink(templateID) + "?review=approve"
}

// notifySubmitted raises a global in-app notification so reviewers learn a
// template is waiting on them without having to poll the list. Failures are
// logged only: a notification must never fail the transition it decorates.
func (s *Service) notifySubmitted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	template *formulatemplate.FormulaTemplate,
) {
	if s.notifications == nil {
		return
	}

	buID := tenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      eventTemplateSubmitted,
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          "Formula template submitted for review",
		Message: fmt.Sprintf(
			"%q was submitted for review and is waiting for an approver.",
			template.Name,
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: templateReviewLink(template.ID)},
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create submission notification",
			zap.Error(err), zap.String("templateID", template.ID.String()))
	}
}

func (s *Service) notifyRequestedReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	template *formulatemplate.FormulaTemplate,
	reviewerID pulid.ID,
) {
	if s.notifications == nil || reviewerID.IsNil() || reviewerID == tenantInfo.UserID {
		return
	}

	buID := tenantInfo.BuID
	target := reviewerID
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &target,
		EventType:      eventTemplateReviewRequested,
		Priority:       notification.PriorityHigh,
		Channel:        notification.ChannelUser,
		Title:          "Your review is requested",
		Message: fmt.Sprintf(
			"%q was submitted for review and you were asked to approve it.",
			template.Name,
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: templateReviewLink(template.ID)},
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to notify the requested reviewer",
			zap.Error(err), zap.String("templateID", template.ID.String()))
	}
}

type reviewOutcome struct {
	Template       *formulatemplate.FormulaTemplate
	SubmitterID    *pulid.ID
	Decision       formulatemplate.ReviewDecision
	Comment        string
	BillingVersion int64
}

// notifyReviewOutcome tells the submitter their template was approved or sent
// back. The reviewer acting on their own submission is never notified — the
// approve path forbids it and the reject path would be telling them what they
// just did.
func (s *Service) notifyReviewOutcome(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	outcome *reviewOutcome,
) {
	if s.notifications == nil || outcome.SubmitterID == nil || outcome.SubmitterID.IsNil() {
		return
	}
	if *outcome.SubmitterID == tenantInfo.UserID {
		return
	}

	title, message, eventType, priority := describeReviewOutcome(outcome)

	buID := tenantInfo.BuID
	submitterID := *outcome.SubmitterID
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &submitterID,
		EventType:      eventType,
		Priority:       priority,
		Channel:        notification.ChannelUser,
		Title:          title,
		Message:        message,
		Source:         notificationSource,
		Data:           map[string]any{dataLink: templateStudioLink(outcome.Template.ID)},
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create review outcome notification",
			zap.Error(err), zap.String("templateID", outcome.Template.ID.String()))
	}
}

// describeReviewOutcome words each decision for the author who submitted.
func describeReviewOutcome(
	outcome *reviewOutcome,
) (title, message, eventType string, priority notification.Priority) {
	name := outcome.Template.Name
	switch outcome.Decision {
	case formulatemplate.ReviewDecisionRejected:
		if outcome.BillingVersion > 0 {
			return "Formula template revision rejected",
				fmt.Sprintf(
					"The revision of %q was rejected and returned to draft: %s v%d keeps billing until a revision is approved.",
					name,
					outcome.Comment,
					outcome.BillingVersion,
				),
				eventTemplateRejected,
				notification.PriorityHigh
		}
		return "Formula template rejected",
			fmt.Sprintf(
				"%q was rejected and archived: %s Edit and resubmit it from the archive if it should be reconsidered.",
				name,
				outcome.Comment,
			),
			eventTemplateRejected,
			notification.PriorityHigh
	case formulatemplate.ReviewDecisionChangesRequested:
		return "Changes requested on formula template",
			fmt.Sprintf(
				"%q needs changes before it can be approved: %s Edit it and resubmit to continue the same review.",
				name,
				outcome.Comment,
			),
			eventTemplateChangesRequested,
			notification.PriorityHigh
	case formulatemplate.ReviewDecisionExpired:
		return "Formula template submission expired",
			fmt.Sprintf(
				"%q waited more than 14 days for a decision and was returned to draft. Resubmit it when the rates are current.",
				name,
			),
			eventTemplateSubmissionExpired,
			notification.PriorityMedium
	case formulatemplate.ReviewDecisionSubmitted, formulatemplate.ReviewDecisionApproved:
		return describeApproval(outcome)
	default:
		return describeApproval(outcome)
	}
}

func describeApproval(
	outcome *reviewOutcome,
) (title, message, eventType string, priority notification.Priority) {
	message = fmt.Sprintf("%q was approved and is now active.", outcome.Template.Name)
	if outcome.Comment != "" {
		message += " Reviewer note: " + outcome.Comment
	}
	return "Formula template approved", message, eventTemplateApproved, notification.PriorityMedium
}
