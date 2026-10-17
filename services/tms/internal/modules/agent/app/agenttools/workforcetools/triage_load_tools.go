package workforcetools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramLoadFields   = "fields"
	paramLoadKey      = "key"
	paramLoadValue    = "value"
	maxLoadFieldValue = 500
)

var triageLoadFieldKeys = agenttoolschema.Source(
	"inboundMessage.loadFieldKey",
	inboundmessage.AllLoadFieldKeys(),
)

type triageLoads interface {
	DraftLoad(
		ctx context.Context,
		req *triageworkservice.LoadRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanDraftLoad(
		ctx context.Context,
		req *triageworkservice.LoadRequest,
	) (*triageworkservice.LoadDraftPlan, error)
	DiscardLoad(
		ctx context.Context,
		req *triageworkservice.LoadRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanDiscardLoad(
		ctx context.Context,
		req *triageworkservice.LoadRequest,
	) (*triageworkservice.LoadDiscardPlan, error)
	CreateLoad(
		ctx context.Context,
		req *triageworkservice.CreateLoadRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanCreateLoad(
		ctx context.Context,
		req *triageworkservice.CreateLoadRequest,
	) (*triageworkservice.CreateLoadPlan, error)
}

var errNoDraftedLoad = errors.New("this message has no drafted load to throw away")

type loadFieldsView map[string]string

func loadDraftView(draft *inboundmessage.LoadDraft) *loadFieldsView {
	view := loadFieldsView{}
	if draft == nil {
		return &view
	}
	for _, field := range draft.Fields {
		if field.Value != "" {
			view[string(field.Key)] = field.Value
		}
	}

	return &view
}

func loadValuesView(fields []inboundmessageports.TriageLoadFieldValue) *loadFieldsView {
	view := make(loadFieldsView, len(fields))
	for _, field := range fields {
		if field.Value != "" {
			view[string(field.Key)] = field.Value
		}
	}

	return &view
}

func loadDraftRecord(message *inboundmessage.InboundMessage) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInboundMessage,
		ID:       message.ID,
		Label:    "Load drafted from " + describeInboundMessage(message),
		Version:  common.PreviewVersion(message.Version),
	}
}

func loadRequest(params *agentports.ToolExecuteParams) (*triageworkservice.LoadRequest, error) {
	messageID, err := common.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}

	return &triageworkservice.LoadRequest{
		TenantInfo: tenantOnly(params),
		UserID:     actorOf(params),
		MessageID:  messageID,
	}, nil
}

func NewDraftInboundLoadTool(p triageToolParams) agentports.AgentTool {
	return newDraftInboundLoadTool(p.Inbox, p.Work)
}

func newDraftInboundLoadTool(inbox inboundMessageReader, loads triageLoads) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "draft_inbound_load",
			Artifact: inboundMessageEntity,
			Description: "Read the load a tender in Triage describes into a draft on the " +
				"message: customer, stops, dates, equipment, weight, rate and references. It " +
				"is read from the attached tender's document when that has one, else from " +
				"the message body. Nothing is booked; check the draft get_inbound_message " +
				"shows, then create it with create_inbound_load.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(inbox),
			Reversible:  true,
			Rationale: "Writes a draft on the message only people inside the organization " +
				"see; nothing is booked.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The tender, from get_inbound_message, list_inbound_messages or this " +
						"run's subject.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"draft the load", "tender fields", "read out the load"},
			Recipe:      []string{toolGetInboundMessage, "draft_inbound_load"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*triageworkservice.LoadRequest, *triageworkservice.LoadDraftPlan]{
			Request: loadRequest,
			Plan: func(
				ctx context.Context,
				req *triageworkservice.LoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*triageworkservice.LoadDraftPlan, error) {
				return loads.PlanDraftLoad(ctx, req)
			},
			Refused: func(*triageworkservice.LoadRequest) string {
				return "Would draft a load from an inbound message."
			},
			Render: func(
				_ *triageworkservice.LoadRequest,
				plan *triageworkservice.LoadDraftPlan,
			) (*agent.ToolPreview, error) {
				if plan.NeedsNova {
					preview := toolpreview.Build(
						"Would have Nova read the load from the message body into a draft on " +
							describeInboundMessage(plan.Message) + "; the fields are read " +
							"when this runs. Nothing is booked.",
					)
					preview.Partial = true

					return preview, nil
				}

				change, err := toolpreview.Changed(
					loadDraftRecord(plan.Message),
					loadDraftView(plan.Message.LoadDraft),
					loadDraftView(plan.Draft),
				)
				if err != nil {
					return nil, err
				}

				return toolpreview.Build(
					"Would read the load from "+plan.Draft.Source+" into a draft on the "+
						"message. Nothing is booked.",
					change,
				), nil
			},
			Run: func(
				ctx context.Context,
				req *triageworkservice.LoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				message, err := loads.DraftLoad(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("drafted a load from", message), nil
			},
		},
	)
}

func NewCreateInboundLoadTool(p triageToolParams) agentports.AgentTool {
	return newCreateInboundLoadTool(p.Work)
}

func newCreateInboundLoadTool(loads triageLoads) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name: "create_inbound_load",
			Description: "Create the shipment a tender in Triage describes, from the fields " +
				"you checked. Start from the draft draft_inbound_load wrote, which " +
				"get_inbound_message shows, and send every field the load should carry: one " +
				"left out is not on it. It is billed to the customer the message is linked " +
				"to, so link that first with link_inbound_message. A message that already " +
				"made its load is refused.",
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpCreate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierActWithApproval,
			MaxTier:     agent.TierActWithApproval,
			Idempotent:  true,
			Artifact:    common.ShipmentRecordEntity,
			Rationale: "A new load commits a customer's freight and the money that follows, " +
				"so no desk books one unattended.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The tender, from get_inbound_message or this run's subject.",
				),
				paramLoadFields: map[string]any{
					toolschema.KeyType: toolschema.TypeArray,
					toolschema.KeyDescription: "The load's fields, each once, as checked " +
						"against the tender and the draft get_inbound_message shows.",
					toolschema.KeyMinItems: 1,
					toolschema.KeyMaxItems: len(triageLoadFieldKeys.Values),
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeObject,
						toolschema.KeyProperties: map[string]any{
							paramLoadKey: agenttoolschema.Enum("Which field.",
								triageLoadFieldKeys),
							paramLoadValue: common.StringProperty(
								"The value as the tender says it: a city with its state, a "+
									"date with its window, a weight with its unit.",
								maxLoadFieldValue,
							),
						},
						toolschema.KeyRequired:             []string{paramLoadKey, paramLoadValue},
						toolschema.KeyAdditionalProperties: false,
					},
				},
			},
			Required:    []string{paramMessageID, paramLoadFields},
			SearchTerms: []string{"enter the tender", "tender from triage"},
			Recipe:      []string{toolGetInboundMessage, "create_inbound_load"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[
			*triageworkservice.CreateLoadRequest,
			*triageworkservice.CreateLoadPlan,
		]{
			Request: createLoadRequest,
			Plan: func(
				ctx context.Context,
				req *triageworkservice.CreateLoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*triageworkservice.CreateLoadPlan, error) {
				return loads.PlanCreateLoad(ctx, req)
			},
			Refused: func(*triageworkservice.CreateLoadRequest) string {
				return "Would create a shipment from an inbound tender."
			},
			Render: renderCreateLoad,
			Run: func(
				ctx context.Context,
				req *triageworkservice.CreateLoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				plan, err := loads.PlanCreateLoad(ctx, req)
				if err != nil {
					return nil, err
				}
				if plan.AlreadyCreated {
					return nil, errLoadAlreadyCreated(plan.Message)
				}

				message, err := loads.CreateLoad(ctx, req)
				if err != nil {
					return nil, err
				}
				shipmentID := message.ActionState.CreatedShipmentID

				return &agent.ToolExecutionResult{
					Action: "created",
					Kind:   "shipment",
					Name:   "Shipment from " + describeInboundMessage(message),
					IDs: map[string]string{
						"shipmentId":   idText(shipmentID),
						paramMessageID: message.ID.String(),
					},
					Record: common.RecordOf(common.ShipmentRecordEntity, shipmentID),
				}, nil
			},
		},
	)
}

func errLoadAlreadyCreated(message *inboundmessage.InboundMessage) error {
	return fmt.Errorf(
		"this message already made its load, shipment %s; do not create it again",
		message.ActionState.CreatedShipmentID,
	)
}

func createLoadRequest(
	params *agentports.ToolExecuteParams,
) (*triageworkservice.CreateLoadRequest, error) {
	messageID, err := common.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err = common.DecodeParam(params.Params, paramLoadFields, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("parameter %q must name at least one field", paramLoadFields)
	}

	fields := make([]triageworkservice.LoadFieldInput, 0, len(raw))
	for idx, field := range raw {
		key := inboundmessage.LoadFieldKey(field.Key)
		if !key.IsValid() {
			return nil, common.ErrUnknownValue(
				fmt.Sprintf("%s[%d].%s", paramLoadFields, idx, paramLoadKey),
				field.Key,
				triageLoadFieldKeys.Names(),
			)
		}
		fields = append(fields, triageworkservice.LoadFieldInput{Key: key, Value: field.Value})
	}

	return &triageworkservice.CreateLoadRequest{
		TenantInfo:     tenantOnly(params),
		UserID:         actorOf(params),
		MessageID:      messageID,
		Fields:         fields,
		IdempotencyKey: params.IdempotencyKey,
	}, nil
}

func renderCreateLoad(
	_ *triageworkservice.CreateLoadRequest,
	plan *triageworkservice.CreateLoadPlan,
) (*agent.ToolPreview, error) {
	summary := "Would create a shipment from " + describeInboundMessage(plan.Message)
	if plan.AlreadyCreated {
		return refusedPreview(summary+".", errLoadAlreadyCreated(plan.Message)), nil
	}

	shipment, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipment,
			Label:    "Shipment from " + describeInboundMessage(plan.Message),
		},
		loadValuesView(plan.Load.Fields),
	)
	if err != nil {
		return nil, err
	}

	if plan.Load.CustomerID.IsNil() {
		summary += ", with no customer linked to the message"
	} else {
		summary += " for the customer the message is linked to"
	}

	return toolpreview.Build(summary+". The message records the shipment it made.", shipment), nil
}

func NewDiscardInboundLoadTool(p triageToolParams) agentports.AgentTool {
	return newDiscardInboundLoadTool(p.Inbox, p.Work)
}

func newDiscardInboundLoadTool(inbox inboundMessageReader, loads triageLoads) agentports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "discard_inbound_load",
			Artifact: inboundMessageEntity,
			Description: "Throw away the load drafted from a tender in Triage, when the " +
				"draft is wrong or the tender is not going to be booked. A load already " +
				"created from the message is refused; draft_inbound_load reads it again.",
			Resource:    permission.ResourceInboundMessage,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(inbox),
			Reversible:  true,
			Rationale: "Removes a draft from the message inside Trenova; it can be read " +
				"again and nothing is booked or sent.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The tender, from get_inbound_message or this run's subject.",
				),
			},
			Required:    []string{paramMessageID},
			SearchTerms: []string{"discard the load draft", "throw away drafted load"},
			Recipe:      []string{toolGetInboundMessage, "discard_inbound_load"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*triageworkservice.LoadRequest, *triageworkservice.LoadDiscardPlan]{
			Request: loadRequest,
			Plan: func(
				ctx context.Context,
				req *triageworkservice.LoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*triageworkservice.LoadDiscardPlan, error) {
				return loads.PlanDiscardLoad(ctx, req)
			},
			Refused: func(*triageworkservice.LoadRequest) string {
				return "Would throw away a drafted load."
			},
			Render: func(
				_ *triageworkservice.LoadRequest,
				plan *triageworkservice.LoadDiscardPlan,
			) (*agent.ToolPreview, error) {
				summary := "Would throw away the load drafted from " +
					describeInboundMessage(plan.Message) + "."
				if plan.Message.LoadDraft == nil {
					return refusedPreview(summary, errNoDraftedLoad), nil
				}
				change, err := toolpreview.Changed(
					loadDraftRecord(plan.Message),
					loadDraftView(plan.Message.LoadDraft),
					loadDraftView(plan.After.LoadDraft),
				)
				if err != nil {
					return nil, err
				}

				return toolpreview.Build(summary, change), nil
			},
			Run: func(
				ctx context.Context,
				req *triageworkservice.LoadRequest,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				plan, err := loads.PlanDiscardLoad(ctx, req)
				if err != nil {
					return nil, err
				}
				if plan.Message.LoadDraft == nil {
					return nil, errNoDraftedLoad
				}
				message, err := loads.DiscardLoad(ctx, req)
				if err != nil {
					return nil, err
				}

				return messageResult("discarded the drafted load of", message), nil
			},
		},
	)
}
