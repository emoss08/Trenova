package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDeliveries struct {
	messages map[pulid.ID]*edi.EDIMessage
	guard    *writeGuard
	retried  *ediservice.BulkRetryMessageDeliveryRequest
	replayed *ediservice.RetryMessageDeliveryRequest
}

func (f *fakeDeliveries) message(id pulid.ID) (*edi.EDIMessage, error) {
	message, ok := f.messages[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("EDIMessage not found")
	}

	return message, nil
}

func (f *fakeDeliveries) PlanRetryMessageDelivery(
	_ context.Context,
	req *ediservice.RetryMessageDeliveryRequest,
) (*ediservice.DeliveryPlan, error) {
	message, err := f.message(req.MessageID)
	if err != nil {
		return nil, err
	}
	if !message.DeliveryStatus.IsRetryable() {
		return nil, errortypes.NewValidationError(
			"deliveryStatus", errortypes.ErrInvalidOperation,
			"Only queued, failed, or dead-lettered EDI messages can be retried",
		)
	}

	return &ediservice.DeliveryPlan{
		Message: message,
		Profile: &edi.EDICommunicationProfile{Name: "Acme SFTP", Method: edi.ConnectionMethodSFTP},
	}, nil
}

func (f *fakeDeliveries) PlanReplayMessageDelivery(
	_ context.Context,
	req *ediservice.RetryMessageDeliveryRequest,
) (*ediservice.DeliveryPlan, error) {
	message, err := f.message(req.MessageID)
	if err != nil {
		return nil, err
	}
	if message.DeliveryStatus != edi.MessageDeliveryStatusSent {
		return nil, errortypes.NewValidationError(
			"deliveryStatus", errortypes.ErrInvalidOperation,
			"Only delivered EDI messages can be replayed; use retry for failed messages",
		)
	}

	return &ediservice.DeliveryPlan{Message: message}, nil
}

func (f *fakeDeliveries) BulkRetryMessageDelivery(
	_ context.Context,
	req *ediservice.BulkRetryMessageDeliveryRequest,
) (*ediservice.BulkEDIActionResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.retried = req

	return &ediservice.BulkEDIActionResult{Succeeded: req.MessageIDs}, nil
}

func (f *fakeDeliveries) ReplayMessageDelivery(
	_ context.Context,
	req *ediservice.RetryMessageDeliveryRequest,
) (*edi.EDIMessage, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.replayed = req

	return f.messages[req.MessageID], nil
}

func outboundMessage(status edi.MessageDeliveryStatus) *edi.EDIMessage {
	return &edi.EDIMessage{
		ID:                       pulid.MustNew("edimsg_"),
		Direction:                edi.DocumentDirectionOutbound,
		TransactionSet:           edi.TransactionSet214,
		InterchangeControlNumber: "000000042",
		DeliveryStatus:           status,
		DeliveryLastError:        "connection refused",
		RawX12:                   "ISA*00*SECRET-RATE~",
		Partner:                  &edi.EDIPartner{Name: "Acme Shipper"},
		Version:                  3,
	}
}

func TestEDIDeliveryTools_ArePersonOnlyProposals(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentTool{
		newRetryEDIMessageDeliveryTool(nil),
		newReplayEDIMessageTool(nil),
		newReprocessEDIInboundFilesTool(nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, permission.ResourceEDI, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(
			t,
			[]agent.EgressClass{agent.EgressExternalRecipient},
			policy.Egress,
			tool.Name(),
		)
	}

	properties := newRetryEDIMessageDeliveryTool(nil).ParamSchema()["properties"].(map[string]any)
	assert.Equal(t, permission.ResourceEDI.String(),
		toolschema.SubsetResource(properties[paramEDIMessageIDs].(map[string]any)),
		"a person approving may untick messages")
}

func TestRetryEDIMessageDelivery_PreviewsEachMessageAndRefusalWithoutItsContent(t *testing.T) {
	t.Parallel()

	failed := outboundMessage(edi.MessageDeliveryStatusFailed)
	sent := outboundMessage(edi.MessageDeliveryStatusSent)
	deliveries := &fakeDeliveries{
		messages: map[pulid.ID]*edi.EDIMessage{failed.ID: failed, sent.ID: sent},
		guard:    &writeGuard{},
	}
	tool := newRetryEDIMessageDeliveryTool(deliveries).(*retryEDIMessageDeliveryTool)
	args := map[string]any{
		paramEDIMessageIDs: []any{failed.ID.String(), sent.ID.String(), failed.ID.String()},
	}

	preview := previewWithoutWrites(t, deliveries.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	require.Len(t, preview.Changes, 2, "a message named twice is sent once")
	refused := previewChange(t, preview, 0)
	assert.Equal(t, sent.ID, refused.EntityID, "refusals come first")
	assert.Contains(t, fieldByPath(t, refused, "outcome").After, "Only queued, failed")
	queued := previewChange(t, preview, 1)
	assert.Equal(t, "Queued", fieldByPath(t, queued, "deliveryStatus").After)
	require.NotNil(t, queued.Message)
	assert.Equal(t, agent.MessageChannelEDI, queued.Message.Channel)
	assert.NotContains(t, queued.Message.Body, "SECRET-RATE", "the document's content stays out")
	assert.Contains(t, preview.Summary, "1 of 2")

	require.NoError(t, tool.Validate(t.Context(), agentParamsFor(args)))
	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)

	result, err := tool.ExecuteWithResult(t.Context(), approvedParams(args))
	require.NoError(t, err)
	require.NotNil(t, deliveries.retried)
	assert.Equal(t, []pulid.ID{failed.ID, sent.ID}, deliveries.retried.MessageIDs)
	assert.Contains(t, result.Name, "2 of 2")
}

func TestReplayEDIMessage_SendsADeliveredDocumentAgainOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	sent := outboundMessage(edi.MessageDeliveryStatusSent)
	failed := outboundMessage(edi.MessageDeliveryStatusFailed)
	deliveries := &fakeDeliveries{
		messages: map[pulid.ID]*edi.EDIMessage{sent.ID: sent, failed.ID: failed},
		guard:    &writeGuard{},
	}
	tool := newReplayEDIMessageTool(deliveries).(*replayEDIMessageTool)

	target, ok := tool.Target(map[string]any{paramEDIMessageID: sent.ID.String()})
	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceEDI, ID: sent.ID}, target)

	args := map[string]any{paramEDIMessageID: sent.ID.String()}
	preview := previewWithoutWrites(t, deliveries.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(3), *change.Version)
	assert.Equal(t, "Queued", fieldByPath(t, change, "deliveryStatus").After)

	refused := map[string]any{paramEDIMessageID: failed.ID.String()}
	require.Error(t, tool.Validate(t.Context(), agentParamsFor(refused)))

	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(args)))
	require.NotNil(t, deliveries.replayed)
	assert.Equal(t, sent.ID, deliveries.replayed.MessageID)
}

type fakeReprocessor struct {
	files       map[pulid.ID]*edi.EDIInboundFile
	guard       *writeGuard
	reprocessed *ediinboundservice.BulkReprocessInboundFilesRequest
}

func (f *fakeReprocessor) GetInboundFile(
	_ context.Context,
	req repositories.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	file, ok := f.files[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("EDIInboundFile not found")
	}

	return file, nil
}

func (f *fakeReprocessor) BulkReprocessInboundFiles(
	_ context.Context,
	req *ediinboundservice.BulkReprocessInboundFilesRequest,
) (*ediservice.BulkEDIActionResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.reprocessed = req

	return &ediservice.BulkEDIActionResult{
		Succeeded: req.FileIDs[:1],
		Failed: []ediservice.BulkEDIActionFailure{{
			ID: req.FileIDs[len(req.FileIDs)-1], Error: "Only quarantined files",
		}},
	}, nil
}

func TestReprocessEDIInboundFiles_PreviewsWhichFilesWouldRunAgain(t *testing.T) {
	t.Parallel()

	quarantined := &edi.EDIInboundFile{
		ID:            pulid.MustNew("ediinf_"),
		FileName:      "204_0921.x12",
		Status:        edi.InboundFileStatusQuarantined,
		FailureReason: "partner could not be resolved",
		Version:       5,
	}
	processed := &edi.EDIInboundFile{
		ID:       pulid.MustNew("ediinf_"),
		FileName: "214_0921.x12",
		Status:   edi.InboundFileStatusProcessed,
	}
	files := &fakeReprocessor{
		files: map[pulid.ID]*edi.EDIInboundFile{
			quarantined.ID: quarantined,
			processed.ID:   processed,
		},
		guard: &writeGuard{},
	}
	tool := newReprocessEDIInboundFilesTool(files).(*reprocessEDIInboundFilesTool)
	args := map[string]any{
		paramEDIInboundFileIDs: []any{quarantined.ID.String(), processed.ID.String()},
	}

	preview := previewWithoutWrites(t, files.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	require.Len(t, preview.Changes, 2)
	refused := previewChange(t, preview, 0)
	assert.Equal(t, processed.ID, refused.EntityID)
	assert.Contains(t, fieldByPath(t, refused, "outcome").After, "Only quarantined")
	again := previewChange(t, preview, 1)
	assert.Equal(t, "204_0921.x12", again.Label)
	assert.True(t, preview.Partial, "what each transaction becomes is decided when it runs")

	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	result, err := tool.ExecuteWithResult(t.Context(), approvedParams(args))
	require.NoError(t, err)
	require.NotNil(t, files.reprocessed)
	assert.Contains(t, result.Name, "1 of 2")
	assert.Contains(t, result.Name, "Only quarantined files")
}

func TestEDIBulkTools_RefuseMoreThanTheServiceTakes(t *testing.T) {
	t.Parallel()

	ids := make([]any, 0, ediservice.MaxBulkEDIActionItems+1)
	for range ediservice.MaxBulkEDIActionItems + 1 {
		ids = append(ids, pulid.MustNew("edimsg_").String())
	}

	tool := newRetryEDIMessageDeliveryTool(&fakeDeliveries{})
	err := tool.(serviceports.ToolValidator).Validate(
		t.Context(),
		agentParamsFor(map[string]any{paramEDIMessageIDs: ids}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "split it into smaller calls")
}
