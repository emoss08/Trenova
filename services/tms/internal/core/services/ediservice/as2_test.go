package ediservice

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const as2TestMIC = "q1w2e3r4, sha256"

type mdnSigner struct {
	certificate    *x509.Certificate
	key            *rsa.PrivateKey
	certificatePEM string
}

func newMDNSigner(t *testing.T, commonName string) *mdnSigner {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &mdnSigner{
		certificate:    certificate,
		key:            key,
		certificatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
	}
}

func pendingAS2Message(messageID string) *edi.EDIMessage {
	return &edi.EDIMessage{
		ID:             pulid.MustNew("edimsg_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Direction:      edi.DocumentDirectionOutbound,
		DeliveryStatus: edi.MessageDeliveryStatusSending,
		AS2MessageID:   messageID,
		AS2MIC:         as2TestMIC,
	}
}

func as2DeliveryProfile(
	message *edi.EDIMessage,
	partner *mdnSigner,
) *edi.EDICommunicationProfile {
	config := map[string]any{
		editransport.ConfigKeyLocalAS2ID: "TRENOVA-AS2",
		"partnerAS2Id":                   "PARTNER-AS2",
		"endpointUrl":                    "https://partner.example.com/as2",
		"mdnMode":                        "async",
	}
	if partner != nil {
		config[editransport.ConfigKeyPartnerSigningCertificate] = partner.certificatePEM
	}

	return &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("ecp_"),
		BusinessUnitID: message.BusinessUnitID,
		OrganizationID: message.OrganizationID,
		EDIPartnerID:   message.EDIPartnerID,
		Method:         edi.ConnectionMethodAS2,
		Status:         domaintypes.StatusActive,
		Config:         config,
	}
}

type mdnTestHarness struct {
	service     *Service
	messageRepo *mocks.MockEDIMessageRepository
}

func newMDNTestHarness(
	t *testing.T,
	message *edi.EDIMessage,
	partner *mdnSigner,
) *mdnTestHarness {
	t.Helper()

	messageRepo := mocks.NewMockEDIMessageRepository(t)
	messageRepo.EXPECT().
		GetOutboundMessageByAS2MessageID(mock.Anything, message.AS2MessageID).
		Return(message, nil).
		Once()

	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveProfileByPartner(
			mock.Anything,
			mock.MatchedBy(func(req repositories.GetActiveEDICommunicationProfileByPartnerRequest) bool {
				return req.PartnerID == message.EDIPartnerID &&
					req.TenantInfo.OrgID == message.OrganizationID
			}),
		).
		Return(as2DeliveryProfile(message, partner), nil).
		Once()

	tenderChangeRepo := mocks.NewMockEDITenderChangeRepository(t)
	tenderChangeRepo.EXPECT().
		GetTenderChangeByOutboundMessageID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("tender change not found")).
		Maybe()

	return &mdnTestHarness{
		service: New(Params{
			Logger:           zap.NewNop(),
			MessageRepo:      messageRepo,
			ProfileRepo:      profileRepo,
			TenderChangeRepo: tenderChangeRepo,
		}),
		messageRepo: messageRepo,
	}
}

func (h *mdnTestHarness) expectDeliveryStatus(
	message *edi.EDIMessage,
	status edi.MessageDeliveryStatus,
) {
	h.messageRepo.EXPECT().
		UpdateMessageDelivery(
			mock.Anything,
			mock.MatchedBy(func(req *repositories.UpdateEDIMessageDeliveryRequest) bool {
				return req.ID == message.ID && req.DeliveryStatus == status
			}),
		).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateEDIMessageDeliveryRequest) (*edi.EDIMessage, error) {
			message.DeliveryStatus = req.DeliveryStatus
			return message, nil
		}).
		Once()
}

func buildMDN(t *testing.T, opts *as2.BuildMDNOptions) *ApplyAS2MDNRequest {
	t.Helper()

	opts.From = "PARTNER-AS2"
	opts.To = "TRENOVA-AS2"
	mdn, err := as2.BuildMDN(opts)
	require.NoError(t, err)

	return &ApplyAS2MDNRequest{ContentType: mdn.ContentType, Body: mdn.Body}
}

func TestApplyAS2MDNResolvesPendingDeliveryWhenMICMatches(t *testing.T) {
	t.Parallel()

	message := pendingAS2Message("<pending-123@trenova.as2>")
	h := newMDNTestHarness(t, message, nil)
	h.expectDeliveryStatus(message, edi.MessageDeliveryStatusSent)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ReceivedContentMIC: as2TestMIC,
	}))

	require.NoError(t, err)
	assert.Equal(t, edi.MessageDeliveryStatusSent, message.DeliveryStatus)
}

func TestApplyAS2MDNRejectsUnauthenticatedFailureReceipt(t *testing.T) {
	t.Parallel()

	message := pendingAS2Message("<pending-456@trenova.as2>")
	h := newMDNTestHarness(t, message, nil)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID: message.AS2MessageID,
		ErrorText:         "unable to decrypt the message",
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Equal(t, edi.MessageDeliveryStatusSending, message.DeliveryStatus)
}

func TestApplyAS2MDNRejectsReceiptWithAnotherMessagesMIC(t *testing.T) {
	t.Parallel()

	message := pendingAS2Message("<pending-789@trenova.as2>")
	h := newMDNTestHarness(t, message, nil)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ReceivedContentMIC: "tampered-digest, sha256",
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Equal(t, edi.MessageDeliveryStatusSending, message.DeliveryStatus)
}

func TestApplyAS2MDNRequiresSignatureWhenPartnerCertificateIsConfigured(t *testing.T) {
	t.Parallel()

	partner := newMDNSigner(t, "partner")
	message := pendingAS2Message("<pending-signed-1@trenova.as2>")
	h := newMDNTestHarness(t, message, partner)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ReceivedContentMIC: as2TestMIC,
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Equal(t, edi.MessageDeliveryStatusSending, message.DeliveryStatus)
}

func TestApplyAS2MDNRejectsReceiptSignedByAnotherKey(t *testing.T) {
	t.Parallel()

	partner := newMDNSigner(t, "partner")
	impostor := newMDNSigner(t, "impostor")
	message := pendingAS2Message("<pending-signed-2@trenova.as2>")
	h := newMDNTestHarness(t, message, partner)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ReceivedContentMIC: as2TestMIC,
		SigningCertificate: impostor.certificate,
		SigningKey:         impostor.key,
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Equal(t, edi.MessageDeliveryStatusSending, message.DeliveryStatus)
}

func TestApplyAS2MDNFailsDeliveryOnSignedRejectedDisposition(t *testing.T) {
	t.Parallel()

	partner := newMDNSigner(t, "partner")
	message := pendingAS2Message("<pending-signed-3@trenova.as2>")
	h := newMDNTestHarness(t, message, partner)
	h.expectDeliveryStatus(message, edi.MessageDeliveryStatusFailed)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ErrorText:          "unable to decrypt the message",
		SigningCertificate: partner.certificate,
		SigningKey:         partner.key,
	}))

	require.NoError(t, err)
	assert.Equal(t, edi.MessageDeliveryStatusFailed, message.DeliveryStatus)
}

func TestApplyAS2MDNFailsDeliveryOnSignedMICMismatch(t *testing.T) {
	t.Parallel()

	partner := newMDNSigner(t, "partner")
	message := pendingAS2Message("<pending-signed-4@trenova.as2>")
	h := newMDNTestHarness(t, message, partner)
	h.expectDeliveryStatus(message, edi.MessageDeliveryStatusFailed)

	err := h.service.ApplyAS2MDN(t.Context(), buildMDN(t, &as2.BuildMDNOptions{
		OriginalMessageID:  message.AS2MessageID,
		ReceivedContentMIC: "tampered-digest, sha256",
		SigningCertificate: partner.certificate,
		SigningKey:         partner.key,
	}))

	require.NoError(t, err)
	assert.Equal(t, edi.MessageDeliveryStatusFailed, message.DeliveryStatus)
}
