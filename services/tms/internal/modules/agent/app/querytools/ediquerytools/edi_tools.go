package ediquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
)

const (
	maxRawX12Bytes        = 16 * 1024
	maxMessageDiagnostics = 50
	ediTransferEntity     = "edi_transfer"
)

func Providers() []any {
	return ediDecisionToolProviders()
}

// rawX12Excerpt is the raw X12 a preview shows, or why it shows none.
type rawX12Excerpt struct {
	Raw          string   `json:"rawX12,omitempty"`
	RawTruncated bool     `json:"rawTruncated,omitempty"`
	Note         string   `json:"note,omitempty"`
	Withheld     []string `json:"withheldByAccess,omitempty"`
}

func rawEDIWithheldNote(gate *fieldgate.Gate, field string) string {
	if gate.HeldByAgent(field) {
		return "Raw EDI is Restricted and this agent reads at Internal, so the X12 is held " +
			"back. Tell the person that someone whose role reaches Restricted can set this " +
			"agent's Data access to Restricted in AI control."
	}

	return "Raw EDI is Restricted and the person's role does not reach it, so the X12 is held " +
		"back. Tell them so."
}

// ediDiagnosticRow is one validation finding on an X12 document a preview
// renders.
type ediDiagnosticRow struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Segment      string `json:"segment,omitempty"`
	Element      int    `json:"element,omitempty"`
	Path         string `json:"path,omitempty"`
	Message      string `json:"message"`
	SuggestedFix string `json:"suggestedFix,omitempty"`
}
