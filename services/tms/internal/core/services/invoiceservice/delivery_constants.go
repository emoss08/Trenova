package invoiceservice

import (
	"regexp"
	"time"
)

const (
	// postmarkMessageLimitBytes and resendMessageLimitBytes are the provider's
	// own ceilings on one message, so a plan that would exceed them is split
	// across sends rather than rejected.
	postmarkMessageLimitBytes = int64(10 * 1024 * 1024)
	resendMessageLimitBytes   = int64(40 * 1024 * 1024)

	defaultBodyOverheadBytes = int64(16 * 1024)

	shareTokenTTL = 14 * 24 * time.Hour

	invoiceDocumentTypeCode = "INVOICE"
)

// invoiceTemplateVariablePattern matches a variable written either as
// {{name}} or as {name}, the second being what an organization's own template
// editor has always accepted.
var invoiceTemplateVariablePattern = regexp.MustCompile(
	`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}|\{\s*([A-Za-z0-9_.]+)\s*\}`,
)
