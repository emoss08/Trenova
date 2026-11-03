package carrierpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
)

var RateConfirmationLabels = map[string]string{
	toolparams.ParamCarrierID:     changepreview.LabelCarrier,
	"sentToEmails":                "Sent to",
	FieldConfirmedByName:          "Confirmed by",
	"confirmedVia":                "Confirmed through",
	changepreview.FieldVoidReason: "Void reason",
	"carrierAssignmentId":         "Carrier assignment",
}
