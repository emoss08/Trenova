package ediservice

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/testutil/mocks/carrierportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type partnerCardRepos struct {
	partners  *ediportsmock.MockEDIPartnerRepository
	documents *ediportsmock.MockEDIPartnerDocumentProfileRepository
	profiles  *ediportsmock.MockEDICommunicationProfileRepository
	mappings  *ediportsmock.MockEDIMappingProfileRepository
	messages  *ediportsmock.MockEDIMessageRepository
	files     *ediportsmock.MockEDIInboundFileRepository
	carriers  *carrierportsmock.MockCarrierRepository
}

func newPartnerCardRepos(t *testing.T) partnerCardRepos {
	t.Helper()
	return partnerCardRepos{
		partners:  ediportsmock.NewMockEDIPartnerRepository(t),
		documents: ediportsmock.NewMockEDIPartnerDocumentProfileRepository(t),
		profiles:  ediportsmock.NewMockEDICommunicationProfileRepository(t),
		mappings:  ediportsmock.NewMockEDIMappingProfileRepository(t),
		messages:  ediportsmock.NewMockEDIMessageRepository(t),
		files:     ediportsmock.NewMockEDIInboundFileRepository(t),
		carriers:  carrierportsmock.NewMockCarrierRepository(t),
	}
}

func (r partnerCardRepos) service() *Service {
	return &Service{
		l:                   zap.NewNop(),
		partnerRepo:         r.partners,
		documentProfileRepo: r.documents,
		profileRepo:         r.profiles,
		mappingProfileRepo:  r.mappings,
		messageRepo:         r.messages,
		inboundFileRepo:     r.files,
		carrierRepo:         r.carriers,
	}
}

func expiringCertificatePEM(t *testing.T, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "edi.tyson.example"},
		Issuer:       pkix.Name{CommonName: "edi.tyson.example"},
		NotBefore:    notAfter.AddDate(-1, 0, 0),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func expectPartnerCardQuiet(r partnerCardRepos, partnerID pulid.ID) {
	r.documents.EXPECT().ListPartnerDocumentProfiles(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{}, nil)
	r.profiles.EXPECT().ListProfilesByPartner(mock.Anything, mock.Anything).
		Return([]*edi.EDICommunicationProfile{}, nil)
	r.partners.EXPECT().GetReadiness(mock.Anything, mock.Anything).
		Return([]*ediports.EDIPartnerReadinessRow{{PartnerID: partnerID, Kind: "External"}}, nil)
	r.mappings.EXPECT().GetMappingSummary(mock.Anything, mock.Anything).
		Return(&ediports.EDIMappingSummary{}, nil)
	r.messages.EXPECT().GetPartnerScorecards(mock.Anything, mock.Anything).Return(nil, nil)
	r.messages.EXPECT().GetAckStatusCounts(mock.Anything, mock.Anything).Return(nil, nil)
	r.messages.EXPECT().GetPartnerSetActivity(mock.Anything, mock.Anything).Return(nil, nil)
	r.messages.EXPECT().GetVolumeSeries(mock.Anything, mock.Anything).Return(nil, nil)
	r.messages.EXPECT().ListOverdueAcks(mock.Anything, mock.Anything).
		Return(&ediports.OverdueEDIAcks{}, nil)
	r.messages.EXPECT().ListRecentDeadLettered(mock.Anything, mock.Anything).Return(nil, nil)
	r.files.EXPECT().ListRecentQuarantined(mock.Anything, mock.Anything).Return(nil, nil)
}

func TestGetPartnerCard_ReadsOnePartnerOverItsWindows(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	partnerID := pulid.MustNew("edip_")
	partner := &edi.EDIPartner{ID: partnerID, Code: "TYSON", Name: "Tyson Foods"}
	overdue := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: partnerID}
	dead := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: partnerID}
	quarantined := &edi.EDIInboundFile{ID: pulid.MustNew("ediinf_")}
	scorecard := &ediports.EDIPartnerScorecardRow{PartnerID: partnerID, SentCount: 1284}
	sets := []*ediports.EDIPartnerSetActivityRow{
		{TransactionSet: edi.TransactionSet("214"), Direction: edi.DocumentDirectionOutbound, Count: 1284},
	}

	r := newPartnerCardRepos(t)
	var partnerReq ediports.GetEDIPartnerByIDRequest
	r.partners.EXPECT().GetByID(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIPartnerByIDRequest) { partnerReq = req }).
		Return(partner, nil)
	var documentsReq *ediports.ListEDIPartnerDocumentProfilesRequest
	r.documents.EXPECT().ListPartnerDocumentProfiles(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.ListEDIPartnerDocumentProfilesRequest) {
			documentsReq = req
		}).
		Return(&pagination.ListResult[*edi.EDIPartnerDocumentProfile]{}, nil)
	r.profiles.EXPECT().ListProfilesByPartner(mock.Anything, mock.Anything).
		Return([]*edi.EDICommunicationProfile{}, nil)
	r.partners.EXPECT().GetReadiness(mock.Anything, mock.Anything).
		Return([]*ediports.EDIPartnerReadinessRow{{PartnerID: partnerID, Kind: "External"}}, nil)
	var mappingReq ediports.GetEDIMappingSummaryRequest
	r.mappings.EXPECT().GetMappingSummary(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIMappingSummaryRequest) { mappingReq = req }).
		Return(&ediports.EDIMappingSummary{}, nil)
	var scorecardReq *ediports.GetEDIPartnerScorecardsRequest
	r.messages.EXPECT().GetPartnerScorecards(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.GetEDIPartnerScorecardsRequest) { scorecardReq = req }).
		Return([]*ediports.EDIPartnerScorecardRow{scorecard}, nil)
	var countsReq ediports.GetEDIMessageStatusCountsRequest
	r.messages.EXPECT().GetAckStatusCounts(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIMessageStatusCountsRequest) { countsReq = req }).
		Return(map[edi.MessageAcknowledgmentStatus]int{edi.MessageAcknowledgmentStatusAccepted: 1399}, nil)
	var setsReq *ediports.GetEDIPartnerSetActivityRequest
	r.messages.EXPECT().GetPartnerSetActivity(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.GetEDIPartnerSetActivityRequest) { setsReq = req }).
		Return(sets, nil)
	var seriesReq ediports.GetEDIVolumeSeriesRequest
	r.messages.EXPECT().GetVolumeSeries(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIVolumeSeriesRequest) { seriesReq = req }).
		Return([]*ediports.EDIVolumePoint{{BucketStart: 3600, OutboundCount: 4}}, nil)
	var overdueReq *ediports.ListOverdueEDIAcksRequest
	r.messages.EXPECT().ListOverdueAcks(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.ListOverdueEDIAcksRequest) { overdueReq = req }).
		Return(&ediports.OverdueEDIAcks{
			Messages: []*edi.EDIMessage{overdue},
			Partners: []*ediports.OverdueEDIAckPartner{
				{PartnerID: partnerID, Count: 3, OldestGeneratedAt: 1_700_000_000},
			},
		}, nil)
	var deadReq *ediports.ListRecentEDIMessageFailuresRequest
	r.messages.EXPECT().ListRecentDeadLettered(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.ListRecentEDIMessageFailuresRequest) { deadReq = req }).
		Return([]*edi.EDIMessage{dead}, nil)
	var filesReq ediports.ListRecentQuarantinedEDIInboundFilesRequest
	r.files.EXPECT().ListRecentQuarantined(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.ListRecentQuarantinedEDIInboundFilesRequest) {
			filesReq = req
		}).
		Return([]*edi.EDIInboundFile{quarantined}, nil)

	before := timeutils.NowUnix()
	card, err := r.service().GetPartnerCard(t.Context(), &GetEDIPartnerCardRequest{
		TenantInfo: tenant,
		PartnerID:  partnerID,
	})
	require.NoError(t, err)
	after := timeutils.NowUnix()

	assert.Equal(t, ediports.GetEDIPartnerByIDRequest{
		ID: partnerID, TenantInfo: tenant, IncludeLinks: true,
	}, partnerReq, "the customer and connection are read with the partner")
	assert.Equal(t, partnerID, documentsReq.PartnerID)
	assert.Equal(t, tenant, documentsReq.Filter.TenantInfo)
	assert.Equal(t, partnerID, mappingReq.PartnerID)

	window := card.Activity.Since
	assert.GreaterOrEqual(t, window, before-PartnerCardActivityWindowSeconds)
	assert.LessOrEqual(t, window, after-PartnerCardActivityWindowSeconds)
	assert.Equal(t, partnerID, scorecardReq.PartnerID, "the scorecard is the partner's alone")
	assert.Equal(t, window, scorecardReq.Since)
	assert.Equal(t, ediports.GetEDIMessageStatusCountsRequest{
		TenantInfo: tenant, Since: window, PartnerID: partnerID,
	}, countsReq)
	assert.Equal(t, &ediports.GetEDIPartnerSetActivityRequest{
		TenantInfo: tenant, PartnerID: partnerID, Since: window,
	}, setsReq)
	assert.Equal(t, ediports.GetEDIVolumeSeriesRequest{
		TenantInfo:    tenant,
		Since:         window + PartnerCardActivityWindowSeconds - PartnerCardSeriesWindowSeconds,
		BucketSeconds: PartnerCardSeriesBucketSeconds,
		PartnerID:     partnerID,
	}, seriesReq, "the series runs fourteen days by the hour, for this partner")

	assert.Equal(t, partnerID, overdueReq.PartnerID)
	assert.Equal(t, PartnerCardAttentionLimit, overdueReq.Limit)
	assert.Equal(t, window+PartnerCardActivityWindowSeconds-DefaultOverdueAckAgeSeconds,
		overdueReq.PendingSince, "overdue is the EDI overview's four hours")
	assert.Equal(t, &ediports.ListRecentEDIMessageFailuresRequest{
		TenantInfo: tenant, PartnerID: partnerID, Since: window, Limit: PartnerCardAttentionLimit,
	}, deadReq, "only dead letters inside the window are drawn as needing a look")
	assert.Equal(t, ediports.ListRecentQuarantinedEDIInboundFilesRequest{
		TenantInfo: tenant, PartnerID: partnerID, Since: window, Limit: PartnerCardAttentionLimit,
	}, filesReq)

	assert.Same(t, partner, card.Partner)
	assert.Same(t, scorecard, card.Activity.Scorecard)
	assert.Equal(t, sets, card.Activity.Sets)
	assert.Equal(t, 1399, card.Activity.AckCounts[edi.MessageAcknowledgmentStatusAccepted])
	assert.Equal(t, 3, card.Attention.OverdueAckCount)
	assert.Equal(t, 3, card.Activity.OverdueAckCount, "the figures and the attention agree")
	assert.Equal(t, int64(1_700_000_000), card.Attention.OldestOverdueAt)
	assert.Equal(t, []*edi.EDIMessage{overdue}, card.Attention.OverdueAcks)
	assert.Equal(t, []*edi.EDIMessage{dead}, card.Attention.DeadLettered)
	assert.Equal(t, []*edi.EDIInboundFile{quarantined}, card.Attention.Quarantined)
	assert.Nil(t, card.Carriers, "carriers are not read unless asked for")
}

func TestGetPartnerCard_DescribesTransportsWithoutTheirSecrets(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	notAfter := time.Now().Add(9 * 24 * time.Hour).Truncate(time.Second)
	as2Profile := &edi.EDICommunicationProfile{
		ID:     pulid.MustNew("edicp_"),
		Method: edi.ConnectionMethodAS2,
		Config: map[string]any{
			editransport.ConfigKeyPartnerEncryptionCertificate: expiringCertificatePEM(t, notAfter),
			editransport.ConfigKeyPartnerSigningCertificate:    "-----BEGIN CERTIFICATE-----\nnot a cert\n-----END CERTIFICATE-----",
		},
		EncryptedSecrets: map[string]string{editransport.SecretKeyBasicAuthPassword: "ciphertext"},
	}
	sftpProfile := &edi.EDICommunicationProfile{
		ID:            pulid.MustNew("edicp_"),
		Method:        edi.ConnectionMethodSFTP,
		Config:        map[string]any{"knownHostKey": "ssh-ed25519 AAAA"},
		LastPollError: "ssh: handshake failed; password: hunter2 was refused",
	}

	r := newPartnerCardRepos(t)
	r.partners.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&edi.EDIPartner{ID: partnerID}, nil)
	expectPartnerCardQuiet(r, partnerID)
	r.profiles.ExpectedCalls = nil
	r.profiles.EXPECT().ListProfilesByPartner(mock.Anything, ediports.ListEDICommunicationProfilesByPartnerRequest{
		PartnerID: partnerID,
	}).Return([]*edi.EDICommunicationProfile{as2Profile, sftpProfile}, nil)

	card, err := r.service().GetPartnerCard(t.Context(), &GetEDIPartnerCardRequest{PartnerID: partnerID})
	require.NoError(t, err)
	require.Len(t, card.Transports, 2)

	as2Transport := card.Transports[0]
	assert.Nil(t, as2Transport.Profile.EncryptedSecrets, "no secret leaves the service")
	assert.Equal(t, []edi.CommunicationProfileSecretState{
		{Key: editransport.SecretKeyBasicAuthPassword, HasValue: true},
	}, as2Transport.Profile.SecretState)
	require.Len(t, as2Transport.Certificates, 2)
	encryption := as2Transport.Certificates[0]
	assert.Equal(t, PartnerCertificateRolePartnerEncryption, encryption.Role)
	require.NotNil(t, encryption.Summary)
	assert.Equal(t, "CN=edi.tyson.example", encryption.Summary.Subject)
	assert.Equal(t, notAfter.Unix(), encryption.Summary.NotAfter)
	assert.False(t, encryption.Summary.Expired)
	assert.Equal(t, EDIPartnerCertificate{
		Role:       PartnerCertificateRolePartnerSigning,
		Unreadable: true,
	}, as2Transport.Certificates[1], "a PEM that does not parse is flagged, not dropped")

	var basicAuth *ediports.CredentialRequirement
	for index := range as2Transport.Credentials {
		if as2Transport.Credentials[index].Keys[0] == editransport.SecretKeyBasicAuthPassword {
			basicAuth = &as2Transport.Credentials[index]
		}
	}
	require.NotNil(t, basicAuth)
	assert.False(t, basicAuth.Missing, "the password is on file")

	sftpTransport := card.Transports[1]
	assert.Empty(t, sftpTransport.Certificates, "only AS2 carries certificates")
	assert.Equal(t, "ssh: handshake failed; password: [redacted] was refused",
		sftpTransport.LastPollError)
	require.Len(t, sftpTransport.Credentials, 2)
	assert.False(t, sftpTransport.Credentials[0].Missing, "the host key is pinned")
	assert.True(t, sftpTransport.Credentials[1].Missing, "no password or private key yet")
	assert.True(t, sftpTransport.Credentials[1].Required)
}

func TestGetPartnerCard_ReadsCarriersOnlyWhenAskedFor(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	partnerID := pulid.MustNew("edip_")
	channel := &carrier.CarrierEDIChannel{
		ID:           pulid.MustNew("cec_"),
		CarrierID:    pulid.MustNew("car_"),
		EDIPartnerID: partnerID,
		Carrier:      &carrier.Carrier{Name: "Red River Freight"},
	}

	r := newPartnerCardRepos(t)
	r.partners.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&edi.EDIPartner{ID: partnerID}, nil)
	expectPartnerCardQuiet(r, partnerID)
	r.carriers.EXPECT().ListEDIChannelsByPartner(mock.Anything, carrierports.ListCarrierEDIChannelsByPartnerRequest{
		TenantInfo:   tenant,
		EDIPartnerID: partnerID,
	}).Return([]*carrier.CarrierEDIChannel{channel}, nil)

	card, err := r.service().GetPartnerCard(t.Context(), &GetEDIPartnerCardRequest{
		TenantInfo:      tenant,
		PartnerID:       partnerID,
		IncludeCarriers: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []*carrier.CarrierEDIChannel{channel}, card.Carriers)
	assert.NotNil(t, card.Activity.Scorecard, "a partner with no traffic still has a scorecard")
	assert.Equal(t, partnerID, card.Activity.Scorecard.PartnerID)
	assert.Equal(t, partnerID, card.Readiness.PartnerID)
}
