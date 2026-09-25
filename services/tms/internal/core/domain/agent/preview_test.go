package agent

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePreview() *ProposalPreview {
	version := int64(7)

	return &ProposalPreview{
		Schema:        PreviewSchemaVersion,
		ProposalID:    pulid.MustNew("ap_"),
		Tool:          "cancel_shipment",
		Summary:       "Would cancel PRO-100.",
		Coverage:      PreviewCoverageFull,
		TargetVersion: &version,
		ComputedAt:    1767225600,
		Changes: []RecordChange{{
			Resource:  permission.ResourceShipment,
			EntityID:  pulid.MustNew("shp_"),
			Label:     "PRO-100",
			Operation: PreviewOperationUpdate,
			Version:   &version,
			Fields: []PreviewFieldChange{
				{Path: "status", Label: "Status", Before: "New", After: "Cancelled"},
				{Path: "ageHours", Label: "Age hours", Before: 3.0, After: 3.0, Volatile: true},
			},
		}},
	}
}

func TestComputeDigest_IsStableAcrossWhatDoesNotMatter(t *testing.T) {
	t.Parallel()

	params := map[string]any{"shipmentId": "shp_1", "reason": "pulled"}
	preview := samplePreview()
	first, err := preview.ComputeDigest(params)
	require.NoError(t, err)
	assert.Len(t, first, 64)

	later := *preview
	later.ComputedAt = 1767229200
	later.Digest = first
	later.Recorded = true
	later.Warnings = []PreviewWarning{{Code: PreviewWarningUnpinned, Message: "unpinned"}}
	later.Changes = append([]RecordChange(nil), preview.Changes...)
	later.Changes[0].Fields = append([]PreviewFieldChange(nil), preview.Changes[0].Fields...)
	later.Changes[0].Fields[1].After = 4.0
	again, err := later.ComputeDigest(map[string]any{"reason": "pulled", "shipmentId": "shp_1"})
	require.NoError(t, err)

	assert.Equal(t, first, again,
		"the instant, the digest, warnings and volatile values are not what was approved")
}

func TestComputeDigest_ChangesWithWhatWasShown(t *testing.T) {
	t.Parallel()

	params := map[string]any{"shipmentId": "shp_1"}
	preview := samplePreview()
	base, err := preview.ComputeDigest(params)
	require.NoError(t, err)

	moved := samplePreview()
	moved.ProposalID = preview.ProposalID
	moved.Changes[0].EntityID = preview.Changes[0].EntityID
	moved.Changes[0].Fields[0].After = "Hold"
	changed, err := moved.ComputeDigest(params)
	require.NoError(t, err)
	assert.NotEqual(t, base, changed)

	otherParams, err := preview.ComputeDigest(map[string]any{"shipmentId": "shp_2"})
	require.NoError(t, err)
	assert.NotEqual(t, base, otherParams)

	withheld := *preview
	withheld.WithheldCount = 1
	fewer, err := withheld.ComputeDigest(params)
	require.NoError(t, err)
	assert.NotEqual(t, base, fewer)

	newer := *preview
	next := int64(8)
	newer.TargetVersion = &next
	versioned, err := newer.ComputeDigest(params)
	require.NoError(t, err)
	assert.NotEqual(t, base, versioned, "versions are part of what was approved")
}

func TestPlanPreviewDigest_CoversTheStepsInOrder(t *testing.T) {
	t.Parallel()

	plan := pulid.MustNew("apl_")
	forward, err := PlanPreviewDigest(plan, []string{"a", "b"})
	require.NoError(t, err)
	reversed, err := PlanPreviewDigest(plan, []string{"b", "a"})
	require.NoError(t, err)
	again, err := PlanPreviewDigest(plan, []string{"a", "b"})
	require.NoError(t, err)

	assert.Equal(t, forward, again)
	assert.NotEqual(t, forward, reversed)
}

func TestPreviewStaleness(t *testing.T) {
	t.Parallel()

	var none *PreviewStaleness
	assert.False(t, none.Stale())
	assert.False(t, (&PreviewStaleness{Pinned: false, Missing: true}).Stale())
	assert.False(t, (&PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: 3}).Stale())
	assert.True(t, (&PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: 4}).Stale())
	assert.True(t, (&PreviewStaleness{Pinned: true, ProposedVersion: 3, Missing: true}).Stale())
}

func TestToolPreviewSimulation_ReadsAsTheRuntimesTranscript(t *testing.T) {
	t.Parallel()

	preview := &ToolPreview{
		Summary: "Would cancel PRO-100 and tell Acme.",
		Changes: []RecordChange{
			{
				Label:     "PRO-100",
				Operation: PreviewOperationUpdate,
				Fields: []PreviewFieldChange{
					{Path: "status", Label: "Status", Before: "New", After: "Cancelled"},
					{
						Path:      "customerId",
						Label:     "Customer",
						Before:    "cus_1",
						After:     "cus_2",
						AfterRef:  &PreviewRef{Label: "Other Co"},
						BeforeRef: &PreviewRef{Label: "Acme"},
					},
					{Path: "notes", Label: "Notes", Before: "x", After: nil},
					{Path: "secret", Label: "Secret", After: "y", Withheld: true},
				},
			},
			{
				Label:     "Acme",
				Operation: PreviewOperationSend,
				Message: &MessagePreview{
					Channel: MessageChannelEmail,
					To:      []string{"ap@acme.test"},
					Subject: "Cancelled",
				},
				Money: &MoneyPreview{
					Lines: []MoneyLine{{
						Label: "Linehaul",
						After: decimal.NewNullDecimal(decimal.RequireFromString("10")),
					}},
				},
			},
		},
	}

	simulation := preview.Simulation()

	assert.True(t, simulation.Previewed)
	described := simulation.Describe()
	assert.Contains(t, described, "- PRO-100: Status: New → Cancelled")
	assert.Contains(t, described, "- PRO-100: Customer: Acme → Other Co")
	assert.Contains(t, described, "- PRO-100: Notes: x → nothing")
	assert.NotContains(t, described, "Secret")
	assert.Contains(t, described, "- Acme: operation: send")
	assert.Contains(t, described, "- Acme: to: ap@acme.test")
	assert.Contains(t, described, "- Acme: Linehaul: 10")
}

func TestToolPreviewSimulation_StaysWithinItsBound(t *testing.T) {
	t.Parallel()

	fields := make([]PreviewFieldChange, 0, 400)
	for range 400 {
		fields = append(fields, PreviewFieldChange{
			Label: "Field", Before: strings.Repeat("a", 100), After: strings.Repeat("b", 100),
		})
	}
	simulation := (&ToolPreview{Summary: "Big", Changes: []RecordChange{{Fields: fields}}}).Simulation()

	assert.LessOrEqual(t, EncodedPreviewSize(simulation), MaxPreviewSimulationBytes+1024)
	assert.Less(t, len(simulation.Changes), 400)
}

func TestProposalPreviewWarnings_AreKeptOncePerCodeAndArguments(t *testing.T) {
	t.Parallel()

	preview := &ProposalPreview{}
	preview.AddWarning(PreviewWarning{Code: PreviewWarningDependsOnStep, Args: []string{"1"}})
	preview.AddWarning(PreviewWarning{Code: PreviewWarningDependsOnStep, Args: []string{"1"}})
	preview.AddWarning(PreviewWarning{Code: PreviewWarningDependsOnStep, Args: []string{"2"}})

	assert.Len(t, preview.Warnings, 2)
	assert.True(t, preview.HasWarning(PreviewWarningDependsOnStep))
	assert.False(t, preview.HasWarning(PreviewWarningWouldFail))
}

func TestBoundPreviewValue(t *testing.T) {
	t.Parallel()

	short, cut := BoundPreviewValue("short")
	assert.Equal(t, "short", short)
	assert.False(t, cut)

	long, cut := BoundPreviewValue(strings.Repeat("é", MaxPreviewValueBytes))
	assert.True(t, cut)
	assert.LessOrEqual(t, len(long.(string)), MaxPreviewValueBytes)
	assert.True(t, strings.HasSuffix(long.(string), "…"))

	list := make([]any, 0, 1000)
	for range 1000 {
		list = append(list, "entry")
	}
	encoded, cut := BoundPreviewValue(list)
	assert.True(t, cut)
	assert.IsType(t, "", encoded)
}
