package formulalibraryservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	notificationSource = "formula_library_service"
	dataLink           = "link"
	titleSubmitted     = "Formula library submitted for review"

	eventLibrarySubmitted         = "formula_library.submitted"
	eventLibraryReviewRequested   = "formula_library.review_requested"
	eventLibraryApproved          = "formula_library.approved"
	eventLibraryRejected          = "formula_library.rejected"
	eventLibraryChangesRequested  = "formula_library.changes_requested"
	eventLibrarySubmissionExpired = "formula_library.submission_expired"
)

func libraryLink(libraryID pulid.ID) string {
	return "/billing/configuration-files/formula-libraries/" + libraryID.String() + "/edit"
}

func libraryReviewLink(libraryID pulid.ID) string {
	return libraryLink(libraryID) + "?review=approve"
}

func (s *Service) notify(
	ctx context.Context,
	entity *notification.Notification,
	libraryID pulid.ID,
) {
	if s.notifications == nil {
		return
	}
	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create formula library notification",
			zap.String("event", entity.EventType),
			zap.String("libraryID", libraryID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) notifySubmitted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	library *formulalibrary.FormulaLibrary,
) {
	buID := tenantInfo.BuID
	s.notify(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      eventLibrarySubmitted,
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          titleSubmitted,
		Message: fmt.Sprintf(
			"%s was submitted for review and is waiting for an approver.",
			library.Path(),
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: libraryReviewLink(library.ID)},
	}, library.ID)
}

func (s *Service) notifyRequestedReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	library *formulalibrary.FormulaLibrary,
	reviewerID pulid.ID,
) {
	if reviewerID.IsNil() || reviewerID == tenantInfo.UserID {
		return
	}
	buID := tenantInfo.BuID
	target := reviewerID
	s.notify(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &target,
		EventType:      eventLibraryReviewRequested,
		Priority:       notification.PriorityHigh,
		Channel:        notification.ChannelUser,
		Title:          "Your review is requested",
		Message: fmt.Sprintf(
			"%s was submitted for review and you were asked to approve it.",
			library.Path(),
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: libraryReviewLink(library.ID)},
	}, library.ID)
}

type reviewOutcome struct {
	Library         *formulalibrary.FormulaLibrary
	SubmitterID     *pulid.ID
	Decision        formulatemplate.ReviewDecision
	Comment         string
	ApprovedVersion int64
}

func (s *Service) notifyReviewOutcome(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	outcome *reviewOutcome,
) {
	if outcome.Library == nil || outcome.SubmitterID == nil || outcome.SubmitterID.IsNil() ||
		*outcome.SubmitterID == tenantInfo.UserID {
		return
	}

	title, message, eventType, priority := describeOutcome(outcome)
	buID := tenantInfo.BuID
	submitter := *outcome.SubmitterID
	s.notify(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &submitter,
		EventType:      eventType,
		Priority:       priority,
		Channel:        notification.ChannelUser,
		Title:          title,
		Message:        message,
		Source:         notificationSource,
		Data:           map[string]any{dataLink: libraryLink(outcome.Library.ID)},
	}, outcome.Library.ID)
}

func describeOutcome(
	outcome *reviewOutcome,
) (title, message, eventType string, priority notification.Priority) {
	path := outcome.Library.Path()
	switch outcome.Decision {
	case formulatemplate.ReviewDecisionRejected:
		if outcome.ApprovedVersion > 0 {
			return "Formula library revision rejected",
				fmt.Sprintf(
					"The revision of %s was rejected and returned to draft: %s Templates keep importing v%d until a revision is approved.",
					path,
					outcome.Comment,
					outcome.ApprovedVersion,
				),
				eventLibraryRejected, notification.PriorityHigh
		}
		return "Formula library rejected",
			fmt.Sprintf("%s was rejected and archived: %s", path, outcome.Comment),
			eventLibraryRejected, notification.PriorityHigh
	case formulatemplate.ReviewDecisionChangesRequested:
		return "Changes requested on formula library",
			fmt.Sprintf(
				"%s needs changes before it can be approved: %s Edit it and resubmit to continue the same review.",
				path,
				outcome.Comment,
			),
			eventLibraryChangesRequested, notification.PriorityHigh
	case formulatemplate.ReviewDecisionExpired:
		return "Formula library submission expired",
			fmt.Sprintf(
				"%s waited more than 14 days for a decision and was returned to draft.",
				path,
			),
			eventLibrarySubmissionExpired, notification.PriorityMedium
	case formulatemplate.ReviewDecisionSubmitted:
		return titleSubmitted,
			fmt.Sprintf("%s was submitted for review.", path),
			eventLibrarySubmitted, notification.PriorityMedium
	case formulatemplate.ReviewDecisionApproved:
		message = fmt.Sprintf(
			"%s was approved. Templates pin this version the next time they are approved.",
			path,
		)
		if outcome.Comment != "" {
			message += " Reviewer note: " + outcome.Comment
		}
		return "Formula library approved", message, eventLibraryApproved, notification.PriorityMedium
	}
	return "Formula library reviewed", path, eventLibraryApproved, notification.PriorityMedium
}
