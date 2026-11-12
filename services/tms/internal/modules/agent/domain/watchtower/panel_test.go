package watchtower

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePanel_KeepsOnlyTheViewItsKindNames(t *testing.T) {
	t.Parallel()

	panel := NormalizePanel(&Panel{
		Kind:  PanelStop,
		Stop:  &StopPanel{Role: StopRoleDelivery, Verdict: StopMissed, Place: "  Dock 4  "},
		Clock: &ClockPanel{Place: "Somewhere else"},
	})

	require.NotNil(t, panel)
	assert.Equal(t, "Dock 4", panel.Stop.Place)
	assert.Nil(t, panel.Clock)
}

func TestNormalizePanel_IsNothingWithoutTheViewItsKindNames(t *testing.T) {
	t.Parallel()

	assert.Nil(t, NormalizePanel(nil))
	assert.Nil(t, NormalizePanel(&Panel{Kind: PanelClock, Stop: &StopPanel{Place: "Dock 4"}}))
	assert.Nil(t, NormalizePanel(&Panel{Kind: PanelKind("Unknown"), Stop: &StopPanel{}}))
	assert.Nil(t, NormalizePanel(&Panel{Kind: PanelPaper, Paper: &PaperPanel{}}))
	assert.Nil(t, NormalizePanel(&Panel{Kind: PanelExpiry, Expiry: &ExpiryPanel{Who: "Jane"}}))
}

func TestNormalizePanel_ShowsNoRecordIDAndBoundsWhatItHolds(t *testing.T) {
	t.Parallel()

	papers := make([]Paper, 0, maxPanelPapers+3)
	for range maxPanelPapers + 3 {
		papers = append(papers, Paper{Name: "Proof of delivery"})
	}
	panel := NormalizePanel(&Panel{
		Kind: PanelPaper,
		Paper: &PaperPanel{
			Customer: "Acme shp_01M717A2F8BF7TD40E2KW8V12H",
			Papers:   append(papers, Paper{Name: "   "}),
			Holds:    strings.Repeat("x", maxPanelText+40),
		},
	})

	require.NotNil(t, panel)
	assert.NotContains(t, panel.Paper.Customer, "shp_")
	assert.Len(t, panel.Paper.Papers, maxPanelPapers)
	assert.LessOrEqual(t, len([]rune(panel.Paper.Holds)), maxPanelText)
}

func TestSamePanel_TellsAChangedViewFromTheSameOne(t *testing.T) {
	t.Parallel()

	a := &Panel{Kind: PanelClock, Clock: &ClockPanel{Place: "Dock", BillableMinutes: 10}}
	b := &Panel{Kind: PanelClock, Clock: &ClockPanel{Place: "Dock", BillableMinutes: 10}}
	c := &Panel{Kind: PanelClock, Clock: &ClockPanel{Place: "Dock", BillableMinutes: 25}}

	assert.True(t, SamePanel(a, b))
	assert.False(t, SamePanel(a, c))
	assert.True(t, SamePanel(nil, nil))
	assert.False(t, SamePanel(a, nil))
}
