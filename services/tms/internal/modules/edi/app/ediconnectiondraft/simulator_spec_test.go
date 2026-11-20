package ediconnectiondraft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func manufacturerGuide(t *testing.T) ediconnectiondraftports.SpecDocument {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	path := filepath.Join(dir, "..", "..", "..", "..", "..", "..", "..",
		"dev", "edi-partner-sim", "specs", "manufacturer-implementation-guide.txt")
	text, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err, "the simulator's manufacturer guide is the fixture for this test")
	return ediconnectiondraftports.SpecDocument{
		ID:       pulid.MustNew("doc_"),
		FileName: "manufacturer-implementation-guide.txt",
		Pages:    []ediconnectiondraftports.SpecPage{{Number: 1, Text: string(text)}},
	}
}

func manufacturerValues() []ediconnectiondraftports.DraftValueInput {
	guide := func(field, v, quote string) ediconnectiondraftports.DraftValueInput {
		return value(field, v, 1, quote)
	}
	const (
		sender   = "Kestrel interchange sender: qualifier ZZ (mutually defined), ID KESTRELVW"
		receiver = "Carrier interchange receiver: qualifier ZZ (mutually defined), ID TRENOVA"
		ack      = "Kestrel returns a 997 Functional Acknowledgment for every transaction set it receives."
		within   = "Kestrel acknowledges within 45 seconds and rejects (AK5 R) any transaction that breaks this guide."
	)
	set := func(
		transactionSet edidocument.TransactionSet,
		direction edidocument.DocumentDirection,
		attr string,
	) string {
		return ediconnectionfield.SetField(transactionSet, direction, attr)
	}
	inbound := edidocument.DocumentDirectionInbound
	outbound := edidocument.DocumentDirectionOutbound
	return []ediconnectiondraftports.DraftValueInput{
		guide(ediconnectionfield.PartnerCode, "KESTRELVW", "Trading Partner Code: KESTRELVW"),
		guide(
			ediconnectionfield.PartnerName,
			"Kestrel Valve Works",
			"Trading Partner Name: Kestrel Valve Works",
		),
		guide(
			ediconnectionfield.PartnerContactName,
			"Kestrel transportation EDI desk",
			"EDI Contact: Kestrel transportation EDI desk",
		),
		guide(
			ediconnectionfield.PartnerContactEmail,
			"edi@kestrel-valve.example",
			"Email: edi@kestrel-valve.example",
		),
		guide(ediconnectionfield.PartnerContactPhone, "+1 815 555 0142", "Phone: +1 815 555 0142"),
		guide(
			ediconnectionfield.PartnerTimezone,
			"America/Chicago",
			"US Central time (America/Chicago)",
		),
		guide(
			ediconnectionfield.TransportMethod,
			"AS2",
			"Kestrel exchanges documents over AS2.",
		),
		guide("as2.partnerAs2Id", "KESTRELVW", "Kestrel AS2 Identifier: KESTRELVW"),
		guide("as2.localAs2Id", "TRENOVA", "Carrier AS2 Identifier: TRENOVA"),
		guide(
			"as2.endpointUrl",
			"http://localhost:9210/as2/manufacturer",
			"AS2 URL: http://localhost:9210/as2/manufacturer",
		),
		guide("as2.mdnMode", "sync", "MDN: synchronous, signed."),
		guide("as2.signingAlgorithm", "sha256", "Signing algorithm: SHA-256."),
		guide("as2.encryptionAlgorithm", "aes256-cbc", "Encryption algorithm: AES-256 (CBC)."),
		guide(ediconnectionfield.PartnerQualifier, "ZZ", sender),
		guide(ediconnectionfield.PartnerID, "KESTRELVW", sender),
		guide(ediconnectionfield.OurQualifier, "ZZ", receiver),
		guide(ediconnectionfield.OurID, "TRENOVA", receiver),
		guide(
			ediconnectionfield.PartnerGSID,
			"KESTRELVW",
			"GS02 Application Sender Code: KESTRELVW",
		),
		guide(ediconnectionfield.OurGSID, "TRENOVA", "GS03 Application Receiver Code: TRENOVA"),
		guide(ediconnectionfield.X12Version, "004010", "GS08 Version: 004010"),
		guide(ediconnectionfield.UsageIndicator, "T", "ISA15 usage indicator: T."),
		guide(ediconnectionfield.ElementSep, "*", "Element separator: * (asterisk)."),
		guide(ediconnectionfield.SegmentSep, "~", "Segment terminator: ~ (tilde)."),
		guide(ediconnectionfield.ComponentSep, ">", "Sub-element separator: > (greater than)."),
		guide(
			ediconnectionfield.LineEnding,
			"line feed",
			"Kestrel writes a line feed after every segment terminator.",
		),
		guide(ediconnectionfield.ControlNumberField(edidocument.ControlNumberKindInterchange), "1",
			"Carrier interchange control numbers (ISA13) start at 1"),
		guide(
			set(edidocument.TransactionSet204, inbound, ediconnectionfield.SetAttrSupported),
			"true",
			"Kestrel sends the 204 Motor Carrier Load Tender to the carrier.",
		),
		guide(
			set(edidocument.TransactionSet990, outbound, ediconnectionfield.SetAttrSupported),
			"true",
			"The carrier answers each tender with a 990 Response to a Load Tender.",
		),
		guide(
			set(edidocument.TransactionSet214, outbound, ediconnectionfield.SetAttrSupported),
			"true",
			"The carrier sends the 214 Transportation Carrier Shipment Status Message",
		),
		guide(
			set(edidocument.TransactionSet210, outbound, ediconnectionfield.SetAttrSupported),
			"true",
			"The carrier bills each load with a 210 Motor Carrier Freight Details and Invoice.",
		),
		guide(
			set(edidocument.TransactionSet990, outbound, ediconnectionfield.SetAttrAck),
			"997",
			ack,
		),
		guide(
			set(edidocument.TransactionSet214, outbound, ediconnectionfield.SetAttrAck),
			"997",
			ack,
		),
		guide(
			set(edidocument.TransactionSet210, outbound, ediconnectionfield.SetAttrAck),
			"997",
			ack,
		),
		guide(
			set(edidocument.TransactionSet990, outbound, ediconnectionfield.SetAttrAckWindow),
			"45 seconds",
			within,
		),
		guide(
			set(edidocument.TransactionSet214, outbound, ediconnectionfield.SetAttrAckWindow),
			"45 seconds",
			within,
		),
		guide(
			set(edidocument.TransactionSet210, outbound, ediconnectionfield.SetAttrAckWindow),
			"45 seconds",
			within,
		),
	}
}

func TestPlan_TheSimulatorsManufacturerGuideDraftsAConnectionToKestrel(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	draft, err := h.service.PlanConnectionDraft(
		t.Context(),
		&ediconnectiondraftports.ConnectionDraftRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Document: manufacturerGuide(t),
			Values:   manufacturerValues(),
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
		"http://localhost:9210/as2/manufacturer",
		valueOf(t, draft.Transport.Fields, "as2.endpointUrl").Value,
	)
	lineEnding := valueOf(t, draft.Interchange, ediconnectionfield.LineEnding)
	assert.Equal(t, ediconnectionfield.LineEndingLF, lineEnding.Value)
	assert.Equal(t, ediconnectiondraftports.ValueStatusAccepted, lineEnding.Status)
	assert.Equal(t, 1, lineEnding.Source.Page)

	assert.Len(t, draft.Sets, 4)
	for _, transactionSet := range []edidocument.TransactionSet{
		edidocument.TransactionSet990,
		edidocument.TransactionSet214,
		edidocument.TransactionSet210,
	} {
		sent := setFor(t, draft, transactionSet, edidocument.DocumentDirectionOutbound)
		window := valueOf(t, sent.Fields, ediconnectionfield.SetField(
			transactionSet,
			edidocument.DocumentDirectionOutbound,
			ediconnectionfield.SetAttrAckWindow,
		))
		assert.Equal(t, "45 seconds", window.Value, transactionSet)
		assert.Equal(t, ediconnectiondraftports.ValueStatusAccepted, window.Status, transactionSet)
		assert.Equal(t, 1, window.Source.Page, transactionSet)
	}
	require.NotNil(t, draft.Project)
	assert.Zero(t, draft.Project.Errors, draft.Project.Diagnostics)

	require.Len(t, draft.Samples, 1, "the guide carries the tender Kestrel sends")
	sample := draft.Samples[0]
	require.True(t, sample.Parsed, sample.Error)
	assert.Equal(t, edidocument.TransactionSet204, sample.TransactionSet)
	assert.Equal(t, edidocument.DocumentDirectionInbound, sample.Direction)
	require.NotEmpty(t, sample.Checks)
	for _, check := range sample.Checks {
		assert.True(t, check.Matches, "%s: the guide says %q, Kestrel sends %q",
			check.Field, check.Expected, check.Found)
	}
}
