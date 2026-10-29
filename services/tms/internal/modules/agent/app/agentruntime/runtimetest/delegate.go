package runtimetest

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/shared/pulid"
)

func ReportBuilder() agentturn.RuntimeDelegate {
	return agentturn.RuntimeDelegate{
		ID:          pulid.MustNew("agdef_"),
		Name:        "Report Builder",
		Description: "Builds and saves reports. It knows every dataset.",
		Icon:        "receipt",
		Accent:      "teal",
		Tools:       []string{"report_create", "report_run"},
	}
}
