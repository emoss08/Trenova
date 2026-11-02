package detentionnoticeservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
)

// NoticeHTMLForTest exposes the send-path HTML reconstruction so the escaping
// guarantee can be asserted without standing up an email provider.
func NoticeHTMLForTest(notice *detentionnotice.DetentionNotice) string {
	return noticeHTML(notice)
}
