package agenttoolstest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func (f *FakeTenders) PreviewWaterfall(
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

func (f *FakeTenders) PreviewSpot(
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
