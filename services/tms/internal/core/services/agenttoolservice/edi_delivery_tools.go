package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramEDIMessageIDs     = "messageIds"
	paramEDIMessageID      = "messageId"
	paramEDIInboundFileIDs = "inboundFileIds"
	maxBulkOutcomeFailures = 3
	ediMessageRecordEntity = "edi_message"
)

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
		req repositories.GetEDIInboundFileByIDRequest,
	) (*edi.EDIInboundFile, error)
	BulkReprocessInboundFiles(
		ctx context.Context,
		req *ediinboundservice.BulkReprocessInboundFilesRequest,
	) (*ediservice.BulkEDIActionResult, error)
}

func ediRecordSet(
	tool serviceports.AgentTool,
	params *serviceports.ToolExecuteParams,
	key string,
) ([]pulid.ID, error) {
	if err := guardPreview(tool, params); err != nil {
		return nil, err
	}

	ids, err := requirePulidSlice(params.Params, key, ediservice.MaxBulkEDIActionItems)
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

func ediOutboundPolicy(name, rationale string) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

type retryEDIMessageDeliveryTool struct {
	deliveries ediDeliveryRetrier
}

var (
	_ serviceports.ToolPreviewer      = (*retryEDIMessageDeliveryTool)(nil)
	_ serviceports.ToolValidator      = (*retryEDIMessageDeliveryTool)(nil)
	_ serviceports.ToolResultReporter = (*retryEDIMessageDeliveryTool)(nil)
)

func newRetryEDIMessageDeliveryTool(deliveries ediDeliveryRetrier) serviceports.AgentTool {
	return &retryEDIMessageDeliveryTool{deliveries: deliveries}
}

func (t *retryEDIMessageDeliveryTool) Name() string { return "retry_edi_message_delivery" }

func (t *retryEDIMessageDeliveryTool) Description() string {
	return "Propose sending outbound EDI documents that failed to reach their trading " +
		"partner again, over the partner's communication profile. Use it once the cause " +
		"of the failure is fixed or was temporary, such as a partner's server that was " +
		"down; a message whose cause still stands fails again. Cover every message that " +
		"can go in one call. A person always decides and may untick messages."
}

func (t *retryEDIMessageDeliveryTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIMessageIDs: ediRecordSubset("The outbound messages to send again, by id " +
			"from list_edi_messages filtered to Failed or DeadLettered. Never guess one."),
	}, paramEDIMessageIDs)
}

func (t *retryEDIMessageDeliveryTool) Policy() serviceports.ToolPolicy {
	policy := ediOutboundPolicy(t.Name(),
		"Sends documents to trading partners outside the organization, who act on "+
			"what they receive; a document sent cannot be recalled.")
	policy.Artifact = ediMessageRecordEntity

	return policy
}

func (t *retryEDIMessageDeliveryTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := ediRecordSet(t, &params, paramEDIMessageIDs)

	return err
}

func (t *retryEDIMessageDeliveryTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *retryEDIMessageDeliveryTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
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
			TenantInfo: tenantFrom(params),
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

type replayEDIMessageTool struct {
	deliveries ediMessageReplayer
}

var (
	_ serviceports.ToolPreviewer = (*replayEDIMessageTool)(nil)
	_ serviceports.ToolValidator = (*replayEDIMessageTool)(nil)
	_ serviceports.TargetedTool  = (*replayEDIMessageTool)(nil)
)

func newReplayEDIMessageTool(deliveries ediMessageReplayer) serviceports.AgentTool {
	return &replayEDIMessageTool{deliveries: deliveries}
}

func (t *replayEDIMessageTool) Name() string { return "replay_edi_message" }

func (t *replayEDIMessageTool) Description() string {
	return "Propose sending an outbound EDI document its trading partner already " +
		"received once more, exactly as it was generated. Use it only when the partner " +
		"says they lost or never processed it; the partner gets it twice otherwise. A " +
		"failed delivery is retry_edi_message_delivery's. A person always decides."
}

func (t *replayEDIMessageTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIMessageID: jsonschemautils.Text("The delivered outbound message, from " +
			"list_edi_messages filtered to Sent. Never guess one."),
	}, paramEDIMessageID)
}

func (t *replayEDIMessageTool) Policy() serviceports.ToolPolicy {
	return ediOutboundPolicy(t.Name(),
		"Sends a trading partner a document it already has, which it may act on a "+
			"second time; a document sent cannot be recalled.")
}

func (t *replayEDIMessageTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramEDIMessageID, permission.ResourceEDI)
}

func (t *replayEDIMessageTool) request(
	params *serviceports.ToolExecuteParams,
) (*ediservice.RetryMessageDeliveryRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	messageID, err := requirePulid(params.Params, paramEDIMessageID)
	if err != nil {
		return nil, err
	}

	return &ediservice.RetryMessageDeliveryRequest{
		MessageID:  messageID,
		TenantInfo: tenantFrom(*params),
	}, nil
}

func (t *replayEDIMessageTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.deliveries.PlanReplayMessageDelivery(ctx, req)

	return err
}

func (t *replayEDIMessageTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
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

type reprocessEDIInboundFilesTool struct {
	files inboundFileReprocessor
}

var (
	_ serviceports.ToolPreviewer      = (*reprocessEDIInboundFilesTool)(nil)
	_ serviceports.ToolValidator      = (*reprocessEDIInboundFilesTool)(nil)
	_ serviceports.ToolResultReporter = (*reprocessEDIInboundFilesTool)(nil)
)

func newReprocessEDIInboundFilesTool(files inboundFileReprocessor) serviceports.AgentTool {
	return &reprocessEDIInboundFilesTool{files: files}
}

func (t *reprocessEDIInboundFilesTool) Name() string { return "reprocess_edi_inbound_files" }

func (t *reprocessEDIInboundFilesTool) Description() string {
	return "Propose running quarantined or partly processed inbound EDI files through " +
		"processing again. Use it once what held them back is fixed, such as a partner " +
		"now set up for inbound. Each file's tenders, status updates and invoices are created " +
		"again and its partner is sent acknowledgments. Never reprocess a file whose cause " +
		"still stands. A person always decides and may untick files."
}

func (t *reprocessEDIInboundFilesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIInboundFileIDs: ediRecordSubset("The inbound files to process again, by id " +
			"from list_edi_inbound_files or get_edi_inbound_file. Never guess one."),
	}, paramEDIInboundFileIDs)
}

func (t *reprocessEDIInboundFilesTool) Policy() serviceports.ToolPolicy {
	policy := ediOutboundPolicy(t.Name(),
		"Turns what trading partners sent into tenders, status updates and invoices "+
			"again, and sends each partner acknowledgments they act on.")
	policy.Artifact = agent.TaintEntityEDIInboundFile

	return policy
}

func (t *reprocessEDIInboundFilesTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := ediRecordSet(t, &params, paramEDIInboundFileIDs)

	return err
}

func (t *reprocessEDIInboundFilesTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *reprocessEDIInboundFilesTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
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
			TenantInfo: tenantFrom(params),
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
