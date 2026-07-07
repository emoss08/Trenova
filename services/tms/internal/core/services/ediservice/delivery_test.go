package ediservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
	message         *edi.EDIMessage
	profile         *edi.EDICommunicationProfile
	messageRepo     *mocks.MockEDIMessageRepository
	profileRepo     *mocks.MockEDICommunicationProfileRepository
	partnerRepo     *mocks.MockEDIPartnerRepository
	tenderChange    *mocks.MockEDITenderChangeRepository
	recipientRepo   *mocks.MockEDITenderRecipientRepository
	transport       *mocks.MockEDITransportDispatcher
	workflowStarter *mocks.MockWorkflowStarter
}

func newDeliveryFixture(t *testing.T) *deliveryFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &deliveryFixture{
		message: &edi.EDIMessage{
			ID:             pulid.MustNew("edimsg_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EDIPartnerID:   partnerID,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.MessageStatusGenerated,
			TransactionSet: edi.TransactionSet204,
			DeliveryStatus: edi.MessageDeliveryStatusQueued,
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
		messageRepo:     mocks.NewMockEDIMessageRepository(t),
		profileRepo:     mocks.NewMockEDICommunicationProfileRepository(t),
		partnerRepo:     mocks.NewMockEDIPartnerRepository(t),
		tenderChange:    mocks.NewMockEDITenderChangeRepository(t),
		recipientRepo:   mocks.NewMockEDITenderRecipientRepository(t),
		transport:       mocks.NewMockEDITransportDispatcher(t),
		workflowStarter: mocks.NewMockWorkflowStarter(t),
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
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
		GetMessageByID(mock.Anything, repositories.GetEDIMessageByIDRequest{
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
			repositories.GetEDITenderChangeByOutboundMessageIDRequest{
				OutboundMessageID: f.message.ID,
				TenantInfo:        messageTenantInfo(f.message),
			},
		).
		Return(nil, errortypes.NewNotFoundError("EDITenderChange")).
		Once()
	f.profileRepo.EXPECT().
		GetActiveProfileByPartner(
			mock.Anything,
			repositories.GetActiveEDICommunicationProfileByPartnerRequest{
				PartnerID:  f.message.EDIPartnerID,
				TenantInfo: messageTenantInfo(f.message),
				Methods:    deliverableMethods,
			},
		).
		Return(f.profile, nil).
		Once()
}

func (f *deliveryFixture) expectDeliveryUpdate(
	status edi.MessageDeliveryStatus,
) {
	f.messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.ID == f.message.ID && req.DeliveryStatus == status
			}),
		).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateEDIMessageDeliveryRequest) (*edi.EDIMessage, error) {
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
	fixture.expectDeliveryUpdate(edi.MessageDeliveryStatusSending)
	fixture.transport.EXPECT().
		Deliver(
			mock.Anything,
			edi.ConnectionMethodSFTP,
			mock.MatchedBy(func(req *services.EDITransportRequest) bool {
				return req.Profile == fixture.profile && req.Contents == fixture.message.RawX12
			}),
		).
		Return(&services.EDITransportResult{RemotePath: "/outbound/file.x12"}, nil).
		Once()
	fixture.expectDeliveryUpdate(edi.MessageDeliveryStatusSent)

	recipient := &edi.TenderRecipient{
		ID:     pulid.MustNew("editr_"),
		Status: edi.TenderRecipientStatusActive,
	}
	change := &edi.TenderChange{
		ID:                pulid.MustNew("editch_"),
		Status:            edi.TenderChangeStatusQueued,
		OutboundMessageID: fixture.message.ID,
		Recipient:         recipient,
	}
	fixture.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(
			mock.Anything,
			repositories.GetEDITenderChangeByOutboundMessageIDRequest{
				OutboundMessageID: fixture.message.ID,
				TenantInfo:        messageTenantInfo(fixture.message),
			},
		).
		Return(change, nil).
		Once()
	fixture.tenderChange.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *edi.TenderChange) bool {
			return updated.Status == edi.TenderChangeStatusSent && updated.FailureReason == ""
		})).
		RunAndReturn(func(_ context.Context, updated *edi.TenderChange) (*edi.TenderChange, error) {
			return updated, nil
		}).
		Once()
	fixture.recipientRepo.EXPECT().
		UpdateTenderRecipient(mock.Anything, mock.MatchedBy(func(updated *edi.TenderRecipient) bool {
			return updated.ID == recipient.ID &&
				updated.BaselineStatus == edi.TenderRecipientBaselineStatusSent
		})).
		RunAndReturn(func(_ context.Context, updated *edi.TenderRecipient) (*edi.TenderRecipient, error) {
			return updated, nil
		}).
		Once()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.NoError(t, err)
	require.Equal(t, edi.MessageDeliveryStatusSent, result.DeliveryStatus)
	require.Equal(t, "/outbound/file.x12", result.RemotePath)
}

func TestService_DeliverMessage_TransportFailureMarksFailedAndReturnsError(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	transportErr := errors.New("connect SFTP server: connection refused")
	fixture.expectLoadMessage()
	fixture.expectResolveProfileByPartner()
	fixture.expectDeliveryUpdate(edi.MessageDeliveryStatusSending)
	fixture.transport.EXPECT().
		Deliver(mock.Anything, edi.ConnectionMethodSFTP, mock.Anything).
		Return(nil, transportErr).
		Once()
	fixture.messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.DeliveryStatus == edi.MessageDeliveryStatusFailed &&
					req.DeliveryLastError == transportErr.Error()
			}),
		).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateEDIMessageDeliveryRequest) (*edi.EDIMessage, error) {
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
	fixture.message.DeliveryStatus = edi.MessageDeliveryStatusSent
	fixture.message.DeliveryRemotePath = "/outbound/sent.x12"
	fixture.expectLoadMessage()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.NoError(t, err)
	require.Equal(t, edi.MessageDeliveryStatusSent, result.DeliveryStatus)
	require.Equal(t, "/outbound/sent.x12", result.RemotePath)
}

func TestService_DeliverMessage_InboundMessageIsRejected(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.Direction = edi.DocumentDirectionInbound
	fixture.expectLoadMessage()

	result, err := fixture.service.DeliverMessage(t.Context(), fixture.payload())

	require.Error(t, err)
	require.Nil(t, result)
}

func TestService_MarkMessageDeadLettered_UpdatesMessageAndFailsTenderChange(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.message.DeliveryStatus = edi.MessageDeliveryStatusFailed
	fixture.expectLoadMessage()
	fixture.expectDeliveryUpdate(edi.MessageDeliveryStatusDeadLettered)

	change := &edi.TenderChange{
		ID:                pulid.MustNew("editch_"),
		Status:            edi.TenderChangeStatusQueued,
		OutboundMessageID: fixture.message.ID,
	}
	fixture.tenderChange.EXPECT().
		GetTenderChangeByOutboundMessageID(mock.Anything, mock.Anything).
		Return(change, nil).
		Once()
	fixture.tenderChange.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *edi.TenderChange) bool {
			return updated.Status == edi.TenderChangeStatusFailed &&
				updated.FailureReason == "delivery exhausted"
		})).
		RunAndReturn(func(_ context.Context, updated *edi.TenderChange) (*edi.TenderChange, error) {
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
	fixture.message.DeliveryStatus = edi.MessageDeliveryStatusSent
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
	fixture.message.DeliveryStatus = edi.MessageDeliveryStatusDeadLettered
	fixture.expectLoadMessage()
	fixture.expectResolveProfileByPartner()
	fixture.expectDeliveryUpdate(edi.MessageDeliveryStatusQueued)
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
	require.Equal(t, edi.MessageDeliveryStatusQueued, message.DeliveryStatus)
}

func TestService_QueueMessageForDelivery_SkipsInternalPartners(t *testing.T) {
	t.Parallel()

	fixture := newDeliveryFixture(t)
	fixture.partnerRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetEDIPartnerByIDRequest{
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
