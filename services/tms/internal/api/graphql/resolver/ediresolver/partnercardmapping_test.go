package ediresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/pulid"
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
				{Key: "test-case", Label: "At least one passing test case"},
			},
		},
		Mappings: &ediports.EDIMappingSummary{},
		Activity: &ediservice.EDIPartnerActivity{
			Since:         100,
			SeriesSince:   50,
			BucketSeconds: 3600,
			Scorecard: &ediports.EDIPartnerScorecardRow{
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
		Credentials: []ediports.CredentialRequirement{
			{Keys: []string{"password", "privateKey"}, Kind: ediports.CredentialKindSecret, Required: true, Missing: true},
			{Keys: []string{"knownHostKey"}, Kind: ediports.CredentialKindSetting, Required: true},
			{Keys: nil, Kind: ediports.CredentialKindSecret},
		},
		Certificates: []ediservice.EDIPartnerCertificate{
			{
				Role: ediservice.PartnerCertificateRolePartnerEncryption,
				Summary: &as2.CertificateSummary{
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
	assert.Equal(t, []*edi.EDIPartnerDocumentProfile{}, model.DocumentProfiles)
	assert.Equal(t, []*edi.EDIMessage{}, model.Attention.OverdueAcks)
	assert.Equal(t, []*edi.EDIInboundFile{}, model.Attention.QuarantinedFiles)

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
	card.Mappings = &ediports.EDIMappingSummary{
		ProfileID: pulid.MustNew("edimp_"),
		UpdatedAt: 1_700_000_000,
		Counts: []*ediports.EDIMappingEntityCount{
			{EntityType: edi.MappingEntityTypeLocation, Count: 41},
		},
	}
	card.Activity.Sets = []*ediports.EDIPartnerSetActivityRow{
		{TransactionSet: "214", Direction: edi.DocumentDirectionOutbound, Count: 1284, FailedCount: 2, LastAt: 1_700_000_500},
		{TransactionSet: "204", Direction: edi.DocumentDirectionInbound, Count: 0},
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
		{TransactionSet: "214", Direction: edi.DocumentDirectionOutbound, Count: 1284, FailedCount: 2, LastAt: new(1_700_000_500)},
		{TransactionSet: "204", Direction: edi.DocumentDirectionInbound},
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
			{Key: "test-case", Label: "At least one passing test case", Complete: false},
		},
	}, model.Readiness)
}
