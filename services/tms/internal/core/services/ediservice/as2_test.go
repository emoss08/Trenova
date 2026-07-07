package ediservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func pendingAS2Message(messageID string) *edi.EDIMessage {
	return &edi.EDIMessage{
		ID:             pulid.MustNew("edimsg_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Direction:      edi.DocumentDirectionOutbound,
		DeliveryStatus: edi.MessageDeliveryStatusSending,
		AS2MessageID:   messageID,
		AS2MIC:         "q1w2e3r4, sha256",
	}
}

func TestApplyAS2MDNResolvesPendingDelivery(t *testing.T) {
	t.Parallel()

	messageID := "<pending-123@trenova.as2>"
	message := pendingAS2Message(messageID)

	messageRepo := mocks.NewMockEDIMessageRepository(t)
	messageRepo.EXPECT().
		GetOutboundMessageByAS2MessageID(mock.Anything, messageID).
		Return(message, nil).
		Once()
	messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.ID == message.ID &&
					req.DeliveryStatus == edi.MessageDeliveryStatusSent &&
					req.DeliverySentAt != nil
			}),
		).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateEDIMessageDeliveryRequest) (*edi.EDIMessage, error) {
			message.DeliveryStatus = req.DeliveryStatus
			return message, nil
		}).
		Once()
	tenderChangeRepo := mocks.NewMockEDITenderChangeRepository(t)
	tenderChangeRepo.EXPECT().
		GetTenderChangeByOutboundMessageID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("tender change not found")).
		Once()

	service := New(Params{
		Logger:           zap.NewNop(),
		MessageRepo:      messageRepo,
		TenderChangeRepo: tenderChangeRepo,
	})

	mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
		From:               "PARTNER-AS2",
		To:                 "TRENOVA-AS2",
		OriginalMessageID:  messageID,
		ReceivedContentMIC: "q1w2e3r4, sha256",
	})
	require.NoError(t, err)

	err = service.ApplyAS2MDN(t.Context(), &ApplyAS2MDNRequest{
		ContentType: mdn.ContentType,
		Body:        mdn.Body,
	})
	require.NoError(t, err)
	require.Equal(t, edi.MessageDeliveryStatusSent, message.DeliveryStatus)
}

func TestApplyAS2MDNFailsDeliveryOnRejectedDisposition(t *testing.T) {
	t.Parallel()

	messageID := "<pending-456@trenova.as2>"
	message := pendingAS2Message(messageID)

	messageRepo := mocks.NewMockEDIMessageRepository(t)
	messageRepo.EXPECT().
		GetOutboundMessageByAS2MessageID(mock.Anything, messageID).
		Return(message, nil).
		Once()
	messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.ID == message.ID &&
					req.DeliveryStatus == edi.MessageDeliveryStatusFailed
			}),
		).
		Return(message, nil).
		Once()

	service := New(Params{Logger: zap.NewNop(), MessageRepo: messageRepo})

	mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
		From:              "PARTNER-AS2",
		To:                "TRENOVA-AS2",
		OriginalMessageID: messageID,
		ErrorText:         "unable to decrypt the message",
	})
	require.NoError(t, err)

	err = service.ApplyAS2MDN(t.Context(), &ApplyAS2MDNRequest{
		ContentType: mdn.ContentType,
		Body:        mdn.Body,
	})
	require.NoError(t, err)
}

func TestApplyAS2MDNFailsDeliveryOnMICMismatch(t *testing.T) {
	t.Parallel()

	messageID := "<pending-789@trenova.as2>"
	message := pendingAS2Message(messageID)

	messageRepo := mocks.NewMockEDIMessageRepository(t)
	messageRepo.EXPECT().
		GetOutboundMessageByAS2MessageID(mock.Anything, messageID).
		Return(message, nil).
		Once()
	messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.DeliveryStatus == edi.MessageDeliveryStatusFailed
			}),
		).
		Return(message, nil).
		Once()

	service := New(Params{Logger: zap.NewNop(), MessageRepo: messageRepo})

	mdn, err := as2.BuildMDN(&as2.BuildMDNOptions{
		From:               "PARTNER-AS2",
		To:                 "TRENOVA-AS2",
		OriginalMessageID:  messageID,
		ReceivedContentMIC: "tampered-digest, sha256",
	})
	require.NoError(t, err)

	err = service.ApplyAS2MDN(t.Context(), &ApplyAS2MDNRequest{
		ContentType: mdn.ContentType,
		Body:        mdn.Body,
	})
	require.NoError(t, err)
}
