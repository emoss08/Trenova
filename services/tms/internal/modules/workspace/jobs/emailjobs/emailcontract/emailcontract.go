package emailcontract

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const SendEmailWorkflowName = "SendEmailWorkflow"

type SendEmailPayload struct {
	temporaltype.BasePayload

	MessageID    pulid.ID          `json:"messageId"`
	HTML         string            `json:"html"`
	Text         string            `json:"text"`
	Headers      map[string]string `json:"headers"`
	OpenTracking bool              `json:"openTracking"`
}
