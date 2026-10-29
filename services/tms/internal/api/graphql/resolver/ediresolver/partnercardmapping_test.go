package ediresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/smime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func partnerCardFixture() *ediservice.EDIPartnerCard {
	partnerID := pulid.MustNew("edip_")
	return &ediservice.EDIPartnerCard{
		Partner: &edi.EDIPartner{ID: partnerID, Code: "TYSON", Name: "Tyson Foods"},
		Readiness: &ediservice.EDIPartnerReadiness{
			PartnerID:      partnerID,
			CompletedCount: 1,
			TotalCount:     2,
			Items: []ediservice.EDIPartnerReadinessItem{
				{Key: "details", Label: "Partner details", Complete: true},
				{Key: "lading-active", Label: "An active Lading version"},
			},
		},
		Mappings: &edipartnerports.EDIMappingSummary{},
		Activity: &ediservice.EDIPartnerActivity{
			Since:         100,
			SeriesSince:   50,
			BucketSeconds: 3600,
			Scorecard: &edimessageports.EDIPartnerScorecardRow{
				PartnerID:   partnerID,
				SentCount:   9,
				FailedCount: 1,
			},
			OverdueAckCount: 3,
		},
		Attention: &ediservice.EDIPartnerAttention{OverdueAckAge: 14400},
	}
}

func TestEDIPartnerCardToModel_NamesCredentialsWithoutValues(t *testing.T) {
	t.Parallel()

	card := partnerCardFixture()
	pollAttempt, pollSuccess := int64(1_700_000_100), int64(1_700_000_000)
	card.Transports = []*ediservice.EDIPartnerTransport{{
		Profile: &edi.EDICommunicationProfile{
			Method:            edi.ConnectionMethodAS2,
			LastPollAttemptAt: &pollAttempt,
			LastPollSuccessAt: &pollSuccess,
		},
		LastPollError: "password: [redacted] refused",
		Credentials: []ediconnectiondraftports.CredentialRequirement{
			{
				Keys:     []string{"password", "privateKey"},
				Kind:     ediconnectiondraftports.CredentialKindSecret,
				Required: true,
				Missing:  true,
			},
			{
				Keys:     []string{"knownHostKey"},
				Kind:     ediconnectiondraftports.CredentialKindSetting,
				Required: true,
			},
			{Keys: nil, Kind: ediconnectiondraftports.CredentialKindSecret},
		},
		Certificates: []ediservice.EDIPartnerCertificate{
			{
				Role: ediservice.PartnerCertificateRolePartnerEncryption,
				Summary: &smime.CertificateSummary{
					Subject:   "CN=edi.tyson.example",
					Issuer:    "CN=Sectigo RSA",
					NotBefore: 1_690_000_000,
					NotAfter:  1_800_000_000,
				},
			},
			{Role: ediservice.PartnerCertificateRolePartnerSigning, Unreadable: true},
		},
	}}

	model := ediPartnerCardToModel(card)

	require.Len(t, model.Transports, 1)
	transport := model.Transports[0]
	assert.Equal(t, new(1_700_000_100), transport.LastPollAttemptAt)
	assert.Equal(t, new(1_700_000_000), transport.LastPollSuccessAt)
	assert.Equal(t, new("password: [redacted] refused"), transport.LastPollError)
	assert.Equal(t, []*gqlmodel.EDITransportCredential{
		{Key: "password", Secret: true, Required: true, OnFile: false},
		{Key: "knownHostKey", Secret: false, Required: true, OnFile: true},
	}, transport.Credentials, "a credential with no key is not listed")
	assert.Equal(t, []*gqlmodel.EDITransportCertificate{
		{
			Role:      gqlmodel.EDITransportCertificateRolePartnerEncryption,
			Readable:  true,
			Subject:   new("CN=edi.tyson.example"),
			Issuer:    new("CN=Sectigo RSA"),
			NotBefore: new(1_690_000_000),
			NotAfter:  new(1_800_000_000),
		},
		{Role: gqlmodel.EDITransportCertificateRolePartnerSigning, Readable: false},
	}, transport.Certificates)
}

func TestEDIPartnerCardToModel_KeepsCarriersNullWhenNotRead(t *testing.T) {
	t.Parallel()

	model := ediPartnerCardToModel(partnerCardFixture())
	assert.Nil(t, model.Carriers, "null tells the card the carriers were not read")
	assert.Nil(t, model.Customer, "a partner that names no customer has none")
	require.NotNil(t, model.Lading, "a partner with no project has an empty Lading block")
	assert.Nil(t, model.Lading.Project)
	assert.Nil(t, model.Lading.Active)
	assert.Equal(t, []*edimessage.EDIMessage{}, model.Attention.OverdueAcks)
	assert.Equal(t, []*ediinboundfile.EDIInboundFile{}, model.Attention.QuarantinedFiles)

	read := partnerCardFixture()
	read.Carriers = []*carrier.CarrierEDIChannel{}
	assert.Equal(t, []*gqlmodel.EDIPartnerCarrier{}, ediPartnerCardToModel(read).Carriers,
		"read and empty is a list, not null")
}

func TestEDIPartnerCardToModel_ReadsTheLinksAndFigures(t *testing.T) {
	t.Parallel()

	card := partnerCardFixture()
	customerID := pulid.MustNew("cus_")
	card.Partner.CustomerID = customerID
	card.Partner.Customer = &customer.Customer{ID: customerID, Name: "Tyson Foods", Code: "TYSON"}
	card.Carriers = []*carrier.CarrierEDIChannel{
		{
			CarrierID:    pulid.MustNew("car_"),
			IsDefault:    true,
			SCACOverride: "RRFX",
			Carrier:      &carrier.Carrier{Name: "Red River Freight", Code: "RRF", SCAC: "RRFT"},
		},
		{CarrierID: pulid.MustNew("car_")},
	}
	card.Mappings = &edipartnerports.EDIMappingSummary{
		ProfileID: pulid.MustNew("edimp_"),
		UpdatedAt: 1_700_000_000,
		Counts: []*edipartnerports.EDIMappingEntityCount{
			{EntityType: edi.MappingEntityTypeLocation, Count: 41},
		},
	}
	card.Activity.Sets = []*edimessageports.EDIPartnerSetActivityRow{
		{
			TransactionSet: "214",
			Direction:      edidocument.DocumentDirectionOutbound,
			Count:          1284,
			FailedCount:    2,
			LastAt:         1_700_000_500,
		},
		{TransactionSet: "204", Direction: edidocument.DocumentDirectionInbound, Count: 0},
	}

	model := ediPartnerCardToModel(card)

	assert.Equal(t, &gqlmodel.EDIPartnerCustomer{
		ID: customerID.String(), Name: "Tyson Foods", Code: "TYSON",
	}, model.Customer)
	assert.Equal(t, []*gqlmodel.EDIPartnerCarrier{{
		CarrierID:   card.Carriers[0].CarrierID.String(),
		CarrierName: "Red River Freight",
		CarrierCode: new("RRF"),
		SCAC:        new("RRFX"),
		IsDefault:   true,
	}}, model.Carriers, "the channel's SCAC wins; a channel without its carrier is skipped")
	assert.Equal(t, &gqlmodel.EDIPartnerMappingSummary{
		ProfileID: new(card.Mappings.ProfileID.String()),
		UpdatedAt: new(1_700_000_000),
		Counts: []*gqlmodel.EDIMappingCount{
			{EntityType: edi.MappingEntityTypeLocation, Count: 41},
		},
	}, model.Mappings)
	assert.Equal(t, []*gqlmodel.EDIPartnerSetActivity{
		{
			TransactionSet: "214",
			Direction:      edidocument.DocumentDirectionOutbound,
			Count:          1284,
			FailedCount:    2,
			LastAt:         new(1_700_000_500),
		},
		{TransactionSet: "204", Direction: edidocument.DocumentDirectionInbound},
	}, model.Activity.Sets)
	assert.Equal(t, 3, model.Activity.Scorecard.OverdueAckCount,
		"overdue is counted once, by the attention's own read")
	require.NotNil(t, model.Activity.Scorecard.DeliverySuccessRate)
	assert.InDelta(t, 0.9, *model.Activity.Scorecard.DeliverySuccessRate, 0.0001)
	assert.Equal(t, 100, model.Activity.Since)
	assert.Equal(t, 50, model.Activity.SeriesSince)
	assert.Equal(t, 14400, model.Attention.OverdueAckAgeSeconds)
	assert.Equal(t, &gqlmodel.EDIPartnerCardReadiness{
		CompletedCount: 1,
		TotalCount:     2,
		Items: []*gqlmodel.EDIPartnerReadinessItem{
			{Key: "details", Label: "Partner details", Complete: true},
			{Key: "lading-active", Label: "An active Lading version", Complete: false},
		},
	}, model.Readiness)
}

func TestEDIPartnerCardToModel_CarriesTheLadingVersionsRunning(t *testing.T) {
	t.Parallel()

	card := partnerCardFixture()
	card.Lading = &ediservice.EDIPartnerLading{
		Project: &ladingproject.LadingProject{
			ID:           pulid.MustNew("ldp_"),
			EDIPartnerID: card.Partner.ID,
		},
		Active: &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_"), Number: 3},
		Shadow: &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_"), Number: 4},
	}

	model := ediPartnerCardToModel(card)

	require.NotNil(t, model.Lading)
	assert.Same(t, card.Lading.Project, model.Lading.Project)
	assert.Same(t, card.Lading.Active, model.Lading.Active)
	assert.Same(t, card.Lading.Shadow, model.Lading.Shadow)
}

func TestLadingVersionModel_ReadsTheEnvelopeAndSetsFromTheManifest(t *testing.T) {
	t.Parallel()

	manifest := &lading.Check{
		Partner: &lading.Partner{Interchange: &lading.Interchange{
			Them:    &lading.Party{Qualifier: "01", ID: "007912345"},
			Us:      &lading.Party{Qualifier: "02", ID: "TRNV"},
			Group:   &lading.GroupCodes{Theirs: "ACME", Ours: "TRNV"},
			Version: "004010",
			Usage:   "test",
		}},
		Maps: []lading.Map{
			{Name: "statusOut", Set: "214", Direction: lading.DirectionOutbound},
			{Name: "tenderIn", Set: "204", Direction: lading.DirectionInbound},
			{Name: "answerOut", Set: "990", Direction: lading.DirectionOutbound},
			{Name: "ordersCsv", Direction: lading.DirectionInbound},
		},
		Flows: []lading.Flow{
			{Name: "tender", Clauses: []lading.Clause{
				{Kind: lading.ClauseReceive, Set: "204", Map: "tenderIn", Acknowledge: "997"},
				{Kind: lading.ClauseRespond, Set: "990", Map: "answerOut"},
				{Kind: lading.ClauseSend, Set: "214", Map: "statusOut"},
				{Kind: lading.ClauseExpect, Set: "997", Within: 1800},
			}},
		},
	}

	interchange := ladingInterchangeToModel(manifest)
	require.NotNil(t, interchange)
	assert.Equal(t, "007912345", *interchange.TheirID)
	assert.Equal(t, "02", *interchange.OurQualifier)
	assert.Equal(t, "ACME", *interchange.TheirGroupCode)
	assert.Equal(t, "test", *interchange.Usage)
	assert.Equal(t, []string{}, interchange.Separators, "none written reads as an empty list")

	ack, within := "997", 1800
	assert.Equal(t, []*gqlmodel.EDILadingExchange{
		{
			TransactionSet: "204",
			Direction:      edidocument.DocumentDirectionInbound,
			Map:            "tenderIn",
			Acknowledgment: &ack,
		},
		{
			TransactionSet: "214", Direction: edidocument.DocumentDirectionOutbound, Map: "statusOut",
			Acknowledgment: &ack, AcknowledgmentWithin: &within,
		},
		{
			TransactionSet: "990", Direction: edidocument.DocumentDirectionOutbound, Map: "answerOut",
			Acknowledgment: &ack, AcknowledgmentWithin: &within,
		},
	}, ladingExchangesToModel(manifest), "a map that reads no X12 set is not an exchange")

	assert.Nil(t, ladingInterchangeToModel(&lading.Check{}), "a project with no partner file")
}
