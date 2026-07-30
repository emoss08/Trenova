package reportjobs

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

// digestNote tells the reader the table is a sample rather than the whole
// answer, so nobody reconciles against 20 rows of a 4,000-row report.
func digestNote(digest *services.ReportDigest, rowCount int64) string {
	if digest.IsEmpty() || len(digest.Rows) == 0 {
		return ""
	}
	if !digest.Truncated {
		return ""
	}
	return fmt.Sprintf(
		"Showing the first %d of %d rows — the full result is in the report file.",
		len(digest.Rows), rowCount,
	)
}
