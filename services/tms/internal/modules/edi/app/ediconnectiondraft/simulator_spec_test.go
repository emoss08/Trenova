package ediconnectiondraft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func simulatorGuide(t *testing.T) ediconnectiondraftports.SpecDocument {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	path := filepath.Join(dir, "..", "..", "..", "..", "..", "..", "..",
		"dev", "edi-partner-sim", "specs", "simpartner-implementation-guide.txt")
	text, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err, "the partner simulator's guide is the fixture for this test")
	return ediconnectiondraftports.SpecDocument{
		ID:       pulid.MustNew("doc_"),
		FileName: "simpartner-implementation-guide.txt",
		Pages:    []ediconnectiondraftports.SpecPage{{Number: 1, Text: string(text)}},
	}
}

func simulatorValues() []ediconnectiondraftports.DraftValueInput {
	sim := func(field, v, quote string) ediconnectiondraftports.DraftValueInput {
		return value(field, v, 1, quote)
	}
	return []ediconnectiondraftports.DraftValueInput{
		sim(fieldPartnerCode, "SIMPARTNER", "Trading Partner Code: SIMPARTNER"),
		sim(
			fieldPartnerName,
			"Trenova Partner Simulator",
			"Trading Partner Name: Trenova Partner Simulator",
		),
		sim(fieldPartnerContactName, "Simulator EDI desk", "EDI Contact: Simulator EDI desk"),
		sim(
			fieldPartnerContactEmail,
			"sim-edi@partner-sim.example",
			"Email: sim-edi@partner-sim.example",
		),
		sim(fieldPartnerContactPhone, "+1 312 555 0190", "Phone: +1 312 555 0190"),
		sim(fieldPartnerTimezone, "Etc/UTC", "UTC (Etc/UTC)"),
		sim(fieldTransportMethod, "AS2", "The simulator exchanges documents over AS2."),
		sim("as2.partnerAs2Id", "SIMPARTNER", "Simulator AS2 Identifier: SIMPARTNER"),
		sim("as2.localAs2Id", "TRENOVA", "Carrier AS2 Identifier: TRENOVA"),
		sim("as2.endpointUrl", "http://localhost:9210/as2", "AS2 URL: http://localhost:9210/as2"),
		sim("as2.mdnMode", "sync", "MDN: synchronous, signed."),
		sim("as2.signingAlgorithm", "sha256", "Signing algorithm: SHA-256."),
		sim("as2.encryptionAlgorithm", "aes256-cbc", "Encryption algorithm: AES-256 (CBC)."),
		sim(
			fieldPartnerQualifier,
			"ZZ",
			"Simulator interchange sender: qualifier ZZ (mutually defined), ID SIMPARTNER",
		),
		sim(
			fieldPartnerID,
			"SIMPARTNER",
			"Simulator interchange sender: qualifier ZZ (mutually defined), ID SIMPARTNER",
		),
		sim(
			fieldOurQualifier,
			"ZZ",
			"Carrier interchange receiver: qualifier ZZ (mutually defined), ID TRENOVA",
		),
		sim(
			fieldOurID,
			"TRENOVA",
			"Carrier interchange receiver: qualifier ZZ (mutually defined), ID TRENOVA",
		),
		sim(fieldPartnerGSID, "SIMPARTNER", "GS02 Application Sender Code: SIMPARTNER"),
		sim(fieldOurGSID, "TRENOVA", "GS03 Application Receiver Code: TRENOVA"),
		sim(fieldX12Version, "004010", "GS08 Version: 004010"),
		sim(fieldUsageIndicator, "T", "ISA15 usage indicator: T."),
		sim(fieldElementSep, "*", "Element separator: * (asterisk)."),
		sim(fieldSegmentSep, "~", "Segment terminator: ~ (tilde)."),
		sim(fieldComponentSep, ">", "Sub-element separator: > (greater than)."),
		sim(controlNumberField(edidocument.ControlNumberKindInterchange), "1",
			"Carrier interchange control numbers (ISA13) start at 1"),
		sim(
			setField(
				edidocument.TransactionSet204,
				edidocument.DocumentDirectionInbound,
				setAttrSupported,
			),
			"true",
			"The simulator sends the 204 Motor Carrier Load Tender to the carrier.",
		),
		sim(
			setField(
				edidocument.TransactionSet990,
				edidocument.DocumentDirectionOutbound,
				setAttrSupported,
			),
			"true",
			"The carrier answers each tender with a 990 Response to a Load Tender.",
		),
		sim(
			setField(
				edidocument.TransactionSet214,
				edidocument.DocumentDirectionOutbound,
				setAttrSupported,
			),
			"true",
			"The carrier sends the 214 Transportation Carrier Shipment Status Message",
		),
		sim(
			setField(
				edidocument.TransactionSet990,
				edidocument.DocumentDirectionOutbound,
				setAttrAck,
			),
			"997",
			"The simulator returns a 997 Functional Acknowledgment for every transaction set it receives.",
		),
		sim(
			setField(
				edidocument.TransactionSet214,
				edidocument.DocumentDirectionOutbound,
				setAttrAck,
			),
			"997",
			"The simulator returns a 997 Functional Acknowledgment for every transaction set it receives.",
		),
	}
}

func TestPlan_ThePartnerSimulatorsGuideDraftsAConnectionToTheSimulator(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	draft, err := h.service.PlanConnectionDraft(
		t.Context(),
		&ediconnectiondraftports.ConnectionDraftRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Document: simulatorGuide(t),
			Values:   simulatorValues(),
			Requirements: []ediconnectiondraftports.DraftRequirementInput{{
				TransactionSet: edidocument.TransactionSet204, Direction: edidocument.DocumentDirectionInbound,
				SegmentID: "L3", Usage: ediconnectiondraftports.RequirementUsageMandatory,
				Citation: cite(1, "L3 Total Weight and Charges is mandatory."),
			}},
		},
	)
	require.NoError(t, err)

	assert.Empty(t, draft.Decisions)
	assert.Empty(t, draft.Failures)
	assert.True(t, draft.Ready, draft.Blockers)
	require.NotNil(t, draft.Transport)
	assert.Equal(
		t,
		"http://localhost:9210/as2",
		valueOf(t, draft.Transport.Fields, "as2.endpointUrl").Value,
	)
	assert.Len(t, draft.Sets, 3)
	require.NotNil(t, draft.Project)
	assert.Zero(t, draft.Project.Errors, draft.Project.Diagnostics)

	require.Len(t, draft.Samples, 1, "the guide carries the tender the simulator sends")
	sample := draft.Samples[0]
	require.True(t, sample.Parsed, sample.Error)
	assert.Equal(t, edidocument.TransactionSet204, sample.TransactionSet)
	assert.Equal(t, edidocument.DocumentDirectionInbound, sample.Direction)
	require.NotEmpty(t, sample.Checks)
	for _, check := range sample.Checks {
		assert.True(t, check.Matches, "%s: the guide says %q, the simulator sends %q",
			check.Field, check.Expected, check.Found)
	}
}
