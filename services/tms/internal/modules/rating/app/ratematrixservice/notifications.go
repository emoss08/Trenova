package ratematrixservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	notificationSource = "rate_matrix_service"
	dataLink           = "link"
	titleSubmitted     = "Rate matrix submitted for review"

	eventSubmitted         = "rate_matrix.submitted"
	eventReviewRequested   = "rate_matrix.review_requested"
	eventPublished         = "rate_matrix.published"
	eventRejected          = "rate_matrix.rejected"
	eventChangesRequested  = "rate_matrix.changes_requested"
	eventSubmissionExpired = "rate_matrix.submission_expired"
)

func matrixLink(matrixID pulid.ID) string {
	return "/billing/configuration-files/rate-matrices?panelType=edit&panelEntityId=" + matrixID.String()
}

func (s *Service) notify(
	ctx context.Context,
	entity *notification.Notification,
	matrixID pulid.ID,
) {
	if s.notifications == nil {
		return
	}
	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create rate matrix notification",
			zap.String("event", entity.EventType),
			zap.String("rateMatrixID", matrixID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) notifySubmitted(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matrix *ratematrix.RateMatrix,
) {
	buID := tenantInfo.BuID
	s.notify(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      eventSubmitted,
		Priority:       notification.PriorityMedium,
		Channel:        notification.ChannelGlobal,
		Title:          titleSubmitted,
		Message: fmt.Sprintf(
			"The draft of rate matrix %s was submitted for review and is waiting for a publisher.",
			matrix.Code,
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: matrixLink(matrix.ID)},
	}, matrix.ID)
}

func (s *Service) notifyRequestedReviewer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	matrix *ratematrix.RateMatrix,
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
		EventType:      eventReviewRequested,
		Priority:       notification.PriorityHigh,
		Channel:        notification.ChannelUser,
		Title:          "Your review is requested",
		Message: fmt.Sprintf(
			"The draft of rate matrix %s was submitted and you were asked to publish it.",
			matrix.Code,
		),
		Source: notificationSource,
		Data:   map[string]any{dataLink: matrixLink(matrix.ID)},
	}, matrix.ID)
}

type reviewOutcome struct {
	Matrix           *ratematrix.RateMatrix
	SubmitterID      *pulid.ID
	Decision         formulatemplate.ReviewDecision
	Comment          string
	PublishedVersion int64
}

func (s *Service) notifyReviewOutcome(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	outcome *reviewOutcome,
) {
	if outcome.Matrix == nil || outcome.SubmitterID == nil || outcome.SubmitterID.IsNil() ||
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
		Data:           map[string]any{dataLink: matrixLink(outcome.Matrix.ID)},
	}, outcome.Matrix.ID)
}

func describeOutcome(
	outcome *reviewOutcome,
) (title, message, eventType string, priority notification.Priority) {
	code := outcome.Matrix.Code
	switch outcome.Decision {
	case formulatemplate.ReviewDecisionRejected:
		if outcome.PublishedVersion > 0 {
			return "Rate matrix draft rejected",
				fmt.Sprintf(
					"The draft of %s was rejected and returned to draft: %s Tariffs keep reading v%d.",
					code,
					outcome.Comment,
					outcome.PublishedVersion,
				),
				eventRejected, notification.PriorityHigh
		}
		return "Rate matrix rejected",
			fmt.Sprintf("%s was rejected and archived: %s", code, outcome.Comment),
			eventRejected, notification.PriorityHigh
	case formulatemplate.ReviewDecisionChangesRequested:
		return "Changes requested on rate matrix",
			fmt.Sprintf(
				"The draft of %s needs changes before it can be published: %s",
				code,
				outcome.Comment,
			),
			eventChangesRequested, notification.PriorityHigh
	case formulatemplate.ReviewDecisionExpired:
		return "Rate matrix submission expired",
			fmt.Sprintf(
				"The draft of %s waited more than 14 days for a decision and was returned to draft.",
				code,
			),
			eventSubmissionExpired, notification.PriorityMedium
	case formulatemplate.ReviewDecisionSubmitted:
		return titleSubmitted,
			fmt.Sprintf("The draft of %s was submitted for review.", code),
			eventSubmitted, notification.PriorityMedium
	case formulatemplate.ReviewDecisionApproved:
		message = fmt.Sprintf(
			"%s v%d was published. Formula templates read it once a new template version is approved.",
			code,
			outcome.Matrix.PublishedVersionNumber,
		)
		if outcome.Comment != "" {
			message += " Reviewer note: " + outcome.Comment
		}
		return "Rate matrix published", message, eventPublished, notification.PriorityMedium
	}
	return "Rate matrix reviewed", code, eventPublished, notification.PriorityMedium
}
