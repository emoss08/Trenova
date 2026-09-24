// Package importassistantjobs runs the shipment import assistant's turns as
// workflows: the model loop in workflow code, each model call and each tool
// call an activity, and the reply streamed to the reader through the Workflow
// Stream the turn hosts.
package importassistantjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentimportassistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/pagination"
)

const ImportAssistantTurnWorkflowName = "ImportAssistantTurnWorkflow"

const (
	// turnTimeout bounds a whole turn: five rounds of model and tool calls.
	// A turn still going past it is looping, and the person is told so.
	turnTimeout = 15 * time.Minute

	prepareTimeout = time.Minute
	modelTimeout   = 5 * time.Minute
	toolTimeout    = 2 * time.Minute
	finishTimeout  = time.Minute

	modelAttempts = 3
	readAttempts  = 3
	// writeAttempts is one: a tool that creates a shipment or a location is
	// not asked twice, because creating it twice is worse than failing once.
	writeAttempts = 1
	saveAttempts  = 5

	// errTypeTurnFailed is a turn that ended without a reply. Its message is
	// written for the person.
	errTypeTurnFailed = "ImportTurnFailed"
)

// TurnPayload is one message to the import assistant. The request's own JSON
// leaves the tenant out, so it travels beside it.
type TurnPayload struct {
	TenantInfo pagination.TenantInfo                   `json:"tenantInfo"`
	Request    *serviceports.ShipmentImportChatRequest `json:"request"`
	// Stream says somebody is reading the reply as it is written.
	Stream bool `json:"stream"`
}

// request is the message as the service reads it, tenant included.
func (p *TurnPayload) request() *serviceports.ShipmentImportChatRequest {
	req := *p.Request
	req.TenantInfo = p.TenantInfo

	return &req
}

func (p *TurnPayload) attribution() serviceports.AIUsageAttribution {
	attribution := serviceports.AIUsageAttribution{
		UserID:  p.TenantInfo.UserID,
		Feature: aiusage.FeatureShipmentImportChat,
	}
	if p.Request != nil {
		attribution.Subject = aiusage.Subject{
			Type: aiusage.SubjectTypeDocument,
			ID:   p.Request.DocumentID,
		}
	}

	return attribution
}

// ModelInput is one model call of a turn.
type ModelInput struct {
	Request *serviceports.ChatCompletionRequest `json:"request"`
	Stream  bool                                `json:"stream"`
}

// ToolInput is one tool call of a turn.
type ToolInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Call       serviceports.ToolCall `json:"call"`
}

// FinishInput is what a turn came to, for saving.
type FinishInput struct {
	Payload *TurnPayload                               `json:"payload"`
	Record  *shipmentimportassistantservice.TurnRecord `json:"record"`
	Failure *modelcall.Failure                         `json:"failure,omitempty"`
}
