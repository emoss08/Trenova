package ediinboundservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const ackInterchangeHeader = "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
	"*260107*1200*^*00401*000000101*0*P*>~"

func functionalAck(set, body string, segments int) string {
	version := "004010"
	header := "ST*" + set + "*0001~"
	if set == "999" {
		version = "005010X231A1"
		header = "ST*999*0001*005010X231A1~"
	}
	return ackInterchangeHeader +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*101*X*" + version + "~" +
		header +
		body +
		fmt.Sprintf("SE*%d*0001~", segments) +
		"GE*1*101~" +
		"IEA*1*000000101~"
}

type flowAckFixture struct {
	*inboundFixture
	instance *ladingproject.FlowInstance
	saved    *ladingproject.FlowInstance
}

func newFlowAckFixture(t *testing.T, raw string) *flowAckFixture {
	t.Helper()

	inbound := newInboundFixture(t, raw)
	inbound.expectFileLoadAndUpdates(t)
	inbound.expectPartnerLoad()
	inbound.expectActiveVersion(t)
	program, err := inbound.service.lading.Active(
		t.Context(),
		inbound.tenantInfo(),
		inbound.partner.ID,
	)
	require.NoError(t, err)
	flow, _ := program.Manifest.Receiving("204")
	require.NotNil(t, flow)

	state, err := sonic.Marshal(map[string]any{
		"key":   "KVW-1",
		"load":  map[string]any{"exists": true, "state": lading.LoadDispatched},
		"sends": 2,
		"pending": []map[string]any{{
			"id":      "s2",
			"set":     "214",
			"control": "0042",
			"due":     time.Now().Add(time.Hour).Unix(),
			"resends": 1,
		}},
	})
	require.NoError(t, err)
	fixture := &flowAckFixture{
		inboundFixture: inbound,
		instance: &ladingproject.FlowInstance{
			ID:             pulid.MustNew("edfi_"),
			OrganizationID: inbound.file.OrganizationID,
			BusinessUnitID: inbound.file.BusinessUnitID,
			EDIPartnerID:   inbound.partner.ID,
			Flow:           flow.Name,
			Key:            "KVW-1",
			State:          state,
			Status:         ladingproject.FlowInstanceStatusOpen,
		},
	}
	return fixture
}

func (f *flowAckFixture) flowMessage() *edimessage.EDIMessage {
	return &edimessage.EDIMessage{
		ID:                       pulid.MustNew("edimsg_"),
		OrganizationID:           f.file.OrganizationID,
		BusinessUnitID:           f.file.BusinessUnitID,
		EDIPartnerID:             f.partner.ID,
		Direction:                edidocument.DocumentDirectionOutbound,
		TransactionSet:           edidocument.TransactionSet214,
		InterchangeControlNumber: "000000077",
		GroupControlNumber:       "42",
		TransactionControlNumber: "0042",
		FlowInstanceID:           f.instance.ID,
		FlowSendID:               "s2",
		AckStatus:                edimessage.MessageAcknowledgmentStatusPending,
	}
}

func (f *flowAckFixture) expectFlowStep() {
	f.flowRepo.EXPECT().
		GetInstance(mock.Anything, ladingports.GetFlowInstanceRequest{
			TenantInfo: f.tenantInfo(),
			ID:         f.instance.ID,
		}).
		Return(f.instance, nil).
		Once()
	f.flowRepo.EXPECT().SaveInstance(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, instance *ladingproject.FlowInstance) (*ladingproject.FlowInstance, error) {
			f.saved = instance
			return instance, nil
		}).
		Once()
	f.flowRepo.EXPECT().AppendEvent(mock.Anything, mock.Anything).Return(nil).Maybe()
}

func (f *flowAckFixture) expectAckRecorded(
	t *testing.T,
	message *edimessage.EDIMessage,
	status edimessage.MessageAcknowledgmentStatus,
) *edimessageports.UpdateEDIMessageAcknowledgmentRequest {
	t.Helper()
	recorded := &edimessageports.UpdateEDIMessageAcknowledgmentRequest{}
	f.messageRepo.EXPECT().
		UpdateMessageAcknowledgment(mock.Anything, mock.MatchedBy(
			func(req *edimessageports.UpdateEDIMessageAcknowledgmentRequest) bool {
				return req.ID == message.ID && req.AckStatus == status
			},
		)).
		RunAndReturn(func(_ context.Context, req *edimessageports.UpdateEDIMessageAcknowledgmentRequest) (*edimessage.EDIMessage, error) {
			*recorded = *req
			return message, nil
		}).
		Once()
	return recorded
}

func (f *flowAckFixture) pending(t *testing.T) []lading.Pending {
	t.Helper()
	require.NotNil(t, f.saved, "the flow heard the acknowledgment")
	var summary lading.FlowSummary
	require.NoError(t, sonic.Unmarshal(f.saved.State, &summary))
	return summary.Pending
}

func (f *flowAckFixture) process(t *testing.T) *ediinboundfile.EDIInboundFile {
	t.Helper()
	file, err := f.service.ProcessInboundFile(t.Context(), &ProcessInboundFileRequest{
		FileID:     f.file.ID,
		TenantInfo: f.tenantInfo(),
	})
	require.NoError(t, err)
	return file
}

func (f *flowAckFixture) expectOriginal(message *edimessage.EDIMessage) {
	f.messageRepo.EXPECT().
		GetOutboundMessageForAck(mock.Anything, mock.MatchedBy(
			func(req *edimessageports.GetEDIOutboundMessageForAckRequest) bool {
				return req.TransactionControlNumber == "0042" && req.GroupControlNumber == "42"
			},
		)).
		Return(message, nil).
		Once()
}

func TestProcessInboundFile_AnAccepted997ClearsTheFlowsPendingSend(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck(
		"997",
		"AK1*QM*42~AK2*214*0042~AK5*A~AK9*A*1*1*1~",
		6,
	))
	fixture.expectInboundDocumentType(edidocument.TransactionSet997)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	fixture.expectOriginal(message)
	fixture.expectAckRecorded(t, message, edimessage.MessageAcknowledgmentStatusAccepted)
	fixture.expectFlowStep()

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Empty(t, fixture.pending(t))
	assert.Equal(t, ladingproject.FlowInstanceStatusOpen, fixture.saved.Status)
}

func TestProcessInboundFile_ARejected997EscalatesTheFlowWithItsCodes(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck(
		"997",
		"AK1*QM*42~AK2*214*0042~AK3*N1*3**8~AK5*R*5~AK9*R*1*1*0~",
		7,
	))
	fixture.expectInboundDocumentType(edidocument.TransactionSet997)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	fixture.expectOriginal(message)
	recorded := fixture.expectAckRecorded(
		t,
		message,
		edimessage.MessageAcknowledgmentStatusRejected,
	)
	fixture.expectFlowStep()

	fixture.process(t)

	assert.Contains(t, recorded.AckLastError, "acknowledgment code R")
	assert.Contains(t, recorded.AckLastError, "syntax error codes 5")
	assert.Empty(t, fixture.pending(t))
	assert.Equal(t, ladingproject.FlowInstanceStatusEscalated, fixture.saved.Status)
	require.NotNil(t, fixture.saved.Reason)
	assert.Contains(t, fixture.saved.Reason.String(), "syntax error codes 5")
	assert.Contains(t, fixture.saved.Reason.String(), "0042")
}

func TestProcessInboundFile_ARejected999EscalatesTheFlow(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck(
		"999",
		"AK1*QM*42*004010~AK2*214*0042~IK5*R*I5~AK9*R*1*1*0~",
		6,
	))
	fixture.expectInboundDocumentType(edidocument.TransactionSet999)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	fixture.expectOriginal(message)
	recorded := fixture.expectAckRecorded(
		t,
		message,
		edimessage.MessageAcknowledgmentStatusRejected,
	)
	fixture.expectFlowStep()

	fixture.process(t)

	assert.Contains(t, recorded.AckLastError, "syntax error codes I5")
	assert.Empty(t, fixture.pending(t))
	assert.Equal(t, ladingproject.FlowInstanceStatusEscalated, fixture.saved.Status)
}

func TestProcessInboundFile_ARedelivered997LeavesTheMessageAndFlowAlone(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck(
		"997",
		"AK1*QM*42~AK2*214*0042~AK5*A~AK9*A*1*1*1~",
		6,
	))
	fixture.expectInboundDocumentType(edidocument.TransactionSet997)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	message.AckStatus = edimessage.MessageAcknowledgmentStatusAccepted
	fixture.expectOriginal(message)

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Nil(t, fixture.saved)
}

func TestProcessInboundFile_AnAckForAMessageNoFlowSentOnlyRecordsIt(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck(
		"997",
		"AK1*QM*42~AK2*214*0042~AK5*R~AK9*R*1*1*0~",
		6,
	))
	fixture.expectInboundDocumentType(edidocument.TransactionSet997)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	message.FlowInstanceID = ""
	fixture.expectOriginal(message)
	fixture.expectAckRecorded(t, message, edimessage.MessageAcknowledgmentStatusRejected)

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Nil(t, fixture.saved)
}

func TestProcessInboundFile_AGroupLevel997AcknowledgesEveryMessageInTheGroup(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, functionalAck("997", "AK1*QM*00042~AK9*A*1*1*1~", 4))
	fixture.expectInboundDocumentType(edidocument.TransactionSet997)
	fixture.expectMessageCreate()
	message := fixture.flowMessage()
	fixture.messageRepo.EXPECT().
		ListUnacknowledgedOutboundMessages(mock.Anything, mock.MatchedBy(
			func(req *edimessageports.ListUnacknowledgedOutboundMessagesRequest) bool {
				return req.PartnerID == fixture.partner.ID &&
					req.GroupControlNumber == "42" &&
					req.InterchangeControlNumber == "" &&
					req.TransactionSet == edidocument.TransactionSet214
			},
		)).
		Return([]*edimessage.EDIMessage{message}, nil).
		Once()
	fixture.expectAckRecorded(t, message, edimessage.MessageAcknowledgmentStatusAccepted)
	fixture.expectFlowStep()

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Empty(t, fixture.pending(t))
}

func TestProcessInboundFile_ATA1RejectionRejectsEveryMessageInTheInterchange(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, ackInterchangeHeader+
		"TA1*77*260107*1200*R*022~"+
		"IEA*0*000000101~")
	sent := fixture.flowMessage()
	other := fixture.flowMessage()
	other.FlowInstanceID = ""
	other.TransactionControlNumber = "0043"
	fixture.messageRepo.EXPECT().
		ListUnacknowledgedOutboundMessages(mock.Anything, mock.MatchedBy(
			func(req *edimessageports.ListUnacknowledgedOutboundMessagesRequest) bool {
				return req.PartnerID == fixture.partner.ID &&
					req.InterchangeControlNumber == "000000077" &&
					req.GroupControlNumber == ""
			},
		)).
		Return([]*edimessage.EDIMessage{sent, other}, nil).
		Once()
	recorded := fixture.expectAckRecorded(t, sent, edimessage.MessageAcknowledgmentStatusRejected)
	fixture.expectAckRecorded(t, other, edimessage.MessageAcknowledgmentStatusRejected)
	fixture.expectFlowStep()

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Contains(t, recorded.AckLastError, "TA1 note code 022")
	assert.Empty(t, fixture.pending(t))
	assert.Equal(t, ladingproject.FlowInstanceStatusEscalated, fixture.saved.Status)
}

func TestProcessInboundFile_AnAcceptingTA1ChangesNothing(t *testing.T) {
	t.Parallel()

	fixture := newFlowAckFixture(t, ackInterchangeHeader+
		"TA1*000000077*260107*1200*A*000~"+
		"IEA*0*000000101~")

	file := fixture.process(t)

	assert.Equal(t, ediinboundfile.InboundFileStatusProcessed, file.Status, file.FailureReason)
	assert.Nil(t, fixture.saved)
}

func TestInterchangeControlNumber_PadsToTheEnvelopeWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "000000077", interchangeControlNumber("77"))
	assert.Equal(t, "000000077", interchangeControlNumber("000000077"))
	assert.Equal(t, "42", groupControlNumber("00042"))
	assert.Equal(t, "0", groupControlNumber("000"))
	assert.Empty(t, interchangeControlNumber(" "))
}
