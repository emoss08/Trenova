package ediservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/testutil/mocks/edimessageportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/editransportportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type deliveryFixture struct {
	service         *Service
	message         *edimessage.EDIMessage
	profile         *edi.EDICommunicationProfile
	messageRepo     *edimessageportsmock.MockEDIMessageRepository
	profileRepo     *edipartnerportsmock.MockEDICommunicationProfileRepository
	partnerRepo     *edipartnerportsmock.MockEDIPartnerRepository
	tenderChange    *editenderportsmock.MockEDITenderChangeRepository
	recipientRepo   *editenderportsmock.MockEDITenderRecipientRepository
	transport       *editransportportsmock.MockEDITransportDispatcher
	workflowStarter *kernelportsmock.MockWorkflowStarter
}

func newDeliveryFixture(t *testing.T) *deliveryFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &deliveryFixture{
		message: &edimessage.EDIMessage{
			ID:             pulid.MustNew("edimsg_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EDIPartnerID:   partnerID,
			Direction:      edidocument.DocumentDirectionOutbound,
			Status:         edimessage.MessageStatusGenerated,
			TransactionSet: edidocument.TransactionSet204,
			DeliveryStatus: edimessage.MessageDeliveryStatusQueued,
			RawX12:         "ISA*...~",
		},
		profile: &edi.EDICommunicationProfile{
			ID:             pulid.MustNew("edicp_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EDIPartnerID:   partnerID,
			Method:         edi.ConnectionMethodSFTP,
			Config:         map[string]any{"host": "sftp.example.com"},
		},
		messageRepo:     edimessageportsmock.NewMockEDIMessageRepository(t),
		profileRepo:     edipartnerportsmock.NewMockEDICommunicationProfileRepository(t),
		partnerRepo:     edipartnerportsmock.NewMockEDIPartnerRepository(t),
		tenderChange:    editenderportsmock.NewMockEDITenderChangeRepository(t),
		recipientRepo:   editenderportsmock.NewMockEDITenderRecipientRepository(t),
		transport:       editransportportsmock.NewMockEDITransportDispatcher(t),
		workflowStarter: kernelportsmock.NewMockWorkflowStarter(t),
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
		metrics:             metrics.NewEDI(nil, zap.NewNop(), false),
		messageRepo:         fixture.messageRepo,
		profileRepo:         fixture.profileRepo,
		partnerRepo:         fixture.partnerRepo,
		tenderChangeRepo:    fixture.tenderChange,
		tenderRecipientRepo: fixture.recipientRepo,
		transport:           fixture.transport,
		workflowStarter:     fixture.workflowStarter,
	}
	return fixture
}

func (f *deliveryFixture) payload() *DeliverEDIMessageWorkflowPayload {
	return &DeliverEDIMessageWorkflowPayload{
		MessageID: f.message.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: f.message.OrganizationID,
			BuID:  f.message.BusinessUnitID,
		},
	}
}

func (f *deliveryFixture) expectLoadMessage() {
	f.messageRepo.EXPECT().
		GetMessageByID(mock.Anything, edimessageports.GetEDIMessageByIDRequest{
			ID:         f.message.ID,
			TenantInfo: messageTenantInfo(f.message),
		}).
		Return(f.message, nil).
		Once()
}

func (f *deliveryFixture) expectResolveProfileByPartner() {
	f.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(
			mock.Anything,
			editenderports.GetEDITenderChangeByOutboundMessageIDRequest{
				OutboundMessageID: f.message.ID,
				TenantInfo:        messageTenantInfo(f.message),
			},
		).
		Return(nil, errortypes.NewNotFoundError("EDITenderChange")).
		Once()
	f.profileRepo.EXPECT().
		GetActiveProfileByPartner(
			mock.Anything,
			edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest{
				PartnerID:  f.message.EDIPartnerID,
				TenantInfo: messageTenantInfo(f.message),
				Methods:    deliverableMethods,
			},
		).
		Return(f.profile, nil).
		Once()
}

func (f *deliveryFixture) expectDeliveryUpdate(
	status edimessage.MessageDeliveryStatus,
) {
	f.messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *edimessageports.UpdateEDIMessageDeliveryRequest) bool {
				return req.ID == f.message.ID && req.DeliveryStatus == status
			}),
		).
		RunAndReturn(func(_ context.Context, req *edimessageports.UpdateEDIMessageDeliveryRequest) (*edimessage.EDIMessage, error) {
			updated := *f.message
			updated.DeliveryStatus = req.DeliveryStatus
			updated.DeliveryRemotePath = req.DeliveryRemotePath
			return &updated, nil
		}).
		Once()
}

func TestService_DeliverMessage_SuccessMarksSentAndCompletesTenderChange(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.expectLoadMessage()
	fixture.expectResolveProfileByPartner()
	fixture.expectDeliveryUpdate(edimessage.MessageDeliveryStatusSending)
	fixture.transport.EXPECT().
		Deliver(
			mock.Anything,
			edi.ConnectionMethodSFTP,
			mock.MatchedBy(func(req *editransportports.EDITransportRequest) bool {
				return req.Profile == fixture.profile && req.Contents == fixture.message.RawX12
			}),
		).
		Return(&editransportports.EDITransportResult{RemotePath: "/outbound/file.x12"}, nil).
		Once()
	fixture.expectDeliveryUpdate(edimessage.MessageDeliveryStatusSent)

	recipient := &editenderchange.TenderRecipient{
		ID:     pulid.MustNew("editr_"),
		Status: editenderchange.TenderRecipientStatusActive,
	}
	change := &editenderchange.TenderChange{
		ID:                pulid.MustNew("editch_"),
		Status:            editenderchange.TenderChangeStatusQueued,
		OutboundMessageID: fixture.message.ID,
		Recipient:         recipient,
	}
	fixture.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(
			mock.Anything,
			editenderports.GetEDITenderChangeByOutboundMessageIDRequest{
				OutboundMessageID: fixture.message.ID,
				TenantInfo:        messageTenantInfo(fixture.message),
			},
		).
		Return(change, nil).
		Once()
	fixture.tenderChange.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *editenderchange.TenderChange) bool {
			return updated.Status == editenderchange.TenderChangeStatusSent &&
				updated.FailureReason == ""
		})).
		RunAndReturn(func(_ context.Context, updated *editenderchange.TenderChange) (*editenderchange.TenderChange, error) {
			return updated, nil
		}).
		Once()
	fixture.recipientRepo.EXPECT().
		UpdateTenderRecipient(mock.Anything, mock.MatchedBy(func(updated *editenderchange.TenderRecipient) bool {
			return updated.ID == recipient.ID &&
				updated.BaselineStatus == editenderchange.TenderRecipientBaselineStatusSent
		})).
		RunAndReturn(func(_ context.Context, updated *editenderchange.TenderRecipient) (*editenderchange.TenderRecipient, error) {
			return updated, nil
		}).
		Once()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.NoError(t, err)
	require.Equal(t, edimessage.MessageDeliveryStatusSent, result.DeliveryStatus)
	require.Equal(t, "/outbound/file.x12", result.RemotePath)
}

func TestService_DeliverMessage_TransportFailureMarksFailedAndReturnsError(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	transportErr := errors.New("connect SFTP server: connection refused")
	fixture.expectLoadMessage()
	fixture.expectResolveProfileByPartner()
	fixture.expectDeliveryUpdate(edimessage.MessageDeliveryStatusSending)
	fixture.transport.EXPECT().
		Deliver(mock.Anything, edi.ConnectionMethodSFTP, mock.Anything).
		Return(nil, transportErr).
		Once()
	fixture.messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *edimessageports.UpdateEDIMessageDeliveryRequest) bool {
				return req.DeliveryStatus == edimessage.MessageDeliveryStatusFailed &&
					req.DeliveryLastError == transportErr.Error()
			}),
		).
		RunAndReturn(func(_ context.Context, req *edimessageports.UpdateEDIMessageDeliveryRequest) (*edimessage.EDIMessage, error) {
			updated := *fixture.message
			updated.DeliveryStatus = req.DeliveryStatus
			return &updated, nil
		}).
		Once()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.ErrorIs(t, err, transportErr)
	require.Nil(t, result)
}

func TestService_DeliverMessage_AlreadySentIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.DeliveryStatus = edimessage.MessageDeliveryStatusSent
	fixture.message.DeliveryRemotePath = "/outbound/sent.x12"
	fixture.expectLoadMessage()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.NoError(t, err)
	require.Equal(t, edimessage.MessageDeliveryStatusSent, result.DeliveryStatus)
	require.Equal(t, "/outbound/sent.x12", result.RemotePath)
}

func TestService_DeliverMessage_InboundMessageIsRejected(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.Direction = edidocument.DocumentDirectionInbound
	fixture.expectLoadMessage()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.Error(t, err)
	require.Nil(t, result)
}

func TestService_MarkMessageDeadLettered_UpdatesMessageAndFailsTenderChange(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.DeliveryStatus = edimessage.MessageDeliveryStatusFailed
	fixture.expectLoadMessage()
	fixture.expectDeliveryUpdate(edimessage.MessageDeliveryStatusDeadLettered)

	change := &editenderchange.TenderChange{
		ID:                pulid.MustNew("editch_"),
		Status:            editenderchange.TenderChangeStatusQueued,
		OutboundMessageID: fixture.message.ID,
	}
	fixture.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(mock.Anything, mock.Anything).
		Return(change, nil).
		Once()
	fixture.tenderChange.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *editenderchange.TenderChange) bool {
			return updated.Status == editenderchange.TenderChangeStatusFailed &&
				updated.FailureReason == "delivery exhausted"
		})).
		RunAndReturn(func(_ context.Context, updated *editenderchange.TenderChange) (*editenderchange.TenderChange, error) {
			return updated, nil
		}).
		Once()

	err := fixture.service.MarkMessageDeadLettered(t.Context(), &MarkEDIMessageDeadLetteredPayload{
		MessageID:  fixture.message.ID,
		TenantInfo: messageTenantInfo(fixture.message),
		Reason:     "delivery exhausted",
	})

	require.NoError(t, err)
}

func TestService_RetryMessageDelivery_RejectsSentMessages(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.DeliveryStatus = edimessage.MessageDeliveryStatusSent
	fixture.expectLoadMessage()

	message, err := fixture.service.RetryMessageDelivery(t.Context(), &RetryMessageDeliveryRequest{
		MessageID:  fixture.message.ID,
		TenantInfo: messageTenantInfo(fixture.message),
	})

	require.Error(t, err)
	require.Nil(t, message)
}

func TestService_RetryMessageDelivery_RequeuesFailedMessageAndStartsWorkflow(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.DeliveryStatus = edimessage.MessageDeliveryStatusDeadLettered
	fixture.expectLoadMessage()
	fixture.expectResolveProfileByPartner()
	fixture.expectDeliveryUpdate(edimessage.MessageDeliveryStatusQueued)
	fixture.workflowStarter.EXPECT().Enabled().Return(true).Once()
	fixture.workflowStarter.EXPECT().
		StartWorkflow(mock.Anything, mock.MatchedBy(func(options client.StartWorkflowOptions) bool {
			return options.ID == buildDeliverMessageWorkflowID(fixture.message.ID)
		}), temporaltype.DeliverEDIMessageWorkflowName, mock.Anything).
		Return(nil, nil).
		Once()

	message, err := fixture.service.RetryMessageDelivery(t.Context(), &RetryMessageDeliveryRequest{
		MessageID:  fixture.message.ID,
		TenantInfo: messageTenantInfo(fixture.message),
	})

	require.NoError(t, err)
	require.Equal(t, edimessage.MessageDeliveryStatusQueued, message.DeliveryStatus)
}

func TestService_QueueMessageForDelivery_SkipsInternalPartners(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.partnerRepo.EXPECT().
		GetByID(mock.Anything, edipartnerports.GetEDIPartnerByIDRequest{
			ID:         fixture.message.EDIPartnerID,
			TenantInfo: messageTenantInfo(fixture.message),
		}).
		Return(&edi.EDIPartner{Kind: edi.PartnerKindInternal}, nil).
		Once()

	err := fixture.service.queueMessageForDelivery(t.Context(), fixture.message)

	require.NoError(t, err)
}

func TestService_QueueMessageForDelivery_SkipsPartnersWithoutDeliverableProfile(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.partnerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&edi.EDIPartner{Kind: edi.PartnerKindExternal}, nil).
		Once()
	fixture.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDITenderChange")).
		Once()
	fixture.profileRepo.EXPECT().
		GetActiveProfileByPartner(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDICommunicationProfile")).
		Once()

	err := fixture.service.queueMessageForDelivery(t.Context(), fixture.message)

	require.NoError(t, err)
}
