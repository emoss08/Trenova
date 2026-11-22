package emaildesign

import (
	"strings"

	"github.com/emoss08/trenova/pkg/templatediag"
)

func Check(content string) templatediag.Diagnostics {
	var diags templatediag.Diagnostics
	structure := func(message string) {
		diags = append(
			diags,
			templatediag.Error(templatediag.CodeEmailStructure, contentField, message),
		)
	}

	switch preheaderMarker.count(content) {
	case 0:
		structure("the email has no preheader. Add {{ preheader (t \"...\") }} with one sentence " +
			"that says what the email is")
	case 1:
	default:
		structure("the email has more than one preheader; keep one")
	}

	switch footerMarker.count(content) {
	case 0:
		structure("the email has no footer. Add {{ footer (t \"...\") }} saying why the reader " +
			"received it")
	case 1:
	default:
		structure("the email has more than one footer; keep one")
	}

	if bandMarker.count(content) > 1 {
		structure("the email has more than one action band. Keep the main action in the band " +
			"and write any other as a link in the text")
	}

	if headings := strings.Count(strings.ToLower(content), "<h1"); headings != 1 {
		structure("the email needs exactly one heading, written with {{ heading \"...\" }}")
	}

	return diags
}

func warnings(diags templatediag.Diagnostics) templatediag.Diagnostics {
	out := make(templatediag.Diagnostics, 0, len(diags))
	for _, d := range diags {
		d.Severity = templatediag.SeverityWarning
		out = append(out, d)
	}
	return out
}
