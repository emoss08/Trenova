package ediconnectiondraft

import (
	_ "embed"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/acme_retail_spec.txt
var acmeRetailSpec string

var pageMarker = regexp.MustCompile(`(?m)^--- page (\d+) ---\n`)

func acmeSpec(t *testing.T) ediports.SpecDocument {
	t.Helper()

	text := acmeRetailSpec
	markers := pageMarker.FindAllStringSubmatchIndex(text, -1)
	pages := make([]ediports.SpecPage, 0, len(markers))
	for index, marker := range markers {
		number, convErr := strconv.Atoi(text[marker[2]:marker[3]])
		require.NoError(t, convErr)
		end := len(text)
		if index+1 < len(markers) {
			end = markers[index+1][0]
		}
		pages = append(pages, ediports.SpecPage{Number: number, Text: text[marker[1]:end]})
	}
	return ediports.SpecDocument{
		ID:       pulid.MustNew("doc_"),
		FileName: "acme-retail-edi-guide.pdf",
		Pages:    pages,
	}
}

func cite(page int, quote string) ediports.DraftCitation {
	return ediports.DraftCitation{Page: page, Quote: quote}
}

func value(field, v string, page int, quote string) ediports.DraftValueInput {
	return ediports.DraftValueInput{Field: field, Value: v, Confidence: 0.95, Citation: cite(page, quote)}
}

func acmeValues() []ediports.DraftValueInput {
	return []ediports.DraftValueInput{
		value(fieldPartnerCode, "ACMERETAIL", 1, "Trading Partner Code: ACMERETAIL"),
		value(fieldPartnerName, "Acme Retail Group", 1, "Trading Partner Name: Acme Retail Group"),
		value(fieldPartnerContactName, "Dana Whitfield", 1, "EDI Coordinator: Dana Whitfield"),
		value(fieldPartnerContactEmail, "edi@acmeretail.example", 1, "Email: edi@acmeretail.example"),
		value(fieldPartnerContactPhone, "+1 312 555 0142", 1, "Phone: +1 312 555 0142"),
		value(fieldPartnerTimezone, "America/Chicago", 1, "Central Time (America/Chicago)"),
		value(fieldTransportMethod, "AS2", 2, "Acme exchanges all documents over AS2 (Applicability Statement 2)."),
		value(fieldTransportEnvironment, "test", 2, "Begin in the test environment"),
		value("as2.partnerAs2Id", "ACMERETAILAS2", 2, "Acme AS2 Identifier: ACMERETAILAS2"),
		value("as2.localAs2Id", "TRENOVA-AS2", 2, "Carrier AS2 Identifier: TRENOVA-AS2"),
		value("as2.endpointUrl", "https://as2-test.acmeretail.example/as2/receive", 2,
			"Test AS2 URL: https://as2-test.acmeretail.example/as2/receive"),
		value("as2.mdnMode", "sync", 2, "MDN: synchronous, signed."),
		value("as2.signingAlgorithm", "sha256", 2, "Signing algorithm: SHA-256."),
		value("as2.encryptionAlgorithm", "aes256-cbc", 2, "Encryption algorithm: AES-256 (CBC)."),
		value(fieldPartnerQualifier, "01", 3, "Acme interchange sender: qualifier 01 (DUNS), ID 007912345"),
		value(fieldPartnerID, "007912345", 3, "Acme interchange sender: qualifier 01 (DUNS), ID 007912345"),
		value(fieldOurQualifier, "02", 3, "Carrier interchange receiver: qualifier 02 (SCAC), ID TRNV"),
		value(fieldOurID, "TRNV", 3, "Carrier interchange receiver: qualifier 02 (SCAC), ID TRNV"),
		value(fieldPartnerGSID, "ACMERETAIL", 3, "GS02 Application Sender Code: ACMERETAIL"),
		value(fieldOurGSID, "TRNV", 3, "GS03 Application Receiver Code: TRNV"),
		value(fieldX12Version, "004010", 3, "GS08 Version: 004010"),
		value(fieldUsageIndicator, "T", 3, "ISA15 usage indicator: T during testing"),
		value(fieldElementSep, "*", 3, "Element separator: * (asterisk)."),
		value(fieldSegmentSep, "~", 3, "Segment terminator: ~ (tilde)."),
		value(fieldComponentSep, ">", 3, "Sub-element separator: > (greater than)."),
		value(controlNumberField(edi.ControlNumberKindInterchange), "1000", 3,
			"Carrier interchange control numbers (ISA13) must start at 1000"),
		value(setField(edi.TransactionSet204, edi.DocumentDirectionInbound, setAttrSupported), "true", 4,
			"Acme sends the 204 Motor Carrier Load Tender to the carrier."),
		value(setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrSupported), "true", 4,
			"The carrier answers each tender with a 990 Response to a Load Tender"),
		value(setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrSupported), "true", 4,
			"The carrier sends the 214 Transportation Carrier Shipment Status Message"),
		value(setField(edi.TransactionSet210, edi.DocumentDirectionOutbound, setAttrSupported), "true", 4,
			"The carrier invoices with the 210 Motor Carrier Freight Details and Invoice."),
		value(setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrAck), "997", 4,
			"Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours."),
		value(setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrAckWindowMinutes), "240", 4,
			"Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours."),
		value(setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrAck), "997", 4,
			"Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours."),
		value(setField(edi.TransactionSet210, edi.DocumentDirectionOutbound, setAttrAck), "997", 4,
			"Acme expects a 997 Functional Acknowledgment for every 990, 214 and 210 within 4 hours."),
	}
}

func acmeRequest(t *testing.T) *ediports.ConnectionDraftRequest {
	t.Helper()

	return &ediports.ConnectionDraftRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Document:   acmeSpec(t),
		Values:     acmeValues(),
		Requirements: []ediports.DraftRequirementInput{
			{
				TransactionSet: edi.TransactionSet214, Direction: edi.DocumentDirectionOutbound,
				SegmentID: "MS1", Usage: ediports.RequirementUsageMandatory,
				Citation: cite(5, "MS1 Equipment Location is mandatory on every status."),
			},
			{
				TransactionSet: edi.TransactionSet214, Direction: edi.DocumentDirectionOutbound,
				SegmentID: "K1", Usage: ediports.RequirementUsageNotUsed,
				Citation: cite(5, "K1 Remarks is not used."),
			},
			{
				TransactionSet: edi.TransactionSet204, Direction: edi.DocumentDirectionInbound,
				SegmentID: "B2", ElementPosition: 4, Usage: ediports.RequirementUsageMandatory,
				Citation: cite(5, "B2-04 Shipment Identification Number is mandatory."),
			},
		},
		Gaps: []ediports.DraftGapInput{{
			Kind:    ediports.GapKindAmbiguous,
			Summary: "Accessorial charge codes are in Appendix B, which is not in this document",
			Citation: cite(6,
				"Detention and other accessorial charges are billed on the 210 with the Acme charge codes listed in Appendix B."),
		}},
	}
}

func valueOf(t *testing.T, values []ediports.DraftValue, field string) ediports.DraftValue {
	t.Helper()
	for _, v := range values {
		if v.Field == field {
			return v
		}
	}
	require.Failf(t, "missing field", "%s is not in the section", field)
	return ediports.DraftValue{}
}

func profileFor(
	t *testing.T,
	draft *ediports.ConnectionDraft,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) ediports.DraftDocumentProfile {
	t.Helper()
	for _, profile := range draft.DocumentProfiles {
		if profile.TransactionSet == set && profile.Direction == direction {
			return profile
		}
	}
	require.Failf(t, "missing profile", "%s %s", set, direction)
	return ediports.DraftDocumentProfile{}
}

func TestPlan_DraftsAWholeNewConnectionFromTheSpec(t *testing.T) {
	t.Parallel()

	h := newHarness()
	draft, err := h.service.PlanConnectionDraft(t.Context(), acmeRequest(t))
	require.NoError(t, err)

	assert.Empty(t, draft.Decisions)
	assert.Empty(t, draft.Failures)
	assert.True(t, draft.Ready, draft.Blockers)
	assert.Equal(t, ediports.PartnerMatchNew, draft.Partner.Match)
	assert.Equal(t, ediports.ChangeKindAdd, draft.Partner.Change)
	assert.Equal(t, domaintypes.StatusInactive, draft.Partner.Status, "nothing drafted can send")
	assert.Equal(t, 7, draft.PageCount)
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, draft.DocumentDigest)

	name := valueOf(t, draft.Partner.Fields, fieldPartnerName)
	assert.Equal(t, "Acme Retail Group", name.Value)
	assert.Equal(t, ediports.SourceKindDocument, name.Source.Kind)
	assert.Equal(t, 1, name.Source.Page)
	assert.Equal(t, "Trading Partner Name: Acme Retail Group", name.Source.Quote)
	assert.InDelta(t, 0.95, name.Confidence, 0.001)
	assert.Equal(t, "US", valueOf(t, draft.Partner.Fields, fieldPartnerCountry).Value)
	assert.Equal(t, ediports.ValueStatusDefaulted, valueOf(t, draft.Partner.Fields, fieldPartnerCountry).Status)

	assert.True(t, draft.Connection.Inbound)
	assert.True(t, draft.Connection.Outbound)
	assert.Equal(t, edi.ConnectionMethodAS2, draft.Connection.Method)
	assert.Len(t, draft.DocumentProfiles, 4)

	tender := profileFor(t, draft, edi.TransactionSet204, edi.DocumentDirectionInbound)
	assert.Equal(t, edi.DocumentStatusInactive, tender.Status)
	assert.Equal(t, "SM", valueOf(t, tender.Fields, setField(edi.TransactionSet204, edi.DocumentDirectionInbound, setAttrFunctionalGroupID)).Value)
	assert.Equal(t, "004010", valueOf(t, tender.Fields, setField(edi.TransactionSet204, edi.DocumentDirectionInbound, setAttrVersion)).Value)
	assert.True(t, tender.Template.Starter)
	assert.Empty(t, tender.ControlNumbers, "inbound documents carry the partner's numbers")

	status := profileFor(t, draft, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	ack := valueOf(t, status.Fields, setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrAck))
	assert.Equal(t, "997", ack.Value)
	require.Len(t, status.ControlNumbers, 1)
	assert.Equal(t, int64(1000), status.ControlNumbers[0].Start)
	assert.Equal(t, edi.ControlNumberKindInterchange, status.ControlNumbers[0].Kind)
	require.Len(t, status.Template.Unmet, 1, "the base 214 has no MS1, and writes no K1")
	assert.Equal(t, "MS1", status.Template.Unmet[0].SegmentID)
	assert.Equal(t, 5, status.Template.Unmet[0].Source.Page)

	response := profileFor(t, draft, edi.TransactionSet990, edi.DocumentDirectionOutbound)
	window := valueOf(t, response.Fields, setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrAckWindowMinutes))
	assert.Equal(t, "240", window.Value, "4 hours")

	require.NotNil(t, draft.Transport)
	assert.Equal(t, ediports.ChangeKindAdd, draft.Transport.Change)
	assert.Equal(t, domaintypes.StatusInactive, draft.Transport.Status)
	assert.Equal(t, "https://as2-test.acmeretail.example/as2/receive",
		valueOf(t, draft.Transport.Fields, "as2.endpointUrl").Value)
	labels := make([]string, 0, len(draft.Transport.Credentials))
	for _, credential := range draft.Transport.Credentials {
		labels = append(labels, credential.Label)
		assert.True(t, credential.Missing)
	}
	assert.Contains(t, labels, "Partner signing certificate")
	assert.Contains(t, labels, "Partner encryption certificate")

	require.Len(t, draft.Samples, 1)
	sample := draft.Samples[0]
	assert.True(t, sample.Parsed)
	assert.Equal(t, 7, sample.Page)
	assert.Equal(t, edi.TransactionSet204, sample.TransactionSet)
	assert.Equal(t, edi.DocumentDirectionInbound, sample.Direction)
	require.NotEmpty(t, sample.Checks)
	for _, check := range sample.Checks {
		assert.True(t, check.Matches, "%s: expected %q, sample has %q", check.Field, check.Expected, check.Found)
	}
	assert.NotEmpty(t, sample.Unmapped, "the sample's codes are not mapped yet")

	require.Len(t, draft.Gaps, 1)
	assert.Empty(t, draft.Gaps[0].Problem)
	assert.Equal(t, 6, draft.Gaps[0].Source.Page)

	encoded, err := sonic.Marshal(draft)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "Acme#Test2026")
}

func TestPlan_HoldsAValueWhoseQuoteIsNotOnThePage(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[0] = value(fieldPartnerCode, "ACMERETAIL", 3, "Partner code is ACMERETAIL")

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	code := valueOf(t, draft.Partner.Fields, fieldPartnerCode)
	assert.Equal(t, ediports.ValueStatusUnverified, code.Status)
	assert.Empty(t, code.Value)
	assert.Contains(t, code.Reason, "does not appear on page 3")
	assert.False(t, draft.Ready)
	assert.Contains(t, draft.Blockers, "Decide partner code")
}

func TestPlan_FollowsAQuoteToThePageItIsOn(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[0] = value(fieldPartnerCode, "ACMERETAIL", 2, "Trading Partner Code: ACMERETAIL")

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	code := valueOf(t, draft.Partner.Fields, fieldPartnerCode)
	assert.Equal(t, ediports.ValueStatusAccepted, code.Status)
	assert.Equal(t, 1, code.Source.Page)
	assert.Equal(t, "Cited as page 2; the quote is on page 1", code.Source.Note)
}

func TestPlan_RefusesAValueTheQuoteDoesNotSay(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[1] = value(fieldPartnerName, "Acme Wholesale", 1, "Trading Partner Name: Acme Retail Group")

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	name := valueOf(t, draft.Partner.Fields, fieldPartnerName)
	assert.Equal(t, ediports.ValueStatusUnverified, name.Status)
	assert.Equal(t, "The quote does not contain this value", name.Reason)
}

func TestPlan_HoldsALowConfidenceValue(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[5].Confidence = 0.55

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	timezone := valueOf(t, draft.Partner.Fields, fieldPartnerTimezone)
	assert.Equal(t, ediports.ValueStatusHeld, timezone.Status)
	assert.Empty(t, timezone.Value, "a held value is never written")
	assert.Contains(t, timezone.Reason, "Confidence 0.55")
	require.Len(t, draft.Decisions, 1)
	assert.Equal(t, fieldPartnerTimezone, draft.Decisions[0].Field)
	assert.False(t, draft.Ready)
}

func TestPlan_HoldsConflictingValuesUntilThePersonPicksOne(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values = append(req.Values, value("as2.endpointUrl",
		"https://as2.acmeretail.example/as2/receive", 2,
		"Production AS2 URL: https://as2.acmeretail.example/as2/receive"))

	h := newHarness()
	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	endpoint := valueOf(t, draft.Transport.Fields, "as2.endpointUrl")
	assert.Equal(t, ediports.ValueStatusHeld, endpoint.Status)
	assert.Empty(t, endpoint.Value)
	assert.Len(t, endpoint.Candidates, 2)
	assert.Contains(t, endpoint.Reason, "2 different values")

	req.Resolutions = []ediports.DraftResolutionInput{{
		Field:  "as2.endpointUrl",
		Value:  "https://as2-test.acmeretail.example/as2/receive",
		Source: ediports.ResolutionSourceCandidate,
	}}
	draft, err = h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	endpoint = valueOf(t, draft.Transport.Fields, "as2.endpointUrl")
	assert.Equal(t, ediports.ValueStatusAccepted, endpoint.Status)
	assert.Equal(t, 2, endpoint.Source.Page)
	assert.Contains(t, endpoint.Source.Note, "Chosen by the person")
	assert.True(t, draft.Ready, draft.Blockers)

	req.Resolutions[0].Value = "https://elsewhere.example/as2"
	draft, err = h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	endpoint = valueOf(t, draft.Transport.Fields, "as2.endpointUrl")
	assert.Equal(t, ediports.ValueStatusInvalid, endpoint.Status, "a pick must be one of the cited values")

	req.Resolutions[0].Source = ediports.ResolutionSourcePerson
	draft, err = h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	endpoint = valueOf(t, draft.Transport.Fields, "as2.endpointUrl")
	assert.Equal(t, ediports.ValueStatusAccepted, endpoint.Status)
	assert.Equal(t, ediports.SourceKindPerson, endpoint.Source.Kind)
}

func TestPlan_HoldsAValueTheSpecsOwnSampleContradicts(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	for index := range req.Values {
		if req.Values[index].Field == fieldOurID {
			req.Values[index] = value(fieldOurID, "TRENOVA", 2, "Carrier AS2 Identifier: TRENOVA-AS2")
		}
	}

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	ours := valueOf(t, draft.Interchange, fieldOurID)
	assert.Equal(t, ediports.ValueStatusHeld, ours.Status)
	kinds := make([]ediports.SourceKind, 0, len(ours.Candidates))
	for _, candidate := range ours.Candidates {
		kinds = append(kinds, candidate.Source.Kind)
	}
	assert.ElementsMatch(t, []ediports.SourceKind{ediports.SourceKindDocument, ediports.SourceKindSample}, kinds)
	assert.False(t, draft.Ready)
}

func TestPlan_TakesEnvelopeValuesFromTheSampleWhenTheTextIsSilent(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	kept := make([]ediports.DraftValueInput, 0, len(req.Values))
	for _, input := range req.Values {
		if input.Field != fieldPartnerQualifier && input.Field != fieldElementSep {
			kept = append(kept, input)
		}
	}
	req.Values = kept

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	qualifier := valueOf(t, draft.Interchange, fieldPartnerQualifier)
	assert.Equal(t, "01", qualifier.Value)
	assert.Equal(t, ediports.SourceKindSample, qualifier.Source.Kind)
	assert.Equal(t, 7, qualifier.Source.Page)
	assert.True(t, strings.HasPrefix(qualifier.Source.Quote, "ISA*00*"))
	assert.Equal(t, "*", valueOf(t, draft.Interchange, fieldElementSep).Value)
}

func TestPlan_RefusesSecretsAndUnknownFields(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"as2.privateKey", "transport.password", "knownHostKey", "as2.partnerSigningCertificate", "isa.securityInformation"} {
		req := acmeRequest(t)
		req.Values = append(req.Values, value(field, "Acme#Test2026", 2, "keystore password: Acme#Test2026"))
		_, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
		require.Error(t, err, field)
		assert.Contains(t, err.Error(), "never taken from a document", field)
	}

	req := acmeRequest(t)
	req.Values = append(req.Values, value("partner.favoriteColor", "blue", 1, "Acme Retail Group"))
	_, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `Unknown field "partner.favoriteColor"`)
}

func TestPlan_NeedsReadableText(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Document.Pages = nil
	_, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no readable text")
}

func existingAcme(req *ediports.ConnectionDraftRequest) *ediports.EDIPartnerSetup {
	partnerID := pulid.MustNew("edip_")
	tenant := req.TenantInfo
	envelope := edi.X12EnvelopeSettings{
		InterchangeSenderQualifier: "02", InterchangeSenderID: "TRNV",
		InterchangeReceiverQualifier: "01", InterchangeReceiverID: "007912345",
		ApplicationSenderCode: "TRNV", ApplicationReceiverCode: "ACMERETAIL",
		InterchangeUsageIndicator: "T", ElementSeparator: "*", SegmentTerminator: "~",
		ComponentSeparator: ">", RepetitionSeparator: "^",
	}
	statusTemplate := pulid.MustNew("editpl_")
	return &ediports.EDIPartnerSetup{
		Partner: &edi.EDIPartner{
			ID: partnerID, OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			Kind: edi.PartnerKindExternal, Status: domaintypes.StatusActive,
			Code: "ACMERETAIL", Name: "Acme Retail Group", Country: "US",
			ContactName: "Lee Park", ContactEmail: "edi@acmeretail.example",
			ContactPhone: "+1 312 555 0142", Timezone: "America/Chicago",
			EnabledForOutbound: true, Version: 4,
		},
		CommunicationProfiles: []*edi.EDICommunicationProfile{{
			ID: pulid.MustNew("edicp_"), EDIPartnerID: partnerID, Method: edi.ConnectionMethodAS2,
			Status: domaintypes.StatusActive, Name: "Acme Retail Group AS2", Version: 2,
			Config: map[string]any{
				"localAS2Id": "TRENOVA-AS2", "partnerAS2Id": "ACMERETAILAS2",
				"endpointUrl": "https://as2-test.acmeretail.example/as2/receive", "mdnMode": "sync",
				"signingAlgorithm": "sha256", "encryptionAlgorithm": "aes256-cbc", "environment": "test",
				"isaSenderQualifier": "02", "isaSenderId": "TRNV", "isaReceiverQualifier": "01",
				"isaReceiverId": "007912345", "gsSenderId": "TRNV", "gsReceiverId": "ACMERETAIL",
				"x12Version": "004010",
			},
			EncryptedSecrets: map[string]string{"privateKey": "trenova-envelope:v1:abc"},
		}},
		DocumentProfiles: []*edi.EDIPartnerDocumentProfile{{
			ID: pulid.MustNew("edidp_"), EDIPartnerID: partnerID,
			DocumentTypeID: documentTypeID(edi.TransactionSet214, edi.DocumentDirectionOutbound),
			TemplateID:     statusTemplate, Name: "Acme 214", Status: edi.DocumentStatusActive,
			Direction: edi.DocumentDirectionOutbound, TransactionSet: edi.TransactionSet214,
			X12VersionOverride: "004010", FunctionalGroupID: "QM", Envelope: envelope,
			Acknowledgment: edi.AcknowledgmentConfig{Expected: true, Type: edi.AcknowledgmentType997, SLAInMinutes: 120, MissingAckSeverity: edi.ValidationSeverityWarning},
			ValidationMode: edi.ValidationModeStrict, Version: 7,
			Template: &edi.EDITemplate{ID: statusTemplate, Name: "Acme 214 template"},
		}},
		MappingItems: []*edi.EDIMappingProfileItem{{
			EntityType: edi.MappingEntityTypeServiceType, SourceID: "OLD", TargetID: pulid.MustNew("st_"),
			TargetLabel: "Legacy",
		}},
		ControlNumbers: []*edi.EDIControlNumberSequence{{
			DocumentTypeID: documentTypeID(edi.TransactionSet214, edi.DocumentDirectionOutbound),
			Kind:           edi.ControlNumberKindInterchange, NextValue: 5000,
		}},
	}
}

func TestPlan_ARevisedSpecForAKnownPartnerIsAChangeSet(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	existing := existingAcme(req)
	h.repo.partners = []*ediports.EDIPartnerSetup{existing}
	standard := pulid.MustNew("st_")
	h.repo.targets[edi.MappingEntityTypeServiceType] = map[pulid.ID]string{standard: "STD"}
	req.Mappings = []ediports.DraftMappingInput{{
		EntityType: edi.MappingEntityTypeServiceType, SourceValue: "STD", SourceLabel: "Standard",
		TargetID: standard, Confidence: 0.9,
		Citation: cite(6, "7.2 Service levels (L5): STD Standard, EXP Expedited."),
	}}

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, ediports.PartnerMatchExisting, draft.Partner.Match)
	assert.Equal(t, existing.Partner.ID, draft.Partner.PartnerID)
	assert.Contains(t, draft.Partner.Reasons, "Same partner code")
	assert.Equal(t, domaintypes.StatusActive, draft.Partner.Status, "a re-run never changes status")
	assert.Equal(t, ediports.ChangeKindChange, draft.Partner.Change)
	contact := valueOf(t, draft.Partner.Fields, fieldPartnerContactName)
	assert.Equal(t, ediports.ChangeKindChange, contact.Change)
	assert.Equal(t, "Lee Park", contact.Current)
	assert.Equal(t, "Dana Whitfield", contact.Value)
	assert.Equal(t, ediports.ChangeKindSame, valueOf(t, draft.Partner.Fields, fieldPartnerName).Change)

	require.NotNil(t, draft.Transport)
	assert.Equal(t, existing.CommunicationProfiles[0].ID, draft.Transport.ProfileID)
	assert.Equal(t, ediports.ChangeKindSame, draft.Transport.Change)
	assert.Equal(t, domaintypes.StatusActive, draft.Transport.Status)

	status := profileFor(t, draft, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	assert.Equal(t, existing.DocumentProfiles[0].ID, status.ProfileID)
	assert.Equal(t, ediports.ChangeKindSame, status.Change, "the profile already matches the spec")
	assert.Equal(t, ediports.ChangeKindSame, status.Template.Change)
	assert.Equal(t, keptTemplateNote, status.Template.Note)
	window := valueOf(t, status.Fields, setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrAckWindowMinutes))
	assert.Equal(t, ediports.ValueStatusKept, window.Status)
	assert.Equal(t, edi.DocumentStatusActive, status.Status)
	require.Len(t, status.ControlNumbers, 1)
	assert.Equal(t, ediports.ChangeKindSame, status.ControlNumbers[0].Change)
	assert.Equal(t, int64(5000), status.ControlNumbers[0].Current)
	assert.Contains(t, status.ControlNumbers[0].Problem, "would reissue numbers")

	assert.Equal(t, ediports.ChangeKindAdd, profileFor(t, draft, edi.TransactionSet990, edi.DocumentDirectionOutbound).Change)

	require.Len(t, draft.Mappings, 1)
	assert.Equal(t, ediports.ChangeKindAdd, draft.Mappings[0].Change)
	assert.Equal(t, "STD", draft.Mappings[0].TargetLabel)
	require.Len(t, draft.Kept, 1)
	assert.Contains(t, draft.Kept[0].Label, "OLD")
	assert.True(t, draft.Ready, draft.Blockers)
}

func TestPlan_NeverPicksBetweenTwoMatchingPartners(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	first := existingAcme(req)
	second := existingAcme(req)
	second.Partner.Code = "ACME-2"
	second.Partner.Name = "Acme Retail Group West"
	h.repo.partners = []*ediports.EDIPartnerSetup{first, second}

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, ediports.PartnerMatchAmbiguous, draft.Partner.Match)
	assert.Len(t, draft.Partner.Candidates, 2)
	assert.False(t, draft.Ready)
	assert.Contains(t, draft.Blockers, "Name which existing partner this spec is for")
}

func TestPlan_ChoosesTheTemplateThatMeetsTheMostRequirements(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	versionID := pulid.MustNew("editv_")
	h.repo.templates = []*edi.EDITemplateVersion{{
		ID: versionID, TemplateID: pulid.MustNew("editpl_"), VersionNumber: 3, IsActive: true,
		Template: &edi.EDITemplate{Name: "Retail 214", TransactionSet: edi.TransactionSet214, Direction: edi.DocumentDirectionOutbound},
		Segments: []*edi.EDITemplateSegment{
			{SegmentID: "AT7", Required: true, Elements: []edi.TemplateElement{{Position: 1}}},
			{SegmentID: "MS1", Required: true, Elements: []edi.TemplateElement{{Position: 1}}},
		},
	}}
	h.repo.templates[0].Template.ID = h.repo.templates[0].TemplateID

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	status := profileFor(t, draft, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	assert.Equal(t, "Retail 214 (version 3)", status.Template.Name)
	assert.False(t, status.Template.Starter)
	assert.Empty(t, status.Template.Unmet)
	assert.Equal(t, 2, status.Template.Met)
	require.Len(t, status.Template.Considered, 2)
}

func TestPlan_ListsWhatAnInvalidPartnerSettingWouldBreak(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.setup.settingsErrors = nil
	req := acmeRequest(t)
	req.Values = append(req.Values, value(fieldCarrierSCAC, "TRNV", 3, "Carrier interchange receiver: qualifier 02 (SCAC), ID TRNV"))

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, draft.Failures)

	h.setup.settingsErrors = []edix12.Diagnostic{{Severity: edi.ValidationSeverityError, Path: "carrier.scac", Message: "SCAC is required"}}
	draft, err = h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)
	require.NotEmpty(t, draft.Failures)
	assert.Equal(t, "partnerSettings.carrier.scac", draft.Failures[0].Field)
	assert.False(t, draft.Ready)
}

func TestDocumentDigest_IsTheTextInPageOrder(t *testing.T) {
	t.Parallel()

	pages := []ediports.SpecPage{{Number: 2, Text: "b"}, {Number: 1, Text: "a"}}
	reordered := []ediports.SpecPage{{Number: 1, Text: "a"}, {Number: 2, Text: "b"}}
	assert.Equal(t, DocumentDigest(pages), DocumentDigest(reordered))
	assert.NotEqual(t, DocumentDigest(pages), DocumentDigest([]ediports.SpecPage{{Number: 1, Text: "a"}, {Number: 2, Text: "c"}}))
}

func TestRedact_KeepsSecretsOutOfQuotes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "the keystore password: [redacted] is sent separately",
		redact("the keystore password: Acme#Test2026 is sent separately"))
	assert.Equal(t, "Shared key = [redacted]", redact("Shared key = s3cr3t"))
	assert.Equal(t, "cert: [redacted key or certificate]",
		redact("cert: -----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"))
	assert.Equal(t,
		"ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		redact("ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME"),
	)
	assert.Equal(t, "ISA*00*          *00*          *ZZ*ACME",
		redact("ISA*00*          *00*          *ZZ*ACME"), "empty authorization is kept")
}
