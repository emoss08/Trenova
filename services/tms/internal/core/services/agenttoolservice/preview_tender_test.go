package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fake tender service answers a preview from the request, so a preview
// that started a tender would show in fakeTenders' own record of the write.

func (f *fakeTenders) PreviewWaterfall(
	_ context.Context,
	req *tenderservice.CreateWaterfallTenderRequest,
) (*tenderservice.TenderPreview, error) {
	first, second := pulid.MustNew("carr_"), pulid.MustNew("carr_")

	return &tenderservice.TenderPreview{
		Tender: &tender.Tender{
			ShipmentMoveID: req.ShipmentMoveID,
			RoutingGuideID: req.RoutingGuideID,
			Mode:           tender.ModeWaterfall,
			Status:         tender.StatusActive,
			Offers: []*tender.TenderOffer{
				{
					CarrierID:       first,
					Rank:            1,
					Rate:            decimal.NewFromInt(1450),
					OfferTTLSeconds: 1800,
					Channel:         tender.ChannelEmail,
					RecipientEmail:  "dispatch@acme.example",
				},
				{
					CarrierID:       second,
					Rank:            2,
					Rate:            decimal.NewFromInt(1500),
					OfferTTLSeconds: 1800,
					Channel:         tender.ChannelEDI,
				},
			},
		},
		Carriers: map[pulid.ID]string{first: "Acme Trucking", second: "Blue Line"},
		Guide:    &tender.RoutingGuide{Name: "Dallas outbound"},
		Screening: &tenderservice.GuideScreeningSummary{
			Skipped: []tenderservice.GuideEntryScreeningResult{
				{CarrierName: "Lapsed Freight", Rank: 3, Reasons: []string{"insurance lapsed"}},
			},
		},
	}, nil
}

func (f *fakeTenders) PreviewSpot(
	_ context.Context,
	req *tenderservice.CreateSpotTenderRequest,
) (*tenderservice.TenderPreview, error) {
	offers := make([]*tender.TenderOffer, 0, len(req.Lines))
	names := make(map[pulid.ID]string, len(req.Lines))
	for index, line := range req.Lines {
		offers = append(offers, &tender.TenderOffer{
			CarrierID:       line.CarrierID,
			Rank:            int16(index + 1),
			RateMethod:      line.RateMethod,
			Rate:            line.Rate,
			OfferTTLSeconds: line.OfferTTLSeconds,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "carrier@example.test",
		})
		names[line.CarrierID] = "Carrier " + string(rune('A'+index))
	}

	preview := &tenderservice.TenderPreview{
		Tender:   &tender.Tender{Mode: req.Mode, Status: tender.StatusActive, Offers: offers},
		Carriers: names,
		Warnings: []string{"Carrier A — insurance expires in 5 days"},
	}
	if !req.OverrideInsuranceWarnings {
		preview.RefusalError = errors.New("Carrier has insurance warnings")
	}

	return preview, nil
}

func TestTenderToRoutingGuidePreview_ShowsTheOffersInOrder(t *testing.T) {
	t.Parallel()

	tenders := &fakeTenders{}
	tool := newTenderToRoutingGuideTool(tenders)
	params := executeParams(map[string]any{"shipmentMoveId": pulid.MustNew("smv_").String()})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, tenders.waterfall, "a preview must not start a tender")
	assert.Contains(t, preview.Summary, "Dallas outbound")
	assert.Contains(t, preview.Summary, "Lapsed Freight (insurance lapsed)")

	created := findChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceTender, created.Resource)
	require.NotNil(t, created.Money)
	require.Len(t, created.Money.Lines, 2)
	assert.Equal(t, "1. Acme Trucking (flat)", created.Money.Lines[0].Label)
	assert.False(t, created.Money.TotalAfter.Valid, "offers are alternatives, not a sum")

	var sends []agent.RecordChange
	for i := range preview.Changes {
		if preview.Changes[i].Operation == agent.PreviewOperationSend {
			sends = append(sends, preview.Changes[i])
		}
	}
	require.Len(t, sends, 2)
	assert.Equal(t, agent.MessageChannelEmail, sends[0].Message.Channel)
	assert.Equal(t, []string{"dispatch@acme.example"}, sends[0].Message.To)
	assert.Equal(t, agent.MessageChannelEDI, sends[1].Message.Channel)
	assert.Contains(t, sends[1].Message.Body, "only if every carrier before it declines")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, tenders.waterfall)
}

func TestTenderToCarriersPreview_WarnsOfTheRefusalWithoutAnOverride(t *testing.T) {
	t.Parallel()

	tenders := &fakeTenders{}
	tool := newTenderToCarriersTool(tenders)
	params := executeParams(map[string]any{
		"shipmentMoveId": pulid.MustNew("smv_").String(),
		"mode":           "SpotBroadcast",
		"lines": []any{
			map[string]any{"carrierId": pulid.MustNew("carr_").String(), "rate": "1450.00"},
		},
	})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, tenders.spot)
	assert.Contains(t, preview.Summary, "all at once")
	assert.Contains(t, preview.Summary, "insurance expires")
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
}
