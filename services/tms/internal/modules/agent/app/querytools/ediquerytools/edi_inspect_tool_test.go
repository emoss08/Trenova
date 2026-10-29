package ediquerytools

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingInspector struct {
	requests []*ediservice.InspectX12Request
}

func (r *recordingInspector) InspectX12(
	ctx context.Context,
	req *ediservice.InspectX12Request,
) (*edix12inspect.InspectX12Result, error) {
	r.requests = append(r.requests, req)

	return (&ediservice.Service{}).InspectX12(ctx, req)
}

const pastedShipmentStatus = "ISA*00*          *00*          *ZZ*ACMERETAIL     *ZZ*TRENOVA        " +
	"*261014*0930*U*00401*000000417*0*P*>~" +
	"GS*QM*ACMERETAIL*TRENOVA*20261014*0930*417*X*004010~" +
	"ST*214*0001~" +
	"B10*PRO-88*BOL-88*ACMT~" +
	"AT7*X3*NS***20261014*0915~" +
	"SE*9*0001~" +
	"GE*1*417~" +
	"IEA*1*000000417~"

func TestInspectX12_ReadsPastedX12WithTheInboundFileCardsOverview(t *testing.T) {
	t.Parallel()

	inspector := &recordingInspector{}
	tool := newInspectX12Tool(inspector)

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		paramX12Content: pastedShipmentStatus,
	}, ""))
	require.NoError(t, err)

	require.Len(t, inspector.requests, 1)
	assert.Equal(t, pastedShipmentStatus, inspector.requests[0].RawX12)

	view := result.(*inspectX12View)
	expected, err := edix12inspect.InspectX12(t.Context(), &edix12inspect.InspectX12Request{
		RawX12: pastedShipmentStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, expected.FileInspection(), view.FileInspection,
		"the overview the inbound file card draws, unchanged")
	assert.Equal(t, "000000417", view.Envelope.ISAControlNumber)
	require.Len(t, view.Transactions, 1)
	assert.Equal(t, "214", view.Transactions[0].TransactionSet)
	assert.Equal(t, 8, view.Summary.SegmentCount, "ISA through IEA")
	assert.True(t, view.Interchange)
	assert.Contains(t, view.Note, "find it among stored messages only when the person asked "+
		"you to", "a pasted file is answered from what was pasted")

	var seCount *edix12inspect.NormalizedDiagnostic
	for idx := range view.Diagnostics {
		if view.Diagnostics[idx].SegmentID == "SE" {
			seCount = &view.Diagnostics[idx]
		}
	}
	require.NotNil(t, seCount, "SE says 9 segments where the set has 4")
	assert.Equal(t, edidocument.ValidationSeverityError, seCount.Severity)
	assert.Positive(t, seCount.SegmentIndex)
	assert.Contains(t, view.Note, "trading partner")
}

func TestInspectX12_RefusesContentPastTheCapAndSaysWhy(t *testing.T) {
	t.Parallel()

	inspector := &recordingInspector{}
	tool := newInspectX12Tool(inspector)
	oversized := strings.Repeat("N1*ST*STORE~", maxInspectX12Bytes/12+1)

	_, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		paramX12Content: oversized,
	}, ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "64 KB")
	assert.Contains(t, err.Error(), "edi_get_inbound_file")
	assert.Empty(t, inspector.requests, "nothing past the cap is parsed")

	_, err = tool.Query(
		t.Context(),
		querytooltest.AgentParams(map[string]any{paramX12Content: "  "}, ""),
	)
	require.Error(t, err)
	assert.Empty(t, inspector.requests)
}

func TestInspectX12_KeepsALargeInspectionToBounds(t *testing.T) {
	t.Parallel()

	var raw strings.Builder
	raw.WriteString("ISA*00*          *00*          *ZZ*ACMERETAIL     *ZZ*TRENOVA        " +
		"*261014*0930*U*00401*000000418*0*P*>~GS*QM*ACMERETAIL*TRENOVA*20261014*0930*418*X*004010~")
	for range maxInspectRows + 5 {
		raw.WriteString("ST*214*0001~B10*PRO*BOL*ACMT~SE*9*0002~")
	}
	raw.WriteString("GE*1*418~IEA*1*000000418~")

	result, err := newInspectX12Tool(
		&recordingInspector{},
	).Query(t.Context(), querytooltest.AgentParams(
		map[string]any{paramX12Content: raw.String()}, "",
	))
	require.NoError(t, err)

	view := result.(*inspectX12View)
	assert.Len(t, view.Transactions, maxInspectRows)
	assert.Equal(t, 5, view.TransactionsOmitted)
	assert.LessOrEqual(t, len(view.Diagnostics), maxInspectRows)
	assert.Positive(t, view.DiagnosticsOmitted)
	assert.Equal(t, maxInspectRows+5, view.Summary.TransactionCount,
		"the summary counts everything, the lists are cut")
}

func TestInspectX12_IsAReadThatHoldsEveryLaterWrite(t *testing.T) {
	t.Parallel()

	policy := newInspectX12Tool(&recordingInspector{}).Policy()
	assert.Equal(t, agenttooltrust.ToolKindQuery, policy.Kind)
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agenttooltrust.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceEDI, policy.Source)
	assert.True(t, policy.CapsLaterWrites)
}
