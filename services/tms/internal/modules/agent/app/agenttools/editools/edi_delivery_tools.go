package editools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const paramEDIMessageIDs = "messageIds"

const paramEDIMessageID = "messageId"

const paramEDIInboundFileIDs = "inboundFileIds"

const maxBulkOutcomeFailures = 3

const ediMessageRecordEntity = "edi_message"

type ediDeliveryRetrier interface {
	PlanRetryMessageDelivery(
		ctx context.Context,
		req *ediservice.RetryMessageDeliveryRequest,
	) (*ediservice.DeliveryPlan, error)
	BulkRetryMessageDelivery(
		ctx context.Context,
		req *ediservice.BulkRetryMessageDeliveryRequest,
	) (*ediservice.BulkEDIActionResult, error)
}

type ediMessageReplayer interface {
	PlanReplayMessageDelivery(
		ctx context.Context,
		req *ediservice.RetryMessageDeliveryRequest,
	) (*ediservice.DeliveryPlan, error)
	ReplayMessageDelivery(
		ctx context.Context,
		req *ediservice.RetryMessageDeliveryRequest,
	) (*edi.EDIMessage, error)
}

type inboundFileReprocessor interface {
	GetInboundFile(
		ctx context.Context,
		req ediports.GetEDIInboundFileByIDRequest,
	) (*edi.EDIInboundFile, error)
	BulkReprocessInboundFiles(
		ctx context.Context,
		req *ediinboundservice.BulkReprocessInboundFilesRequest,
	) (*ediservice.BulkEDIActionResult, error)
}

func ediRecordSet(
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
	key string,
) ([]pulid.ID, error) {
	if err := common.GuardPreview(tool, params); err != nil {
		return nil, err
	}

	ids, err := common.RequirePulidSlice(params.Params, key, ediservice.MaxBulkEDIActionItems)
	if err != nil {
		return nil, err
	}

	return sliceutils.Dedupe(ids), nil
}

func ediRecordSubset(description string) map[string]any {
	return toolschema.RecordSubset(permission.ResourceEDI.String(), map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    ediservice.MaxBulkEDIActionItems,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
		},
	})
}

func ediOutboundPolicy(name, rationale string) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceEDI,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     rationale + ediDecisionRationaleEnd,
	}
}

func bulkOutcome(result *ediservice.BulkEDIActionResult, verb string) string {
	total := len(result.Succeeded) + len(result.Failed)
	outcome := fmt.Sprintf("%d of %d %s", len(result.Succeeded), total, verb)
	failures := make([]string, 0, min(len(result.Failed), maxBulkOutcomeFailures))
	for _, failure := range result.Failed {
		if len(failures) == maxBulkOutcomeFailures {
			break
		}
		failures = append(failures, failure.ID.String()+": "+failure.Error)
	}
	if len(failures) > 0 {
		outcome += "; refused " + strings.Join(failures, "; ")
	}

	return outcome
}

type RetryEDIMessageDeliveryTool struct {
	deliveries ediDeliveryRetrier
}

func NewRetryEDIMessageDeliveryTool(deliveries ediDeliveryRetrier) agenttoolports.AgentTool {
	return &RetryEDIMessageDeliveryTool{deliveries: deliveries}
}

func (t *RetryEDIMessageDeliveryTool) Name() string { return "edi_retry_message" }

func (t *RetryEDIMessageDeliveryTool) Description() string {
	return "Propose sending outbound EDI documents that failed to reach their trading " +
		"partner again, over the partner's communication profile. Use it once the cause " +
		"of the failure is fixed or was temporary, such as a partner's server that was " +
		"down; a message whose cause still stands fails again. Cover every message that " +
		"can go in one call. A person always decides and may untick messages."
}

func (t *RetryEDIMessageDeliveryTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIMessageIDs: agenttoolschema.OfKinds(ediRecordSubset("The outbound messages "+
			"to send again, by id from edi_list_messages filtered to Failed or DeadLettered. "+
			"Never guess one."), permission.KindEDIMessage),
	}, paramEDIMessageIDs)
}

func (t *RetryEDIMessageDeliveryTool) Policy() agenttoolports.ToolPolicy {
	policy := ediOutboundPolicy(t.Name(),
		"Sends documents to trading partners outside the organization, who act on "+
			"what they receive; a document sent cannot be recalled.")
	policy.Artifact = ediMessageRecordEntity

	return policy
}

func (t *RetryEDIMessageDeliveryTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := ediRecordSet(t, &params, paramEDIMessageIDs)

	return err
}

func (t *RetryEDIMessageDeliveryTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *RetryEDIMessageDeliveryTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrEDIDecisionNeedsAPerson
	}

	ids, err := ediRecordSet(t, &params, paramEDIMessageIDs)
	if err != nil {
		return nil, err
	}

	result, err := t.deliveries.BulkRetryMessageDelivery(
		ctx,
		&ediservice.BulkRetryMessageDeliveryRequest{
			TenantInfo: common.TenantFrom(params),
			MessageIDs: ids,
		},
	)
	if err != nil {
		return nil, err
	}

	return &agent.ToolExecutionResult{
		Action: "queued",
		Kind:   "EDI messages",
		Name:   bulkOutcome(result, "queued to send again"),
	}, nil
}

type ReplayEDIMessageTool struct {
	deliveries ediMessageReplayer
}

func NewReplayEDIMessageTool(deliveries ediMessageReplayer) agenttoolports.AgentTool {
	return &ReplayEDIMessageTool{deliveries: deliveries}
}

func (t *ReplayEDIMessageTool) Name() string { return "edi_replay_message" }

func (t *ReplayEDIMessageTool) Description() string {
	return "Propose sending an outbound EDI document its trading partner already " +
		"received once more, exactly as it was generated. Use it only when the partner " +
		"says they lost or never processed it; the partner gets it twice otherwise. A " +
		"failed delivery is edi_retry_message's. A person always decides."
}

func (t *ReplayEDIMessageTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIMessageID: agenttoolschema.KindID("The delivered outbound message, from "+
			"edi_list_messages filtered to Sent. Never guess one.", permission.KindEDIMessage),
	}, paramEDIMessageID)
}

func (t *ReplayEDIMessageTool) Policy() agenttoolports.ToolPolicy {
	return ediOutboundPolicy(t.Name(),
		"Sends a trading partner a document it already has, which it may act on a "+
			"second time; a document sent cannot be recalled.")
}

func (t *ReplayEDIMessageTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramEDIMessageID, permission.ResourceEDI)
}

func (t *ReplayEDIMessageTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*ediservice.RetryMessageDeliveryRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	messageID, err := common.RequirePulid(params.Params, paramEDIMessageID)
	if err != nil {
		return nil, err
	}

	return &ediservice.RetryMessageDeliveryRequest{
		MessageID:  messageID,
		TenantInfo: common.TenantFrom(*params),
	}, nil
}

func (t *ReplayEDIMessageTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.deliveries.PlanReplayMessageDelivery(ctx, req)

	return err
}

func (t *ReplayEDIMessageTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.deliveries.ReplayMessageDelivery(ctx, req)

	return err
}

type ReprocessEDIInboundFilesTool struct {
	files inboundFileReprocessor
}

func NewReprocessEDIInboundFilesTool(files inboundFileReprocessor) agenttoolports.AgentTool {
	return &ReprocessEDIInboundFilesTool{files: files}
}

func (t *ReprocessEDIInboundFilesTool) Name() string { return "edi_reprocess_inbound_file" }

func (t *ReprocessEDIInboundFilesTool) Recipe() []string {
	return []string{
		"edi_list_inbound_files",
		"edi_get_inbound_file",
		"edi_get_partner",
		"edi_reprocess_inbound_file",
	}
}

func (t *ReprocessEDIInboundFilesTool) Description() string {
	return "Propose running quarantined or partly processed inbound EDI files through " +
		"processing again. Use it once what held them back is fixed, such as a partner " +
		"now set up for inbound. Each file's tenders, status updates and invoices are created " +
		"again and its partner is sent acknowledgments. Never reprocess a file whose cause " +
		"still stands. A person always decides and may untick files."
}

func (t *ReprocessEDIInboundFilesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIInboundFileIDs: agenttoolschema.OfKinds(ediRecordSubset("The inbound files "+
			"to process again, by id from edi_list_inbound_files or edi_get_inbound_file. "+
			"Never guess one."), permission.KindEDIInboundFile),
	}, paramEDIInboundFileIDs)
}

func (t *ReprocessEDIInboundFilesTool) Policy() agenttoolports.ToolPolicy {
	policy := ediOutboundPolicy(t.Name(),
		"Turns what trading partners sent into tenders, status updates and invoices "+
			"again, and sends each partner acknowledgments they act on.")
	policy.Artifact = agent.TaintEntityEDIInboundFile

	return policy
}

func (t *ReprocessEDIInboundFilesTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := ediRecordSet(t, &params, paramEDIInboundFileIDs)

	return err
}

func (t *ReprocessEDIInboundFilesTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *ReprocessEDIInboundFilesTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrEDIDecisionNeedsAPerson
	}

	ids, err := ediRecordSet(t, &params, paramEDIInboundFileIDs)
	if err != nil {
		return nil, err
	}

	result, err := t.files.BulkReprocessInboundFiles(
		ctx,
		&ediinboundservice.BulkReprocessInboundFilesRequest{
			TenantInfo: common.TenantFrom(params),
			FileIDs:    ids,
		},
	)
	if err != nil {
		return nil, err
	}

	return &agent.ToolExecutionResult{
		Action: "reprocessed",
		Kind:   "EDI inbound files",
		Name:   bulkOutcome(result, "processed again"),
	}, nil
}
