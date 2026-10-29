package carrierpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
)

var RateConfirmationLabels = map[string]string{
	FieldCarrierID:                changepreview.LabelCarrier,
	"sentToEmails":                "Sent to",
	FieldConfirmedByName:          "Confirmed by",
	"confirmedVia":                "Confirmed through",
	changepreview.FieldVoidReason: "Void reason",
	"carrierAssignmentId":         "Carrier assignment",
}
