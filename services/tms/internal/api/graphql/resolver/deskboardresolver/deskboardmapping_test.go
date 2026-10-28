package deskboardresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDeskBoardCardRefusesSomeoneWhoMayNotUseTheDesk(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core: &base.Core{
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
				{Text: "late", Numeral: 1, Tone: board.ToneWarn},
			},
			Blocks: []board.FilledBlock{{
				Type: board.BlockRecords, Numeral: 1, State: board.StateReady, Tone: board.ToneWarn,
				Measure: board.MeasureLateness,
				Source: board.Source{
					Kind:     board.SourceRecords,
					Resource: permission.ResourceShipment,
					Count:    2,
					Readable: 1,
				},
				Rows: []board.Row{
					{
						Resource: permission.ResourceShipment, ID: shipment, Readable: true,
						Label: "S2610-0412", Status: "Delayed",
						Measure: &board.RowMeasure{
							Measure: board.MeasureLateness, State: board.MeasureLate, At: 1_792_994_000,
							Tone: board.ToneWarn,
							Value: board.Value{
								Kind:   board.ValueDuration,
								Number: decimal.NewFromInt(6000),
							},
						},
					},
					{Resource: permission.ResourceShipment, ID: pulid.MustNew("shp_")},
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
	assert.Equal(t, "shipment", block.Source.Resource)
	assert.Equal(t, 1, block.Source.Readable)
	require.Len(t, block.Rows, 2)
	assert.Equal(t, shipment.String(), block.Rows[0].ID)
	assert.Equal(t, "6000", block.Rows[0].Measure.Value.Number)
	require.NotNil(t, block.Rows[0].Measure.At)
	assert.Equal(t, 1_792_994_000, *block.Rows[0].Measure.At)
	assert.False(t, block.Rows[1].Readable)
	assert.Nil(t, block.Rows[1].Measure)
	assert.Nil(t, block.Compare)
	assert.Nil(t, block.Table)
	assert.Nil(t, block.Chart)
	assert.Empty(t, block.Figures)
}
