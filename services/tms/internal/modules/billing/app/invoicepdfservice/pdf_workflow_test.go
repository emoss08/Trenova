package invoicepdfservice

import (
	"regexp"
)

var pdfPageObjectPattern = regexp.MustCompile(`/Type\s*/Page\b`)
