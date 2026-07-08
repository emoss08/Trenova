package ediinboundservice

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
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type as2ReceiverIdentity struct {
	certificate    *x509.Certificate
	key            *rsa.PrivateKey
	certificatePEM string
	keyPEM         string
}

func newAS2ReceiverIdentity(t *testing.T, commonName string) *as2ReceiverIdentity {
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
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return &as2ReceiverIdentity{
		certificate: certificate,
		key:         key,
		certificatePEM: string(
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		),
		keyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}
}

const as2ReceiverPayload = "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
	"*260107*1200*^*00401*000000002*0*P*>~GS*SM*PARTNER*TRENOVA*20260107*1200*2*X*004010~" +
	"ST*204*0001~B2**SCAC**SHIP2**PP~SE*3*0001~GE*1*2~IEA*1*000000002~"

func TestReceiveAS2MessageStagesFileAndReturnsMDN(t *testing.T) {
	t.Parallel()

	trenova := newAS2ReceiverIdentity(t, "trenova.example")
	partner := newAS2ReceiverIdentity(t, "partner.example")

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	profile := &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("edicp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		EDIPartnerID:   partnerID,
		Method:         edi.ConnectionMethodAS2,
		Config: map[string]any{
			editransport.ConfigKeyLocalAS2ID:                "TRENOVA-AS2",
			editransport.ConfigKeyPartnerAS2ID:              "PARTNER-AS2",
			editransport.ConfigKeyEndpointURL:               "https://partner.example/as2",
			editransport.ConfigKeyMDNMode:                   editransport.MDNModeSync,
			editransport.ConfigKeyLocalCertificate:          trenova.certificatePEM,
			editransport.ConfigKeyPartnerSigningCertificate: partner.certificatePEM,
		},
		EncryptedSecrets: map[string]string{
			editransport.SecretKeyAS2PrivateKey: trenova.keyPEM,
		},
	}

	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileByIdentifiers(
			mock.Anything,
			repositories.GetActiveAS2ProfileByIdentifiersRequest{
				LocalAS2ID:   "TRENOVA-AS2",
				PartnerAS2ID: "PARTNER-AS2",
			},
		).
		Return(profile, nil)
	inboundFileRepo := mocks.NewMockEDIInboundFileRepository(t)
	inboundFileRepo.EXPECT().
		ExistsByChecksum(mock.Anything, mock.Anything).
		Return(false, nil).
		Once()
	inboundFileRepo.EXPECT().
		CreateInboundFile(mock.Anything, mock.MatchedBy(func(file *edi.EDIInboundFile) bool {
			return file.Method == edi.ConnectionMethodAS2 &&
				file.RawContent == as2ReceiverPayload &&
				file.EDIPartnerID == partnerID
		})).
		RunAndReturn(func(_ context.Context, file *edi.EDIInboundFile) (*edi.EDIInboundFile, error) {
			file.ID = pulid.MustNew("ediinf_")
			return file, nil
		}).
		Once()
	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	ediSvc := ediservice.New(ediservice.Params{Logger: zap.NewNop()})
	service := New(Params{
		Logger:          zap.NewNop(),
		InboundFileRepo: inboundFileRepo,
		ProfileRepo:     profileRepo,
		EDIService:      ediSvc,
		WorkflowStarter: workflowStarter,
	})

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:                  "PARTNER-AS2",
		To:                    "TRENOVA-AS2",
		Payload:               []byte(as2ReceiverPayload),
		SigningCertificate:    partner.certificate,
		SigningKey:            partner.key,
		EncryptionCertificate: trenova.certificate,
		RequestMDN:            true,
		RequestSignedMDN:      true,
	})
	require.NoError(t, err)

	request := &ReceiveAS2MessageRequest{
		From:             "PARTNER-AS2",
		To:               "TRENOVA-AS2",
		MessageID:        built.MessageID,
		ContentType:      built.ContentType,
		TransferEncoding: built.Headers.Get("Content-Transfer-Encoding"),
		DispositionNotificationTo: built.Headers.Get(
			as2.HeaderDispositionNotificationTo,
		),
		DispositionNotificationOptions: built.Headers.Get(
			as2.HeaderDispositionNotificationOptions,
		),
		Body: built.Body,
	}
	result, err := service.ReceiveAS2Message(t.Context(), request)
	require.NoError(t, err)
	require.False(t, result.Duplicate)
	require.False(t, result.Rejected)
	require.False(t, result.AsyncMDN)
	require.NotEmpty(t, result.MDNBody)

	mdn, err := as2.ParseMDN(result.MDNContentType, result.MDNBody, trenova.certificate)
	require.NoError(t, err)
	require.True(t, mdn.Signed)
	require.True(t, mdn.Processed())
	require.Equal(t, built.MessageID, mdn.OriginalMessageID)
	require.True(t, as2.MICMatches(built.MIC, mdn.ReceivedContentMIC))

	inboundFileRepo.EXPECT().
		ExistsByChecksum(mock.Anything, mock.Anything).
		Return(true, nil).
		Once()
	duplicate, err := service.ReceiveAS2Message(t.Context(), request)
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)
}

func TestReceiveAS2MessageRejectsUnsignedWhenPartnerCertConfigured(t *testing.T) {
	t.Parallel()

	trenova := newAS2ReceiverIdentity(t, "trenova.example")
	partner := newAS2ReceiverIdentity(t, "partner.example")

	profile := &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("edicp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Method:         edi.ConnectionMethodAS2,
		Config: map[string]any{
			editransport.ConfigKeyLocalAS2ID:                "TRENOVA-AS2",
			editransport.ConfigKeyPartnerAS2ID:              "PARTNER-AS2",
			editransport.ConfigKeyLocalCertificate:          trenova.certificatePEM,
			editransport.ConfigKeyPartnerSigningCertificate: partner.certificatePEM,
		},
		EncryptedSecrets: map[string]string{
			editransport.SecretKeyAS2PrivateKey: trenova.keyPEM,
		},
	}
	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileByIdentifiers(mock.Anything, mock.Anything).
		Return(profile, nil).
		Once()

	service := New(Params{
		Logger:      zap.NewNop(),
		ProfileRepo: profileRepo,
		EDIService:  ediservice.New(ediservice.Params{Logger: zap.NewNop()}),
	})

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:    "PARTNER-AS2",
		To:      "TRENOVA-AS2",
		Payload: []byte(as2ReceiverPayload),
	})
	require.NoError(t, err)

	result, err := service.ReceiveAS2Message(t.Context(), &ReceiveAS2MessageRequest{
		From:             "PARTNER-AS2",
		To:               "TRENOVA-AS2",
		MessageID:        built.MessageID,
		ContentType:      built.ContentType,
		TransferEncoding: built.Headers.Get("Content-Transfer-Encoding"),
		Body:             built.Body,
	})
	require.NoError(t, err)
	require.True(t, result.Rejected)
	require.NotEmpty(t, result.MDNBody)

	mdn, err := as2.ParseMDN(result.MDNContentType, result.MDNBody, nil)
	require.NoError(t, err)
	require.False(t, mdn.Processed())
}

func TestReceiveAS2MessageAcceptsUnsignedWhenRequirementsDisabled(t *testing.T) {
	t.Parallel()

	trenova := newAS2ReceiverIdentity(t, "trenova.example")
	partner := newAS2ReceiverIdentity(t, "partner.example")

	partnerID := pulid.MustNew("edip_")
	profile := &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("edicp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   partnerID,
		Method:         edi.ConnectionMethodAS2,
		Config: map[string]any{
			editransport.ConfigKeyLocalAS2ID:                "TRENOVA-AS2",
			editransport.ConfigKeyPartnerAS2ID:              "PARTNER-AS2",
			editransport.ConfigKeyLocalCertificate:          trenova.certificatePEM,
			editransport.ConfigKeyPartnerSigningCertificate: partner.certificatePEM,
			editransport.ConfigKeyRequireSignedInbound:      "false",
			editransport.ConfigKeyRequireEncryptedInbound:   "false",
		},
		EncryptedSecrets: map[string]string{
			editransport.SecretKeyAS2PrivateKey: trenova.keyPEM,
		},
	}
	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileByIdentifiers(mock.Anything, mock.Anything).
		Return(profile, nil).
		Once()
	inboundFileRepo := mocks.NewMockEDIInboundFileRepository(t)
	inboundFileRepo.EXPECT().
		ExistsByChecksum(mock.Anything, mock.Anything).
		Return(false, nil).
		Once()
	inboundFileRepo.EXPECT().
		CreateInboundFile(mock.Anything, mock.MatchedBy(func(file *edi.EDIInboundFile) bool {
			return file.RawContent == as2ReceiverPayload && file.EDIPartnerID == partnerID
		})).
		RunAndReturn(func(_ context.Context, file *edi.EDIInboundFile) (*edi.EDIInboundFile, error) {
			file.ID = pulid.MustNew("ediinf_")
			return file, nil
		}).
		Once()
	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true)
	workflowStarter.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	service := New(Params{
		Logger:          zap.NewNop(),
		InboundFileRepo: inboundFileRepo,
		ProfileRepo:     profileRepo,
		EDIService:      ediservice.New(ediservice.Params{Logger: zap.NewNop()}),
		WorkflowStarter: workflowStarter,
	})

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:    "PARTNER-AS2",
		To:      "TRENOVA-AS2",
		Payload: []byte(as2ReceiverPayload),
	})
	require.NoError(t, err)

	result, err := service.ReceiveAS2Message(t.Context(), &ReceiveAS2MessageRequest{
		From:             "PARTNER-AS2",
		To:               "TRENOVA-AS2",
		MessageID:        built.MessageID,
		ContentType:      built.ContentType,
		TransferEncoding: built.Headers.Get("Content-Transfer-Encoding"),
		Body:             built.Body,
	})
	require.NoError(t, err)
	require.False(t, result.Rejected)
	require.False(t, result.Duplicate)
	require.NotEmpty(t, result.MDNBody)
}

func TestReceiveAS2MessageRejectsUnknownIdentifiers(t *testing.T) {
	t.Parallel()

	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileByIdentifiers(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDICommunicationProfile not found")).
		Once()

	service := New(Params{
		Logger:      zap.NewNop(),
		ProfileRepo: profileRepo,
		EDIService:  ediservice.New(ediservice.Params{Logger: zap.NewNop()}),
	})

	_, err := service.ReceiveAS2Message(t.Context(), &ReceiveAS2MessageRequest{
		From:        "UNKNOWN",
		To:          "TRENOVA-AS2",
		ContentType: "application/edi-x12",
		Body:        []byte(as2ReceiverPayload),
	})
	require.ErrorIs(t, err, ErrAS2ProfileNotFound)
}

func TestReceiveAS2MessageReturnsFailureMDNOnBadPayload(t *testing.T) {
	t.Parallel()

	trenova := newAS2ReceiverIdentity(t, "trenova.example")
	partner := newAS2ReceiverIdentity(t, "partner.example")
	impostor := newAS2ReceiverIdentity(t, "impostor.example")

	profile := &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("edicp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EDIPartnerID:   pulid.MustNew("edip_"),
		Method:         edi.ConnectionMethodAS2,
		Config: map[string]any{
			editransport.ConfigKeyLocalAS2ID:                "TRENOVA-AS2",
			editransport.ConfigKeyPartnerAS2ID:              "PARTNER-AS2",
			editransport.ConfigKeyPartnerSigningCertificate: partner.certificatePEM,
			editransport.ConfigKeyLocalCertificate:          trenova.certificatePEM,
		},
		EncryptedSecrets: map[string]string{
			editransport.SecretKeyAS2PrivateKey: trenova.keyPEM,
		},
	}
	profileRepo := mocks.NewMockEDICommunicationProfileRepository(t)
	profileRepo.EXPECT().
		GetActiveAS2ProfileByIdentifiers(mock.Anything, mock.Anything).
		Return(profile, nil).
		Once()

	service := New(Params{
		Logger:      zap.NewNop(),
		ProfileRepo: profileRepo,
		EDIService:  ediservice.New(ediservice.Params{Logger: zap.NewNop()}),
	})

	built, err := as2.BuildMessage(&as2.BuildMessageOptions{
		From:                  "PARTNER-AS2",
		To:                    "TRENOVA-AS2",
		Payload:               []byte(as2ReceiverPayload),
		SigningCertificate:    impostor.certificate,
		SigningKey:            impostor.key,
		EncryptionCertificate: trenova.certificate,
	})
	require.NoError(t, err)

	result, err := service.ReceiveAS2Message(t.Context(), &ReceiveAS2MessageRequest{
		From:             "PARTNER-AS2",
		To:               "TRENOVA-AS2",
		MessageID:        built.MessageID,
		ContentType:      built.ContentType,
		TransferEncoding: built.Headers.Get("Content-Transfer-Encoding"),
		Body:             built.Body,
	})
	require.NoError(t, err)
	require.True(t, result.Rejected)
	require.NotEmpty(t, result.MDNBody)

	mdn, err := as2.ParseMDN(result.MDNContentType, result.MDNBody, nil)
	require.NoError(t, err)
	require.False(t, mdn.Processed())
}
