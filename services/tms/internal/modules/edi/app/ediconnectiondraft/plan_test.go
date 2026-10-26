package ediconnectiondraft

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lading"
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

func setFor(
	t *testing.T,
	draft *ediports.ConnectionDraft,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) ediports.DraftSet {
	t.Helper()
	for _, drafted := range draft.Sets {
		if drafted.TransactionSet == set && drafted.Direction == direction {
			return drafted
		}
	}
	require.Failf(t, "missing set", "%s %s", set, direction)
	return ediports.DraftSet{}
}

func projectFile(t *testing.T, files []lading.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file.Source
		}
	}
	require.Failf(t, "missing file", "%s is not in the project", path)
	return ""
}

func draftedFile(
	t *testing.T,
	project *ediports.DraftProject,
	path string,
) ediports.DraftProjectFile {
	t.Helper()
	require.NotNil(t, project)
	for _, file := range project.Files {
		if file.Path == path {
			return file
		}
	}
	require.Failf(t, "missing file", "%s is not in the drafted project", path)
	return ediports.DraftProjectFile{}
}

func TestPlan_DraftsAWholeNewConnectionFromTheSpec(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
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
	assert.Len(t, draft.Sets, 4)

	tender := setFor(t, draft, edi.TransactionSet204, edi.DocumentDirectionInbound)
	assert.Equal(t, ediports.ChangeKindAdd, tender.Change)
	assert.Equal(t, "004010", valueOf(t, tender.Fields, setField(edi.TransactionSet204, edi.DocumentDirectionInbound, setAttrVersion)).Value)
	assert.Equal(t, 1, tender.Requirements)
	assert.Empty(t, tender.ControlNumbers, "inbound documents carry the partner's numbers")

	status := setFor(t, draft, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	ack := valueOf(t, status.Fields, setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrAck))
	assert.Equal(t, "997", ack.Value)
	require.Len(t, status.ControlNumbers, 1)
	assert.Equal(t, int64(1000), status.ControlNumbers[0].Start)
	assert.Equal(t, edi.ControlNumberKindInterchange, status.ControlNumbers[0].Kind)
	assert.Equal(t, 2, status.Requirements, "MS1 is required and K1 is not used")
	for _, unmet := range status.Unmet {
		assert.Equal(t, 5, unmet.Source.Page, "an unmet rule cites the page that asks for it")
	}

	require.NotNil(t, draft.Project)
	assert.Equal(t, ediports.ChangeKindAdd, draft.Project.Change)
	assert.Equal(t, lading.RoleCarrier, draft.Project.Role, "the partner tenders and the carrier answers")
	assert.Zero(t, draft.Project.Errors, draft.Project.Diagnostics)
	assert.Equal(t, ediports.ChangeKindAdd, draftedFile(t, draft.Project, "guides/214.lading").Change)
	assert.Equal(t, ediports.ChangeKindAdd, draftedFile(t, draft.Project, "guides/204.lading").Change)

	response := setFor(t, draft, edi.TransactionSet990, edi.DocumentDirectionOutbound)
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

func TestPlan_TakesSettingsAsTheGuideWritesThem(t *testing.T) {
	t.Parallel()

	written := map[string]string{
		"as2.mdnMode":             "Synchronous, signed",
		"as2.signingAlgorithm":    "SHA-256",
		"as2.encryptionAlgorithm": "AES-256 (CBC)",
		fieldPartnerQualifier:     "qualifier 01 (DUNS)",
		fieldOurQualifier:         "SCAC",
		fieldElementSep:           "* (asterisk)",
		fieldSegmentSep:           "tilde",
	}
	req := acmeRequest(t)
	for idx := range req.Values {
		if v, ok := written[req.Values[idx].Field]; ok {
			req.Values[idx].Value = v
		}
	}

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.Empty(t, draft.Decisions)
	assert.Empty(t, draft.Failures)
	assert.True(t, draft.Ready, draft.Blockers)
	require.NotNil(t, draft.Transport)
	assert.Equal(t, "sync", valueOf(t, draft.Transport.Fields, "as2.mdnMode").Value)
	assert.Equal(t, "sha256", valueOf(t, draft.Transport.Fields, "as2.signingAlgorithm").Value)
	assert.Equal(t, "aes256-cbc", valueOf(t, draft.Transport.Fields, "as2.encryptionAlgorithm").Value)
	assert.Equal(t, "01", valueOf(t, draft.Interchange, fieldPartnerQualifier).Value)
	assert.Equal(t, "02", valueOf(t, draft.Interchange, fieldOurQualifier).Value)
	assert.Equal(t, "*", valueOf(t, draft.Interchange, fieldElementSep).Value)
	assert.Equal(t, "~", valueOf(t, draft.Interchange, fieldSegmentSep).Value)
}

func TestCatalog_ReadsAValueThroughItsSpellings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field string
		value string
		want  string
		fails bool
	}{
		{field: "as2.mdnMode", value: "synchronous, signed", want: "sync"},
		{field: "as2.mdnMode", value: "Asynchronous MDN to the return URL", want: "async"},
		{field: "as2.mdnMode", value: "synchronous or asynchronous", fails: true},
		{field: "as2.signingAlgorithm", value: "SHA 256", want: "sha256"},
		{field: "as2.signingAlgorithm", value: "SHA-512 with RSA", want: "sha512"},
		{field: "as2.signingAlgorithm", value: "MD5", fails: true},
		{field: "as2.encryptionAlgorithm", value: "AES-256-GCM", want: "aes256-gcm"},
		{field: "as2.encryptionAlgorithm", value: "AES-128 (CBC)", want: "aes128-cbc"},
		{field: "as2.encryptionAlgorithm", value: "3DES", want: "3des"},
		{field: fieldPartnerQualifier, value: "DUNS plus 4", want: "16"},
		{field: fieldPartnerQualifier, value: "DUNS or SCAC", fails: true},
		{field: fieldTransportMethod, value: "AS2 over HTTPS", want: "AS2"},
		{field: fieldTransportEnvironment, value: "test or production", fails: true},
		{field: fieldElementSep, value: "| (pipe)", want: "|"},
		{field: "as2.mdnMode", value: "   ", fails: true},
	}
	for _, tt := range tests {
		t.Run(tt.field+"/"+tt.value, func(t *testing.T) {
			t.Parallel()

			got, err := fieldCatalog[tt.field].normalize(tt.value)
			if tt.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlan_HoldsAValueWhoseQuoteIsNotOnThePage(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[0] = value(fieldPartnerCode, "ACMERETAIL", 3, "Partner code is ACMERETAIL")

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
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

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
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

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	name := valueOf(t, draft.Partner.Fields, fieldPartnerName)
	assert.Equal(t, ediports.ValueStatusUnverified, name.Status)
	assert.Equal(t, "The quote does not contain this value", name.Reason)
}

func TestPlan_HoldsALowConfidenceValue(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values[1].Confidence = 0.55

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	name := valueOf(t, draft.Partner.Fields, fieldPartnerName)
	assert.Equal(t, ediports.ValueStatusHeld, name.Status)
	assert.Empty(t, name.Value, "a held value is never written")
	assert.Contains(t, name.Reason, "Confidence 0.55")
	require.Len(t, draft.Decisions, 1)
	assert.Equal(t, fieldPartnerName, draft.Decisions[0].Field)
	assert.Contains(t, draft.Blockers, "Decide partner name")
	assert.False(t, draft.Ready)
}

func TestPlan_HoldsConflictingValuesUntilThePersonPicksOne(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Values = append(req.Values, value("as2.endpointUrl",
		"https://as2.acmeretail.example/as2/receive", 2,
		"Production AS2 URL: https://as2.acmeretail.example/as2/receive"))

	h := newHarness(t)
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

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
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

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
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
		_, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
		require.Error(t, err, field)
		assert.Contains(t, err.Error(), "never taken from a document", field)
	}

	req := acmeRequest(t)
	req.Values = append(req.Values, value("partner.favoriteColor", "blue", 1, "Acme Retail Group"))
	_, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `Unknown field "partner.favoriteColor"`)
}

func TestPlan_DraftsTheSameFromAQuoteWhoseSecretWasRedacted(t *testing.T) {
	t.Parallel()

	const raw = "For the test environment, the keystore password: Acme#Test2026 is sent separately."
	const redacted = "For the test environment, the keystore password: [redacted] is sent separately."

	planWith := func(quote string) *ediports.ConnectionDraft {
		req := acmeRequest(t)
		for index := range req.Values {
			if req.Values[index].Field == fieldTransportEnvironment {
				req.Values[index].Citation = cite(2, quote)
			}
		}
		draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
		require.NoError(t, err)
		return draft
	}

	fromRaw := planWith(raw)
	fromRedacted := planWith(redacted)

	require.NotNil(t, fromRaw.Transport)
	require.NotNil(t, fromRedacted.Transport)
	environment := valueOf(t, fromRedacted.Transport.Fields, fieldTransportEnvironment)
	assert.Equal(t, ediports.ValueStatusAccepted, environment.Status)
	assert.Equal(t, "test", environment.Value)
	assert.Equal(t, 2, environment.Source.Page)
	assert.Equal(t, redacted, environment.Source.Quote)
	assert.Equal(t, valueOf(t, fromRaw.Transport.Fields, fieldTransportEnvironment), environment)
	assert.Equal(t, fromRaw.Ready, fromRedacted.Ready)
	assert.Equal(t, fromRaw.Blockers, fromRedacted.Blockers)
}

func TestPlan_NeedsReadableText(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	req.Document.Pages = nil
	_, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no readable text")
}

// acmeVersion is the active Lading version a partner set up before this guide would have: the
// carrier starter for Acme's interchange, receiving 204s and sending 214s.
func acmeVersion(t *testing.T, projectID pulid.ID) *edi.LadingVersion {
	t.Helper()
	client := lading.Default()
	starter, err := client.Starter(t.Context(), &lading.StarterRequest{
		Name:        "ACMERETAIL",
		Label:       "Acme Retail Group",
		Role:        lading.RoleCarrier,
		Zone:        "America/Chicago",
		Version:     "004010",
		Them:        &[2]string{"01", "007912345"},
		Us:          &[2]string{"02", "TRNV"},
		Group:       &[2]string{"ACMERETAIL", "TRNV"},
		Usage:       valueTest,
		Acknowledge: "997",
		AckWithin:   240,
		Sets:        []string{"204", "214"},
	})
	require.NoError(t, err)
	encoded, err := lading.Encode(
		&lading.Project{Files: starter.Files, Samples: starter.Samples},
		&lading.Context{Org: lading.Org{Name: "Trenova Logistics", SCAC: "TRNV", Zone: "America/Chicago"}},
	)
	require.NoError(t, err)
	check, err := client.Check(t.Context(), encoded)
	require.NoError(t, err)
	require.Zero(t, check.Errors)
	return &edi.LadingVersion{
		ID:        pulid.MustNew("ldv_"),
		ProjectID: projectID,
		Number:    2,
		Status:    edi.LadingVersionStatusActive,
		Files:     starter.Files,
		Manifest:  *check,
	}
}

func existingAcme(t *testing.T, req *ediports.ConnectionDraftRequest) *ediports.EDIPartnerSetup {
	t.Helper()
	partnerID := pulid.MustNew("edip_")
	tenant := req.TenantInfo
	project := &edi.LadingProject{
		ID:             pulid.MustNew("ldp_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		EDIPartnerID:   partnerID,
	}
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
			},
			EncryptedSecrets: map[string]string{"privateKey": "trenova-envelope:v1:abc"},
		}},
		Project:       project,
		LatestVersion: acmeVersion(t, project.ID),
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

	h := newHarness(t)
	req := acmeRequest(t)
	existing := existingAcme(t, req)
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

	status := setFor(t, draft, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	assert.NotEqual(t, ediports.ChangeKindAdd, status.Change, "the project already sends 214s")
	require.Len(t, status.ControlNumbers, 1)
	assert.Equal(t, ediports.ChangeKindSame, status.ControlNumbers[0].Change)
	assert.Equal(t, int64(5000), status.ControlNumbers[0].Current)
	assert.Contains(t, status.ControlNumbers[0].Problem, "would reissue numbers")

	assert.Equal(t, ediports.ChangeKindAdd, setFor(t, draft, edi.TransactionSet990, edi.DocumentDirectionOutbound).Change)

	require.NotNil(t, draft.Project)
	assert.Equal(t, existing.Project.ID, draft.Project.ProjectID)
	assert.Equal(t, ediports.ChangeKindChange, draft.Project.Change, "the guide's requirements are added")
	assert.Equal(t, ediports.ChangeKindAdd, draftedFile(t, draft.Project, "guides/214.lading").Change)
	gaps := make([]string, 0, len(draft.Gaps))
	for _, gap := range draft.Gaps {
		gaps = append(gaps, gap.Summary)
	}
	assert.Contains(t, gaps, fmt.Sprintf(noMapSummary, titleSet(edi.TransactionSet990, edi.DocumentDirectionOutbound)),
		"a set the project has no map for is named for the studio")

	require.Len(t, draft.Mappings, 1)
	assert.Equal(t, ediports.ChangeKindAdd, draft.Mappings[0].Change)
	assert.Equal(t, "STD", draft.Mappings[0].TargetLabel)
	kept := make([]string, 0, len(draft.Kept))
	for _, item := range draft.Kept {
		kept = append(kept, item.Label)
	}
	assert.Contains(t, strings.Join(kept, "\n"), "OLD")
	assert.True(t, draft.Ready, draft.Blockers)
}

func TestPlan_AnUnsureValueThatIsAlreadySavedChangesNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(t, req)}
	for idx := range req.Values {
		if req.Values[idx].Field == "as2.endpointUrl" {
			req.Values[idx].Confidence = 0.7
		}
	}

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	endpoint := valueOf(t, draft.Transport.Fields, "as2.endpointUrl")
	assert.Equal(t, ediports.ValueStatusKept, endpoint.Status)
	assert.Equal(t, ediports.ChangeKindSame, endpoint.Change)
	assert.Equal(t, "https://as2-test.acmeretail.example/as2/receive", endpoint.Value)
	assert.Equal(t, 2, endpoint.Source.Page, "the guide's page still stands behind it")
	for _, decision := range draft.Decisions {
		assert.NotEqual(t, "as2.endpointUrl", decision.Field)
	}
	assert.NotContains(t, draft.Blockers, "Decide AS2 endpoint URL")

	moved := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(t, moved)}
	for idx := range moved.Values {
		if moved.Values[idx].Field == "as2.endpointUrl" {
			moved.Values[idx] = value("as2.endpointUrl", "https://as2.acmeretail.example/as2/receive",
				2, "Production AS2 URL: https://as2.acmeretail.example/as2/receive")
			moved.Values[idx].Confidence = 0.7
		}
	}
	draft, err = h.service.PlanConnectionDraft(t.Context(), moved)
	require.NoError(t, err)
	assert.Contains(t, draft.Blockers, "Decide AS2 endpoint URL",
		"an unsure value that would change what is saved is still the person's to decide")
}

func TestPlan_NeverPicksBetweenTwoMatchingPartners(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	first := existingAcme(t, req)
	second := existingAcme(t, req)
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

func TestPlan_WritesTheGuidesRequirementsIntoGuideFilesWithTheirPages(t *testing.T) {
	t.Parallel()

	plan, err := newHarness(t).service.PreviewConnectionDraft(t.Context(), acmeRequest(t))
	require.NoError(t, err)
	require.NotNil(t, plan.Project)
	assert.Equal(t, ediports.ChangeKindAdd, plan.Project.Change)
	assert.Nil(t, plan.Project.Before)

	status := projectFile(t, plan.Project.Files, "guides/214.lading")
	assert.True(t, strings.HasPrefix(status, "guide "), status)
	assert.Contains(t, status, "from x12.004010.214")
	assert.Contains(t, status, "/// p.5: MS1 Equipment Location is mandatory on every status.")
	assert.Contains(t, status, "/// p.5: K1 Remarks is not used.")

	tender := projectFile(t, plan.Project.Files, "guides/204.lading")
	assert.Contains(t, tender, "B2 { 04 required }")

	maps := 0
	for _, file := range plan.Project.Files {
		if strings.HasPrefix(strings.TrimSpace(file.Source), "map ") &&
			strings.Contains(file.Source, "x12.004010.214") {
			maps++
		}
	}
	assert.Zero(t, maps, "every 214 map reads or writes through the partner's guide, not the base set")

	paths := make([]string, 0, len(plan.Project.Samples))
	for _, sample := range plan.Project.Samples {
		paths = append(paths, sample.Path)
	}
	assert.Contains(t, paths, "samples/spec-page-7.x12", "the guide's own example is kept as a sample")
}

func TestPlan_NamesTheAcknowledgmentItChoseWhenTheGuideAsksForTwo(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	for index := range req.Values {
		if req.Values[index].Field == setField(edi.TransactionSet210, edi.DocumentDirectionOutbound, setAttrAck) {
			req.Values[index] = value(req.Values[index].Field, "999", 4,
				"Acme does not support the 999 Implementation Acknowledgment.")
		}
	}

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	summaries := make([]string, 0, len(draft.Gaps))
	for _, gap := range draft.Gaps {
		summaries = append(summaries, gap.Summary)
	}
	assert.Contains(t, summaries, fmt.Sprintf(ackConflict, "997 and 999", "997"))
}

func TestDocumentDigest_IsTheTextInPageOrder(t *testing.T) {
	t.Parallel()

	pages := []ediports.SpecPage{{Number: 2, Text: "b"}, {Number: 1, Text: "a"}}
	reordered := []ediports.SpecPage{{Number: 1, Text: "a"}, {Number: 2, Text: "b"}}
	assert.Equal(t, DocumentDigest(pages), DocumentDigest(reordered))
	assert.NotEqual(t, DocumentDigest(pages), DocumentDigest([]ediports.SpecPage{{Number: 1, Text: "a"}, {Number: 2, Text: "c"}}))
}
