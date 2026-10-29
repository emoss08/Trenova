package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edibulk"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDeliveries struct {
	messages map[pulid.ID]*edimessage.EDIMessage
	guard    *agenttoolstest.WriteGuard
	retried  *edidelivery.BulkRetryMessageDeliveryRequest
	replayed *edidelivery.RetryMessageDeliveryRequest
}

func (f *fakeDeliveries) message(id pulid.ID) (*edimessage.EDIMessage, error) {
	message, ok := f.messages[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("EDIMessage not found")
	}

	return message, nil
}

func (f *fakeDeliveries) PlanRetryMessageDelivery(
	_ context.Context,
	req *edidelivery.RetryMessageDeliveryRequest,
) (*edidelivery.DeliveryPlan, error) {
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

	return &edidelivery.DeliveryPlan{
		Message: message,
		Profile: &edi.EDICommunicationProfile{Name: "Acme SFTP", Method: edi.ConnectionMethodSFTP},
	}, nil
}

func (f *fakeDeliveries) PlanReplayMessageDelivery(
	_ context.Context,
	req *edidelivery.RetryMessageDeliveryRequest,
) (*edidelivery.DeliveryPlan, error) {
	message, err := f.message(req.MessageID)
	if err != nil {
		return nil, err
	}
	if message.DeliveryStatus != edimessage.MessageDeliveryStatusSent {
		return nil, errortypes.NewValidationError(
			"deliveryStatus", errortypes.ErrInvalidOperation,
			"Only delivered EDI messages can be replayed; use retry for failed messages",
		)
	}

	return &edidelivery.DeliveryPlan{Message: message}, nil
}

func (f *fakeDeliveries) BulkRetryMessageDelivery(
	_ context.Context,
	req *edidelivery.BulkRetryMessageDeliveryRequest,
) (*edibulk.Result, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.retried = req

	return &edibulk.Result{Succeeded: req.MessageIDs}, nil
}

func (f *fakeDeliveries) ReplayMessageDelivery(
	_ context.Context,
	req *edidelivery.RetryMessageDeliveryRequest,
) (*edimessage.EDIMessage, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.replayed = req

	return f.messages[req.MessageID], nil
}

func outboundMessage(status edimessage.MessageDeliveryStatus) *edimessage.EDIMessage {
	return &edimessage.EDIMessage{
		ID:                       pulid.MustNew("edimsg_"),
		Direction:                edidocument.DocumentDirectionOutbound,
		TransactionSet:           edidocument.TransactionSet214,
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

	for _, tool := range []agenttoolports.AgentTool{
		NewRetryEDIMessageDeliveryTool(nil),
		NewReplayEDIMessageTool(nil),
		NewReprocessEDIInboundFilesTool(nil),
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

	properties := NewRetryEDIMessageDeliveryTool(nil).ParamSchema()["properties"].(map[string]any)
	assert.Equal(t, permission.ResourceEDI.String(),
		toolschema.SubsetResource(properties[paramEDIMessageIDs].(map[string]any)),
		"a person approving may untick messages")
}

func TestRetryEDIMessageDelivery_PreviewsEachMessageAndRefusalWithoutItsContent(t *testing.T) {
	t.Parallel()

	failed := outboundMessage(edimessage.MessageDeliveryStatusFailed)
	sent := outboundMessage(edimessage.MessageDeliveryStatusSent)
	deliveries := &fakeDeliveries{
		messages: map[pulid.ID]*edimessage.EDIMessage{failed.ID: failed, sent.ID: sent},
		guard:    &agenttoolstest.WriteGuard{},
	}
	tool := NewRetryEDIMessageDeliveryTool(deliveries).(*RetryEDIMessageDeliveryTool)
	args := map[string]any{
		paramEDIMessageIDs: []any{failed.ID.String(), sent.ID.String(), failed.ID.String()},
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		deliveries.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	require.Len(t, preview.Changes, 2, "a message named twice is sent once")
	refused := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, sent.ID, refused.EntityID, "refusals come first")
	assert.Contains(
		t,
		agenttoolstest.FieldByPath(t, refused, "outcome").After,
		"Only queued, failed",
	)
	queued := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, "Queued", agenttoolstest.FieldByPath(t, queued, "deliveryStatus").After)
	require.NotNil(t, queued.Message)
	assert.Equal(t, agentpreview.MessageChannelEDI, queued.Message.Channel)
	assert.NotContains(t, queued.Message.Body, "SECRET-RATE", "the document's content stays out")
	assert.Contains(t, preview.Summary, "1 of 2")

	require.NoError(t, tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(args)))
	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)),
		ErrEDIDecisionNeedsAPerson,
	)

	result, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ApprovedParams(args))
	require.NoError(t, err)
	require.NotNil(t, deliveries.retried)
	assert.Equal(t, []pulid.ID{failed.ID, sent.ID}, deliveries.retried.MessageIDs)
	assert.Contains(t, result.Name, "2 of 2")
}

func TestReplayEDIMessage_SendsADeliveredDocumentAgainOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	sent := outboundMessage(edimessage.MessageDeliveryStatusSent)
	failed := outboundMessage(edimessage.MessageDeliveryStatusFailed)
	deliveries := &fakeDeliveries{
		messages: map[pulid.ID]*edimessage.EDIMessage{sent.ID: sent, failed.ID: failed},
		guard:    &agenttoolstest.WriteGuard{},
	}
	tool := NewReplayEDIMessageTool(deliveries).(*ReplayEDIMessageTool)

	target, ok := tool.Target(map[string]any{paramEDIMessageID: sent.ID.String()})
	require.True(t, ok)
	assert.Equal(
		t,
		agenttoolports.ToolTarget{Resource: permission.ResourceEDI, ID: sent.ID},
		target,
	)

	args := map[string]any{paramEDIMessageID: sent.ID.String()}
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		deliveries.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(3), *change.Version)
	assert.Equal(t, "Queued", agenttoolstest.FieldByPath(t, change, "deliveryStatus").After)

	refused := map[string]any{paramEDIMessageID: failed.ID.String()}
	require.Error(t, tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(refused)))

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)),
		ErrEDIDecisionNeedsAPerson,
	)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(args)))
	require.NotNil(t, deliveries.replayed)
	assert.Equal(t, sent.ID, deliveries.replayed.MessageID)
}

type fakeReprocessor struct {
	files       map[pulid.ID]*ediinboundfile.EDIInboundFile
	guard       *agenttoolstest.WriteGuard
	reprocessed *ediinboundservice.BulkReprocessInboundFilesRequest
}

func (f *fakeReprocessor) GetInboundFile(
	_ context.Context,
	req edimessageports.GetEDIInboundFileByIDRequest,
) (*ediinboundfile.EDIInboundFile, error) {
	file, ok := f.files[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("EDIInboundFile not found")
	}

	return file, nil
}

func (f *fakeReprocessor) BulkReprocessInboundFiles(
	_ context.Context,
	req *ediinboundservice.BulkReprocessInboundFilesRequest,
) (*edibulk.Result, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.reprocessed = req

	return &edibulk.Result{
		Succeeded: req.FileIDs[:1],
		Failed: []edibulk.Failure{{
			ID: req.FileIDs[len(req.FileIDs)-1], Error: "Only quarantined files",
		}},
	}, nil
}

func TestReprocessEDIInboundFiles_PreviewsWhichFilesWouldRunAgain(t *testing.T) {
	t.Parallel()

	quarantined := &ediinboundfile.EDIInboundFile{
		ID:            pulid.MustNew("ediinf_"),
		FileName:      "204_0921.x12",
		Status:        ediinboundfile.InboundFileStatusQuarantined,
		FailureReason: "partner could not be resolved",
		Version:       5,
	}
	processed := &ediinboundfile.EDIInboundFile{
		ID:       pulid.MustNew("ediinf_"),
		FileName: "214_0921.x12",
		Status:   ediinboundfile.InboundFileStatusProcessed,
	}
	files := &fakeReprocessor{
		files: map[pulid.ID]*ediinboundfile.EDIInboundFile{
			quarantined.ID: quarantined,
			processed.ID:   processed,
		},
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewReprocessEDIInboundFilesTool(files).(*ReprocessEDIInboundFilesTool)
	args := map[string]any{
		paramEDIInboundFileIDs: []any{quarantined.ID.String(), processed.ID.String()},
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		files.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	require.Len(t, preview.Changes, 2)
	refused := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, processed.ID, refused.EntityID)
	assert.Contains(t, agenttoolstest.FieldByPath(t, refused, "outcome").After, "Only quarantined")
	again := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, "204_0921.x12", again.Label)
	assert.True(t, preview.Partial, "what each transaction becomes is decided when it runs")

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)),
		ErrEDIDecisionNeedsAPerson,
	)
	result, err := tool.ExecuteWithResult(t.Context(), agenttoolstest.ApprovedParams(args))
	require.NoError(t, err)
	require.NotNil(t, files.reprocessed)
	assert.Contains(t, result.Name, "1 of 2")
	assert.Contains(t, result.Name, "Only quarantined files")
}

func TestEDIBulkTools_RefuseMoreThanTheServiceTakes(t *testing.T) {
	t.Parallel()

	ids := make([]any, 0, edibulk.MaxItems+1)
	for range edibulk.MaxItems + 1 {
		ids = append(ids, pulid.MustNew("edimsg_").String())
	}

	tool := NewRetryEDIMessageDeliveryTool(&fakeDeliveries{})
	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.AgentParamsFor(map[string]any{paramEDIMessageIDs: ids}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "split it into smaller calls")
}
