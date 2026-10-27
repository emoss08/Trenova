package x12_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tender = "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        " +
	"*260107*1200*^*00401*000000101*0*T*>~" +
	"GS*SM*SIMPARTNER*TRENOVA*20260107*1200*101*X*004010~" +
	"ST*204*0001~B2**SIML**SHIP-1**PP~B2A*00~S5*1*CL~N1**Shipper~N4*Chicago*IL*60601~SE*7*0001~" +
	"ST*204*0002~B2**SIML**SHIP-2**PP~B2A*00~S5*1*CL~SE*5*0002~" +
	"GE*2*101~IEA*1*000000101~"

func newRuntime(t *testing.T) *x12.Runtime {
	t.Helper()
	r := x12.New(wasmmodule.Options{PoolSize: 1})
	t.Cleanup(func() {
		assert.NoError(t, r.Close(context.WithoutCancel(t.Context())))
	})
	return r
}

func TestEmbeddedModuleMatchesItsChecksum(t *testing.T) {
	t.Parallel()

	module, err := os.ReadFile("edi.wasm")
	require.NoError(t, err)
	sum := sha256.Sum256(module)
	sidecar, err := os.ReadFile("edi.wasm.sha256")
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(sum[:])+"  edi.wasm\n", string(sidecar))
}

func TestReadingRejectsTheBadSetAndPointsAtTheElement(t *testing.T) {
	t.Parallel()

	reading, err := newRuntime(t).Read(t.Context(), &x12.ReadRequest{Text: tender, IncludeSegments: true})
	require.NoError(t, err)
	require.Len(t, reading.Interchanges, 1)
	group := reading.Interchanges[0].Groups[0]
	assert.Equal(t, "P", group.Acknowledgment.Code)
	require.Len(t, group.Transactions, 2)

	rejected := group.Transactions[0]
	assert.Equal(t, x12.TransactionRejected, rejected.Status)
	assert.Equal(t, "x12.004010.204", rejected.Guide)
	require.Len(t, rejected.Findings, 1)
	finding := rejected.Findings[0]
	assert.Equal(t, "mandatoryElementMissing", finding.Kind)
	assert.Equal(t, "N1", finding.SegmentID)
	assert.Equal(t, "0310", finding.LoopID)
	require.NotNil(t, finding.Position)
	assert.Equal(t, 5, *finding.Position)
	assert.Equal(t, "N101 (Entity Identifier Code) is mandatory but is empty", finding.Message.String())

	assert.Equal(t, x12.TransactionAccepted, group.Transactions[1].Status)
	assert.Equal(t, "0300/0310", reading.Segments[6].LoopPath)
	assert.Equal(t, 1, reading.Summary.Rejected)
}

func TestAcknowledgingAGroupNestsItsErrors(t *testing.T) {
	t.Parallel()

	runtime := newRuntime(t)
	reading, err := runtime.Read(t.Context(), &x12.ReadRequest{Text: tender})
	require.NoError(t, err)

	acknowledgments, err := runtime.Acknowledge(t.Context(), &x12.AcknowledgeRequest{
		Kind: x12.AcknowledgmentKind997,
		Acknowledgments: []x12.AcknowledgeInput{{
			Group: &reading.Interchanges[0].Groups[0],
			Envelope: x12.AcknowledgmentEnvelope{
				SenderQualifier:          "ZZ",
				SenderID:                 "TRENOVA",
				ReceiverQualifier:        "ZZ",
				ReceiverID:               "SIMPARTNER",
				ApplicationSender:        "TRENOVA",
				ApplicationReceiver:      "SIMPARTNER",
				Date:                     "20260107",
				Time:                     "1201",
				InterchangeControlNumber: "000000555",
				GroupControlNumber:       "555",
				TransactionControlNumber: "0001",
				Usage:                    "T",
			},
		}},
	})
	require.NoError(t, err)
	require.Len(t, acknowledgments, 1)
	assert.Equal(t, "P", acknowledgments[0].Code)
	assert.Contains(t,
		acknowledgments[0].Text,
		"AK1*SM*101~AK2*204*0001~AK3*N1*5*0310*8~AK4*1*98*1~AK5*R*5~AK2*204*0002~AK5*A~AK9*P*2*2*1~",
	)
}

func TestWritingPadsTheISAAndCountsTheTrailers(t *testing.T) {
	t.Parallel()

	written, err := newRuntime(t).Write(t.Context(), &x12.WriteRequest{
		Counts: true,
		Segments: [][]any{
			{"ST", "990", "0001"},
			{"B1", "SIML", "SHIP-1", "20260107", "A"},
			{"SE", "", "0001"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "ST*990*0001~B1*SIML*SHIP-1*20260107*A~SE*3*0001~", written.Text)
	assert.Equal(t, 3, written.SegmentCount)
}

func TestBaseGuidesAreListedAndChecked(t *testing.T) {
	t.Parallel()

	runtime := newRuntime(t)
	guides, err := runtime.Guides(t.Context())
	require.NoError(t, err)
	ids := make([]string, 0, len(guides))
	for _, guide := range guides {
		ids = append(ids, guide.ID)
	}
	assert.Equal(t, []string{
		"x12.004010.204", "x12.004010.210", "x12.004010.214",
		"x12.004010.990", "x12.004010.997", "x12.005010.999",
	}, ids)

	raw, err := runtime.Guide(t.Context(), "x12.004010.990")
	require.NoError(t, err)
	problems, err := runtime.CheckGuide(t.Context(), raw)
	require.NoError(t, err)
	assert.Empty(t, problems)

	var guide map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &guide))
	guide["derivedFrom"] = "x12.004010.990"
	guide["transactionSet"] = "99"
	widened, err := sonic.Marshal(guide)
	require.NoError(t, err)
	problems, err = runtime.CheckGuide(t.Context(), widened)
	require.NoError(t, err)
	require.NotEmpty(t, problems)
	assert.Equal(t, "transactionSet", problems[0].Path)
}

func TestAnUnknownBaseGuideIsARequestError(t *testing.T) {
	t.Parallel()

	_, err := newRuntime(t).Read(t.Context(), &x12.ReadRequest{
		Text:   tender,
		Guides: []x12.GuideChoice{{TransactionSet: "204", Builtin: "nope"}},
	})
	var requestErr *wasmmodule.RequestError
	require.ErrorAs(t, err, &requestErr)
	assert.Equal(t, wasmmodule.CodeInvalidInput, requestErr.Code)
	assert.True(t, strings.Contains(requestErr.Detail, "nope"))
}
