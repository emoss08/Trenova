package detentionnoticeservice

import (
	"html"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ScheduleNoticeParams describes an outbound notice to queue.
type ScheduleNoticeParams struct {
	Occurrence   *detention.DetentionOccurrence
	Kind         detentionnotice.NoticeKind
	ScheduledFor int64
	Recipients   []string
	Automatic    bool
	SentByID     *pulid.ID
	FacilityName string
	ShipmentRef  string
	AttachPDF    bool
}

// ScheduledNotice is a queued notice plus the artifact the row cannot hold.
//
// The PDF travels in memory from the render to the send rather than being
// re-rendered there, because the notice body is a snapshot and an attachment
// rendered later could quote figures the snapshot does not.
type ScheduledNotice struct {
	Notice *detentionnotice.DetentionNotice
	PDF    *detentionnoticerender.NoticePDF
}

// SendNoticeParams is a queued notice, the occurrence it belongs to, and the
// optional PDF rendered alongside its body.
type SendNoticeParams struct {
	Notice     *detentionnotice.DetentionNotice
	Occurrence *detention.DetentionOccurrence
	PDF        *detentionnoticerender.NoticePDF
}

// noticeAttachments hands the rendered PDF to the email service, which stores it
// and references it by object key rather than carrying bytes through the send
// workflow.
func noticeAttachments(pdf *detentionnoticerender.NoticePDF) []emailports.EmailAttachment {
	if pdf == nil || len(pdf.Content) == 0 {
		return nil
	}

	return []emailports.EmailAttachment{
		{
			FileName:    pdf.FileName,
			ContentType: "application/pdf",
			Content:     pdf.Content,
			SizeBytes:   int64(len(pdf.Content)),
		},
	}
}

func noticeOutcomeVerb(notice *detentionnotice.DetentionNotice) string {
	switch {
	case notice.DeliveryStatus.IsFailure():
		return "failed to send"
	case notice.SatisfiesRequirement:
		return "sent within the contractual window"
	default:
		return "sent after the contractual deadline"
	}
}

type SendOccurrenceNoticeParams struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	Automatic    bool

	// Policy is the governing policy when the caller already has it. The sweep
	// does — it reads the policy to decide whether to send at all — so passing it
	// here saves re-reading the same row for every occurrence it sends.
	Policy *detentionpolicy.DetentionPolicy
}

// noticeKindFor picks the message the moment calls for: a warning while free
// time is still running, a start-of-charges notice once it lapses, an update on
// a repeat send, and a final summary after the clock has stopped.
func noticeKindFor(
	occurrence *detention.DetentionOccurrence,
	now int64,
) detentionnotice.NoticeKind {
	switch {
	case occurrence.ClockStopAt != nil:
		return detentionnotice.NoticeKindFinal
	case now < occurrence.FreeTimeExpiresAt:
		return detentionnotice.NoticeKindWarning
	case occurrence.NoticeSentAt == nil:
		return detentionnotice.NoticeKindStarted
	default:
		return detentionnotice.NoticeKindUpdate
	}
}

// noticeHTML returns the HTML snapshotted when the notice was built.
//
// It never constructs markup from Body. The previous implementation wrapped the
// plain text in <pre> at send time, which meant any angle bracket in a facility
// or customer name was interpolated into the email unescaped.
func noticeHTML(notice *detentionnotice.DetentionNotice) string {
	if strings.TrimSpace(notice.BodyHTML) != "" {
		return notice.BodyHTML
	}

	// A notice queued before this column existed has no snapshot. Escaping the
	// stored text is the safe reconstruction; it renders as plain text rather
	// than as whatever the text happens to look like.
	return "<pre style=\"font-family:inherit;white-space:pre-wrap\">" +
		html.EscapeString(notice.Body) + "</pre>"
}

// NoticeHTMLForTest exposes the send-path HTML reconstruction so the escaping
// guarantee can be asserted without standing up an email provider.
func NoticeHTMLForTest(notice *detentionnotice.DetentionNotice) string {
	return noticeHTML(notice)
}
