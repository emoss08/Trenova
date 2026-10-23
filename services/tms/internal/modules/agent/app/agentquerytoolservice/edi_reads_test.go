package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The partner get reads the Desk's partner card by the id it names, and
// refuses a name in its place before anything is read.
func TestEDIPartner_ReadsTheCardByThePartnersID(t *testing.T) {
	t.Parallel()

	partner := pulid.MustNew("edip_")
	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskEdiPartnerCard": {Data: map[string]any{"ediPartnerCard": map[string]any{
			"partner":   map[string]any{"id": partner.String(), "code": "ACME"},
			"readiness": map[string]any{"ready": false, "items": []any{}},
		}}},
	}}
	tool := ediTool(t, "edi_get_partner", reader)

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"partnerId": "Acme Foods",
	}))
	require.Error(t, err)
	assert.Empty(t, reader.requests)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{"partnerId": partner.String()}))
	require.NoError(t, err)
	sent := reader.sent("DeskEdiPartnerCard")
	require.NotNil(t, sent)
	assert.Equal(t, partner.String(), sent.Variables["id"])
	record := data.(map[string]any)
	assert.Contains(t, record, "readiness")
	assert.NotContains(t, record["readiness"], "items", "an empty list tells a model nothing")
}

// A partner's answer counts every acknowledgment that came back, a
// rejection included, out of those that were expected.
func TestAckAnswer_SaysWhatCameBackOutOfWhatWasSent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "nothing sent in this window expects an acknowledgment",
		ackAnswer(map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusNotExpected: 4,
		}))
	assert.Equal(t,
		"3 of 5 came back with a 997 or 999 (2 accepted, 1 rejected); 1 still awaiting one; "+
			"1 marked Failed",
		ackAnswer(map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusAccepted: 2,
			edi.MessageAcknowledgmentStatusRejected: 1,
			edi.MessageAcknowledgmentStatusPending:  1,
			edi.MessageAcknowledgmentStatusFailed:   1,
		}))
}

// A transfer list reads Inbound unless a call names a direction, so the
// schema does not require one.
func TestEDITransfers_DefaultTheDirectionTheRootRequires(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskEdiTransfers": {Data: connectionPage("ediTransfers",
			map[string]any{"id": "edilt_1", "status": "PendingApproval"})},
	}}
	tool := ediTool(t, "edi_list_transfers", reader)
	assert.NotContains(t, tool.ParamSchema(), "required")

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)
	sent := reader.sent("DeskEdiTransfers")
	require.NotNil(t, sent)
	assert.Equal(t, transferInbound, sent.Variables[paramDirection])
}
