package inboundtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagepeekservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	MessageEntity    = "inbound_message"
	ToolGetMessage   = "inbox_get_inbound_message"
	FieldMatchReason = "matchReason"
)

type TriageToolParams struct {
	fx.In

	Inbox     *inboundqueue.Service
	Work      *triageworkservice.Service
	Replies   *inboundreplyservice.Service
	Conflicts *triageconflictservice.Service
	Peeks     *triagepeekservice.Service
}

func MessageIDProperty(description string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceInboundMessage, description)
}

func TargetMessage(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamMessageID, permission.ResourceInboundMessage)
}

func TenantOnly(params *agenttoolports.ToolExecuteParams) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: params.OrganizationID, BuID: params.BusinessUnitID}
}

func ActorOf(params *agenttoolports.ToolExecuteParams) pulid.ID {
	return params.Actor.UserID
}

func MessageAction(
	params *agenttoolports.ToolExecuteParams,
) (*inboundqueue.MessageActionRequest, error) {
	messageID, err := toolparams.RequirePulid(params.Params, ParamMessageID)
	if err != nil {
		return nil, err
	}

	return &inboundqueue.MessageActionRequest{
		TenantInfo: TenantOnly(params),
		ActorID:    ActorOf(params),
		MessageID:  messageID,
	}, nil
}

func IDText(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

func MessageResult(
	action string,
	message *inboundmessage.InboundMessage,
) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "inbound message",
		Name:   DescribeMessage(message),
		IDs:    map[string]string{ParamMessageID: message.ID.String()},
		Record: changepreview.RecordOf(MessageEntity, message.ID),
	}
}

func MessageChange(
	before, after *inboundmessage.InboundMessage,
	opts ...toolpreview.Option,
) (*agentpreview.RecordChange, error) {
	return toolpreview.Changed(MessageRecord(before), before, after, opts...)
}

func RefusedPreview(summary string, err error) *agentpreview.ToolPreview {
	return changepreview.WarnWouldFail(toolpreview.Build(summary), err)
}
