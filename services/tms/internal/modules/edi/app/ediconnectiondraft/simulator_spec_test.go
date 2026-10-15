package ediconnectiondraft

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func simulatorGuide(t *testing.T) ediports.SpecDocument {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..", "..",
		"dev", "edi-partner-sim", "specs", "simpartner-implementation-guide.txt")
	text, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err, "the partner simulator's guide is the fixture for this test")
	return ediports.SpecDocument{
		ID:       pulid.MustNew("doc_"),
		FileName: "simpartner-implementation-guide.txt",
		Pages:    []ediports.SpecPage{{Number: 1, Text: string(text)}},
	}
}

func simulatorValues() []ediports.DraftValueInput {
	sim := func(field, v, quote string) ediports.DraftValueInput {
		return value(field, v, 1, quote)
	}
	return []ediports.DraftValueInput{
		sim(fieldPartnerCode, "SIMPARTNER", "Trading Partner Code: SIMPARTNER"),
		sim(fieldPartnerName, "Trenova Partner Simulator", "Trading Partner Name: Trenova Partner Simulator"),
		sim(fieldPartnerContactName, "Simulator EDI desk", "EDI Contact: Simulator EDI desk"),
		sim(fieldPartnerContactEmail, "sim-edi@partner-sim.example", "Email: sim-edi@partner-sim.example"),
		sim(fieldPartnerContactPhone, "+1 312 555 0190", "Phone: +1 312 555 0190"),
		sim(fieldPartnerTimezone, "Etc/UTC", "UTC (Etc/UTC)"),
		sim(fieldTransportMethod, "AS2", "The simulator exchanges documents over AS2."),
		sim("as2.partnerAs2Id", "SIMPARTNER", "Simulator AS2 Identifier: SIMPARTNER"),
		sim("as2.localAs2Id", "TRENOVA", "Carrier AS2 Identifier: TRENOVA"),
		sim("as2.endpointUrl", "http://localhost:9210/as2", "AS2 URL: http://localhost:9210/as2"),
		sim("as2.mdnMode", "sync", "MDN: synchronous, signed."),
		sim("as2.signingAlgorithm", "sha256", "Signing algorithm: SHA-256."),
		sim("as2.encryptionAlgorithm", "aes256-cbc", "Encryption algorithm: AES-256 (CBC)."),
		sim(fieldPartnerQualifier, "ZZ", "Simulator interchange sender: qualifier ZZ (mutually defined), ID SIMPARTNER"),
		sim(fieldPartnerID, "SIMPARTNER", "Simulator interchange sender: qualifier ZZ (mutually defined), ID SIMPARTNER"),
		sim(fieldOurQualifier, "ZZ", "Carrier interchange receiver: qualifier ZZ (mutually defined), ID TRENOVA"),
		sim(fieldOurID, "TRENOVA", "Carrier interchange receiver: qualifier ZZ (mutually defined), ID TRENOVA"),
		sim(fieldPartnerGSID, "SIMPARTNER", "GS02 Application Sender Code: SIMPARTNER"),
		sim(fieldOurGSID, "TRENOVA", "GS03 Application Receiver Code: TRENOVA"),
		sim(fieldX12Version, "004010", "GS08 Version: 004010"),
		sim(fieldUsageIndicator, "T", "ISA15 usage indicator: T."),
		sim(fieldElementSep, "*", "Element separator: * (asterisk)."),
		sim(fieldSegmentSep, "~", "Segment terminator: ~ (tilde)."),
		sim(fieldComponentSep, ">", "Sub-element separator: > (greater than)."),
		sim(controlNumberField(edi.ControlNumberKindInterchange), "1",
			"Carrier interchange control numbers (ISA13) start at 1"),
		sim(setField(edi.TransactionSet204, edi.DocumentDirectionInbound, setAttrSupported), "true",
			"The simulator sends the 204 Motor Carrier Load Tender to the carrier."),
		sim(setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrSupported), "true",
			"The carrier answers each tender with a 990 Response to a Load Tender."),
		sim(setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrSupported), "true",
			"The carrier sends the 214 Transportation Carrier Shipment Status Message"),
		sim(setField(edi.TransactionSet990, edi.DocumentDirectionOutbound, setAttrAck), "997",
			"The simulator returns a 997 Functional Acknowledgment for every transaction set it receives."),
		sim(setField(edi.TransactionSet214, edi.DocumentDirectionOutbound, setAttrAck), "997",
			"The simulator returns a 997 Functional Acknowledgment for every transaction set it receives."),
	}
}

func TestPlan_ThePartnerSimulatorsGuideDraftsAConnectionToTheSimulator(t *testing.T) {
	t.Parallel()

	h := newHarness()
	draft, err := h.service.PlanConnectionDraft(t.Context(), &ediports.ConnectionDraftRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Document:   simulatorGuide(t),
		Values:     simulatorValues(),
		Requirements: []ediports.DraftRequirementInput{{
			TransactionSet: edi.TransactionSet204, Direction: edi.DocumentDirectionInbound,
			SegmentID: "L3", Usage: ediports.RequirementUsageMandatory,
			Citation: cite(1, "L3 Total Weight and Charges is mandatory."),
		}},
	})
	require.NoError(t, err)

	assert.Empty(t, draft.Decisions)
	assert.Empty(t, draft.Failures)
	assert.True(t, draft.Ready, draft.Blockers)
	require.NotNil(t, draft.Transport)
	assert.Equal(t, "http://localhost:9210/as2", valueOf(t, draft.Transport.Fields, "as2.endpointUrl").Value)
	assert.Len(t, draft.DocumentProfiles, 3)

	require.Len(t, draft.Samples, 1, "the guide carries the tender the simulator sends")
	sample := draft.Samples[0]
	require.True(t, sample.Parsed, sample.Error)
	assert.Equal(t, edi.TransactionSet204, sample.TransactionSet)
	assert.Equal(t, edi.DocumentDirectionInbound, sample.Direction)
	require.NotEmpty(t, sample.Checks)
	for _, check := range sample.Checks {
		assert.True(t, check.Matches, "%s: the guide says %q, the simulator sends %q",
			check.Field, check.Expected, check.Found)
	}
}
