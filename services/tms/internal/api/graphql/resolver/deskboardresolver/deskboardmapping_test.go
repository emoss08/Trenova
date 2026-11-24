package deskboardresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardbar"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var shipmentKind = recordkind.Kind(permission.ResourceShipment)

func TestDeskBoardCardRefusesSomeoneWhoMayNotUseTheDesk(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			L: zap.NewNop(),
			PermissionEngine: &mocks.DenyResourcesPermissionEngine{
				Denied: []string{permission.ResourceAssistant.String()},
			},
		},
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	_, err := resolver.DeskBoardCard(
		ctx,
		pulid.MustNew("thr_").String(),
		pulid.MustNew("art_").String(),
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBoardCardToModelCarriesEveryBlockAsTheServerFilledIt(t *testing.T) {
	t.Parallel()

	shipment := pulid.MustNew("shp_")
	behind := boardcatalog.Duration(6000)
	card := &assistantports.BoardCard{
		ArtifactID: pulid.MustNew("art_"),
		ThreadID:   pulid.MustNew("thr_"),
		Title:      "Late loads",
		Author:     "Dispatch Desk",
		AnsweredAt: 1_793_000_000,
		Markdown:   "# Late loads\n",
		Filled: &board.Filled{
			Title:  "Late loads",
			ReadAt: 1_793_000_600,
			Verdict: []board.FilledSegment{
				{Text: "One is "},
				{Text: "late", Numeral: 1, Tone: boardcatalog.ToneWarn},
			},
			Blocks: []board.FilledBlock{{
				Type: board.BlockRecords, Numeral: 1, State: board.StateReady,
				Tone:    boardcatalog.ToneWarn,
				Measure: "lateness",
				Source: board.Source{
					Kind:       board.SourceRecords,
					RecordKind: shipmentKind,
					Count:      2,
					Readable:   1,
				},
				Rows: []board.Row{
					{
						Kind: shipmentKind, ID: shipment, Readable: true,
						Label: "S2610-0412", Status: "Delayed",
						Opens: &boardcatalog.Ref{Kind: shipmentKind, ID: shipment},
						Measure: &board.RowMeasure{
							Measure:  "lateness",
							Value:    &behind,
							Standing: boardreading.Words{Text: "{0} behind", Value: &behind},
							Fact: &boardreading.Words{
								Text: "Due {0}", At: 1_792_994_000, Place: "at Rockford, IL",
							},
							Tone: boardcatalog.ToneWarn,
							Bar: &boardbar.Bar{Segments: []boardbar.Segment{{
								Kind: boardbar.SegmentUsed, Width: 1, Tone: boardcatalog.ToneWarn,
							}}},
						},
					},
					{Kind: shipmentKind, ID: pulid.MustNew("shp_")},
				},
			}},
		},
	}

	model := boardCardToModel(card)

	assert.Equal(t, 1_793_000_600, model.ReadAt)
	assert.Equal(t, "warn", model.Verdict[1].Tone)
	require.Len(t, model.Blocks, 1)
	block := model.Blocks[0]
	assert.Equal(t, "records", block.Type)
	assert.Equal(t, "shipment", block.Source.RecordKind)
	assert.Equal(t, 1, block.Source.Readable)
	require.Len(t, block.Rows, 2)
	row := block.Rows[0]
	assert.Equal(t, shipment.String(), row.ID)
	assert.Equal(t, "shipment", row.Kind)
	assert.Equal(t, shipment.String(), row.Opens.ID)
	assert.Equal(t, "6000", row.Measure.Value.Number)
	assert.Equal(t, "{0} behind", row.Measure.Standing.Text)
	assert.Equal(t, "6000", row.Measure.Standing.Value.Number)
	require.NotNil(t, row.Measure.Fact.At)
	assert.Equal(t, 1_792_994_000, *row.Measure.Fact.At)
	assert.Equal(t, "at Rockford, IL", row.Measure.Fact.Place)
	require.Len(t, row.Measure.Bar.Segments, 1)
	assert.Equal(t, "used", row.Measure.Bar.Segments[0].Kind)
	assert.Equal(t, "warn", row.Measure.Bar.Segments[0].Tone)
	assert.Nil(t, row.Measure.Bar.Bound)
	assert.False(t, block.Rows[1].Readable)
	assert.Nil(t, block.Rows[1].Measure)
	assert.Nil(t, block.Compare)
	assert.Nil(t, block.Table)
	assert.Nil(t, block.Metric)
	assert.Empty(t, block.Figures)
}

func TestBoardCardToModelCarriesThePlaceAndEachReading(t *testing.T) {
	t.Parallel()

	moved := &boardcatalog.Value{Kind: boardcatalog.ValueCount, Number: decimal.NewFromInt(6)}
	card := &assistantports.BoardCard{
		ArtifactID: pulid.MustNew("art_"),
		ThreadID:   pulid.MustNew("thr_"),
		Title:      "Late loads",
		Filled: &board.Filled{
			Place: board.PlaceConversation,
			Verdict: []board.FilledSegment{
				{Text: "There are "}, {Text: "6", Reading: "late"}, {Text: " late."},
			},
			Readings: []board.FilledReading{
				{
					Key:   "late",
					Label: "Late now",
					Value: &boardcatalog.Value{
						Kind: boardcatalog.ValueCount, Number: decimal.NewFromInt(2),
					},
					Was: moved,
					Source: board.Source{
						Kind: board.SourceRecords, RecordKind: shipmentKind,
						Count: 6, Readable: 5,
					},
					Measure:   "lateness",
					Aggregate: board.AggregateCount,
				},
				{Key: "owed", Label: "Owed", Source: board.Source{Kind: board.SourceMetrics}},
			},
		},
	}

	model := boardCardToModel(card)

	assert.Equal(t, "conversation", model.Place)
	assert.Equal(t, "late", model.Verdict[1].Reading)
	assert.Equal(t, "6", model.Verdict[1].Text)
	assert.Empty(t, model.Verdict[0].Reading)
	require.Len(t, model.Readings, 2)
	late := model.Readings[0]
	assert.Equal(t, "late", late.Key)
	assert.Equal(t, "Late now", late.Label)
	assert.Equal(t, "2", late.Value.Number)
	assert.Equal(t, "6", late.Was.Number)
	assert.Equal(t, "shipment", late.Source.RecordKind)
	assert.Equal(t, 5, late.Source.Readable)
	assert.Equal(t, "lateness", late.Measure)
	assert.Equal(t, "count", late.Aggregate)
	assert.Nil(t, model.Readings[1].Value)
	assert.Empty(t, model.Readings[1].Measure)
}
