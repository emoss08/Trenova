package detentionnoticeservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
)

// NoticePreview is the notice SendOccurrenceNotice would send now: rendered
// by the same template, to the same recipients, and the occurrence as the
// send would leave it.
type NoticePreview struct {
	Before     *detention.DetentionOccurrence
	After      *detention.DetentionOccurrence
	Kind       detentionnotice.NoticeKind
	Recipients []string
	Content    detentionnoticerender.NoticeContent
	Attachment string
	Sender     *emailports.EmailSender
}

type occurrenceNoticePlan struct {
	schedule ScheduleNoticeParams
	now      int64
}

// applyNoticeOutcome moves the occurrence's notification status to what a
// notice's delivery says happened.
func applyNoticeOutcome(
	occurrence *detention.DetentionOccurrence,
	notice *detentionnotice.DetentionNotice,
) {
	switch {
	case notice.DeliveryStatus.IsFailure():
		occurrence.NotificationStatus = detention.NotificationStatusFailed
	case notice.SatisfiesRequirement:
		occurrence.NotificationStatus = detention.NotificationStatusSent
		occurrence.NoticeSentAt = notice.SentAt
	default:
		occurrence.NotificationStatus = detention.NotificationStatusLate
		occurrence.NoticeSentAt = notice.SentAt
	}
}
