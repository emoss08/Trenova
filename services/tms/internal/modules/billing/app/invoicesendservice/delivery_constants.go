package invoicesendservice

import (
	"regexp"
	"time"
)

const (
	shareTokenTTL = 14 * 24 * time.Hour

	invoiceDocumentTypeCode = "INVOICE"
)

// invoiceTemplateVariablePattern matches a variable written either as
// {{name}} or as {name}, the second being what an organization's own template
// editor has always accepted.
var invoiceTemplateVariablePattern = regexp.MustCompile(
	`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}|\{\s*([A-Za-z0-9_.]+)\s*\}`,
)
