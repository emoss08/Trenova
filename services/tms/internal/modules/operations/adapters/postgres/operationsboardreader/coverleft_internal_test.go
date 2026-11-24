package operationsboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lapseDay = int64(24 * 60 * 60)

var lapseNow = time.Date(2026, time.November, 23, 9, 41, 0, 0, time.UTC).Unix()

type fakeCoverReader struct {
	covers      map[pulid.ID]*carriercoverports.Cover
	coversAsked *carriercoverports.CoversRequest
	ending      []*carriercoverports.Cover
	loads       map[pulid.ID][]carriercoverports.Load
	coversCalls int
	endingCalls int
	loadsCalls  int
	loadsAsked  *carriercoverports.LoadsRequest
	endingAsked *carriercoverports.EndingRequest
}

func (f *fakeCoverReader) Covers(
	_ context.Context,
	req *carriercoverports.CoversRequest,
) (map[pulid.ID]*carriercoverports.Cover, error) {
	f.coversCalls++
	f.coversAsked = req
	out := map[pulid.ID]*carriercoverports.Cover{}
	for _, id := range req.IDs {
		if cover, ok := f.covers[id]; ok {
			out[id] = cover
		}
	}

	return out, nil
}

func (f *fakeCoverReader) Ending(
	_ context.Context,
	req *carriercoverports.EndingRequest,
) ([]*carriercoverports.Cover, error) {
	f.endingCalls++
	f.endingAsked = req

	return f.ending, nil
}

func (f *fakeCoverReader) Loads(
	_ context.Context,
	req *carriercoverports.LoadsRequest,
) (map[pulid.ID][]carriercoverports.Load, error) {
	f.loadsCalls++
	f.loadsAsked = req

	return f.loads, nil
}

func lapseCover(
	name string,
	policyType carrier.InsurancePolicyType,
	endsIn int64,
	loads int64,
) *carriercoverports.Cover {
	return &carriercoverports.Cover{
		CarrierID:        pulid.MustNew("car_"),
		Name:             name,
		Status:           carrier.StatusActive,
		ComplianceStatus: carrier.ComplianceStatusQualified,
		Policy: &carrier.CarrierInsurancePolicy{
			PolicyType:     policyType,
			PolicyNumber:   "POL-" + name,
			ExpirationDate: lapseNow + endsIn*lapseDay,
		},
		LoadsRead:     true,
		OpenLoads:     loads,
		ReadableLoads: loads,
	}
}

func lapseLoad(cover *carriercoverports.Cover, pro string, inDays int64) carriercoverports.Load {
	return carriercoverports.Load{
		CarrierID:  cover.CarrierID,
		ShipmentID: pulid.MustNew("shp_"),
		ProNumber:  pro,
		DeliversAt: lapseNow + inDays*lapseDay,
	}
}

func TestCoverLeftReadsTheSoonestRequiredPolicyAgainstTheWarningWindow(t *testing.T) {
	t.Parallel()

	ridgeline := lapseCover("Ridgeline Freight", carrier.InsurancePolicyTypeAutoLiability, 6, 4)
	lapsed := lapseCover("Ironhorse", carrier.InsurancePolicyTypeCargoLiability, -3, 0)
	hidden := lapseCover("Blue Line", carrier.InsurancePolicyTypeCargoLiability, 18, 2)
	hidden.ReadableLoads = 1
	missing := lapseCover("Palmer", carrier.InsurancePolicyTypeAutoLiability, 40, 3)
	missing.Missing = []carrier.InsurancePolicyType{carrier.InsurancePolicyTypeCargoLiability}
	fake := &fakeCoverReader{covers: map[pulid.ID]*carriercoverports.Cover{
		ridgeline.CarrierID: ridgeline, lapsed.CarrierID: lapsed,
		hidden.CarrierID: hidden, missing.CarrierID: missing,
	}}
	reader := NewCoverLeft(CoverParams{Cover: fake})

	readings, err := reader.Readings(context.Background(), &boardreaderports.RecordRequest{
		IDs: []pulid.ID{
			ridgeline.CarrierID, lapsed.CarrierID, hidden.CarrierID, missing.CarrierID,
			pulid.MustNew("car_"),
		},
		Now:      lapseNow,
		Location: time.UTC,
		Place:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.coversCalls, "one read for the whole board")
	assert.True(t, fake.coversAsked.Loads)

	soon := readings[ridgeline.CarrierID]
	assert.Equal(t, boardcatalog.Days(6), soon.Value)
	assert.Equal(t, "{0} left", soon.Standing.Text)
	assert.Equal(t, "Auto liability ends {0}", soon.Fact.Text)
	assert.True(t, soon.Fact.Day)
	assert.Equal(t, ridgeline.Policy.ExpirationDate, soon.Fact.At)
	assert.Equal(t, "with 4 of our loads now", soon.Fact.Place)
	assert.Equal(t, boardcatalog.ToneWarn, soon.Tone)
	assert.Equal(t, &boardreading.WindowFacts{
		Left: 6 * lapseDay,
		Span: carrier.InsuranceWarningWindowSeconds,
	}, soon.Window)

	gone := readings[lapsed.CarrierID]
	assert.Equal(t, "{0} lapsed", gone.Standing.Text)
	assert.Equal(t, "Cargo liability ended {0}", gone.Fact.Text)
	assert.Equal(t, "with none of our loads now", gone.Fact.Place)
	assert.Equal(t, boardcatalog.ToneBad, gone.Tone)

	assert.Empty(t, readings[hidden.CarrierID].Fact.Place,
		"a count over loads the reader may not all see is withheld, not the visible part")
	_, drawn := readings[missing.CarrierID]
	assert.False(t, drawn, "a carrier missing a required policy has no cover to count down")
	assert.Len(t, readings, 3)
}

func TestCoverAgainstLoadsMarksTheLoadThatDeliversAfterCoverEnds(t *testing.T) {
	t.Parallel()

	ridgeline := lapseCover("Ridgeline Freight", carrier.InsurancePolicyTypeAutoLiability, 6, 2)
	copperhead := lapseCover(
		"Copperhead Transport",
		carrier.InsurancePolicyTypeCargoLiability,
		10,
		1,
	)
	sandoval := lapseCover("Sandoval Trucking", carrier.InsurancePolicyTypeAutoLiability, 27, 0)
	fake := &fakeCoverReader{
		ending: []*carriercoverports.Cover{ridgeline, copperhead, sandoval},
		loads: map[pulid.ID][]carriercoverports.Load{
			ridgeline.CarrierID: {
				lapseLoad(ridgeline, "SEED-CAR-001", 1),
				lapseLoad(ridgeline, "SEED-CAR-004", 8),
			},
			copperhead.CarrierID: {lapseLoad(copperhead, "SEED-CAR-005", 7)},
		},
	}
	reader := NewCoverAgainstLoads(CoverParams{Cover: fake})

	read, err := reader.Read(context.Background(), &boardreaderports.MetricRequest{
		Period:   boardcatalog.PeriodNext30Days,
		Now:      lapseNow,
		Location: time.UTC,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.endingCalls)
	assert.Equal(t, 1, fake.loadsCalls)
	assert.Equal(t, operationsboard.MaxCoverLoads+1, fake.loadsAsked.PerCarrier)
	assert.Equal(t, []pulid.ID{ridgeline.CarrierID, copperhead.CarrierID, sandoval.CarrierID},
		fake.loadsAsked.CarrierIDs)
	assert.Equal(t,
		time.Date(2026, time.December, 23, 23, 59, 59, 0, time.UTC).Unix(),
		fake.endingAsked.Until)

	require.Len(t, read.Chart.Rows, 3)
	first := read.Chart.Rows[0]
	assert.Equal(t, "Ridgeline Freight", first.Label)
	assert.Equal(t, &boardcatalog.Ref{Kind: carrierKind, ID: ridgeline.CarrierID}, first.Ref)
	assert.Equal(t, []boardcatalog.Span{{
		From: lapseNow, To: ridgeline.Policy.ExpirationDate, Kind: boardcatalog.SpanCover,
	}}, first.Spans)
	require.Len(t, first.Dots, 2)
	assert.False(t, first.Dots[0].Marked)
	assert.True(t, first.Dots[1].Marked)
	assert.Equal(t, "SEED-CAR-004 delivers Dec 1", first.Dots[1].Label)
	assert.Empty(t, read.Chart.Rows[2].Dots)

	assert.Equal(t, new(boardcatalog.Count(3)), read.Figures[operationsboard.FigureLapsingCarriers])
	assert.Equal(t, new(boardcatalog.Count(0)), read.Figures[operationsboard.FigureLapsedCarriers])
	assert.Equal(t, new(boardcatalog.Count(1)), read.Figures[operationsboard.FigurePastCover])
	assert.Equal(t, new(boardcatalog.Count(3)), read.Figures[operationsboard.FigureLoadsWithThem])
	assert.Equal(t, time.Date(2026, time.December, 23, 0, 0, 0, 0, time.UTC).Unix(),
		read.Chart.X[len(read.Chart.X)-1].At, "a tick at the period's end draws each bound")
}

func TestCoverAgainstLoadsWithholdsTheLoadFiguresWhenALoadIsNotReadable(t *testing.T) {
	t.Parallel()

	ridgeline := lapseCover("Ridgeline Freight", carrier.InsurancePolicyTypeAutoLiability, 6, 2)
	ridgeline.ReadableLoads = 1
	fake := &fakeCoverReader{
		ending: []*carriercoverports.Cover{ridgeline},
		loads: map[pulid.ID][]carriercoverports.Load{
			ridgeline.CarrierID: {lapseLoad(ridgeline, "SEED-CAR-004", 8)},
		},
	}

	read, err := NewCoverAgainstLoads(CoverParams{Cover: fake}).Read(
		context.Background(),
		&boardreaderports.MetricRequest{Now: lapseNow, Location: time.UTC},
	)
	require.NoError(t, err)
	assert.Nil(t, read.Figures[operationsboard.FigurePastCover])
	assert.Nil(t, read.Figures[operationsboard.FigureLoadsWithThem])
	assert.Equal(t, new(boardcatalog.Count(1)), read.Figures[operationsboard.FigureLapsingCarriers])
	assert.Len(t, read.Chart.Rows[0].Dots, 1, "the readable load is still drawn")
}

func TestCoverAgainstLoadsDrawsAtMostEightCarriersAndWithholdsWhatItCannotCount(t *testing.T) {
	t.Parallel()

	ending := make([]*carriercoverports.Cover, 0, 10)
	for idx := range 10 {
		ending = append(ending,
			lapseCover("Carrier", carrier.InsurancePolicyTypeAutoLiability, int64(idx+1), 0))
	}
	fake := &fakeCoverReader{ending: ending}

	read, err := NewCoverAgainstLoads(CoverParams{Cover: fake}).Read(
		context.Background(),
		&boardreaderports.MetricRequest{Now: lapseNow, Location: time.UTC},
	)
	require.NoError(t, err)
	assert.Len(t, read.Chart.Rows, operationsboard.MaxCoverCarriers)
	assert.Len(t, fake.loadsAsked.CarrierIDs, operationsboard.MaxCoverCarriers)
	assert.Equal(
		t,
		new(boardcatalog.Count(10)),
		read.Figures[operationsboard.FigureLapsingCarriers],
	)
	assert.Nil(t, read.Figures[operationsboard.FigurePastCover],
		"loads past cover over carriers the chart does not draw are not counted")
}

func TestCoverLeftReadsNoLoadsForAReaderWhoMayNotReadShipments(t *testing.T) {
	t.Parallel()

	ridgeline := lapseCover("Ridgeline Freight", carrier.InsurancePolicyTypeAutoLiability, 6, 0)
	ridgeline.LoadsRead = false
	fake := &fakeCoverReader{covers: map[pulid.ID]*carriercoverports.Cover{
		ridgeline.CarrierID: ridgeline,
	}}

	readings, err := NewCoverLeft(CoverParams{Cover: fake}).Readings(
		context.Background(),
		&boardreaderports.RecordRequest{
			IDs: []pulid.ID{ridgeline.CarrierID}, Now: lapseNow, Location: time.UTC,
		},
	)
	require.NoError(t, err)
	assert.False(t, fake.coversAsked.Loads, "the loads are not even counted")
	reading := readings[ridgeline.CarrierID]
	assert.Equal(t, "Auto liability ends {0}", reading.Fact.Text)
	assert.Empty(t, reading.Fact.Place, "no loads phrase, never \"with none of our loads\"")
}
