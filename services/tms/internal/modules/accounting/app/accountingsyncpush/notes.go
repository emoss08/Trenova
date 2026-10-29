package accountingsyncpush

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/timeutils"
)

func GLAccountLabel(code, name string) string {
	if full := GLAccountName(code, name); full != "" {
		return "GL account " + full
	}
	return ""
}

func GLAccountName(code, name string) string {
	return strings.TrimSpace(strings.TrimSpace(code) + " " + strings.TrimSpace(name))
}

const NotePrefix = "Trenova "

func SentDateNote(
	record *accountingsync.AccountingSyncRecord,
	documentDate int64,
	loc *time.Location,
) string {
	if record.RedatedTo == nil {
		return ""
	}
	return " Dated " + timeutils.FormatCalendarDate(documentDate, loc) +
		" in Trenova; sent on the first open day of the books."
}
