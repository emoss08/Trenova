package rateresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/ratetypes"
)

func withholdShopResult(result *ratequoteports.ShopResult, visible protectedfield.GroupSet) {
	result.WithholdFields(visible)
	for _, option := range result.Options {
		if option != nil {
			option.Quote.WithholdFields(visible)
		}
	}
}

func rateShopResultToModel(result *ratequoteports.ShopResult) (*gqlmodel.RateShopResult, error) {
	options := make([]*gqlmodel.RateShopOption, 0, len(result.Options))
	for _, option := range result.Options {
		if option == nil {
			continue
		}
		quote, err := rateQuoteToModel(option.Quote)
		if err != nil {
			return nil, err
		}
		options = append(options, &gqlmodel.RateShopOption{
			CarrierID:       option.CarrierID.String(),
			CarrierName:     option.CarrierName,
			Rank:            option.Rank,
			GuideRank:       int(option.GuideRank),
			OfferTTLSeconds: int(option.OfferTTLSeconds),
			Outcome:         gqlmodel.RateQuoteOutcome(option.Outcome),
			Cost:            option.Cost.String(),
			Currency:        option.Currency,
			Margin:          rateMarginVerdictToModel(option.Margin),
			Quote:           quote,
			AgreementID:     gqlid.IDPtrFromPulidPtr(option.AgreementID),
			RuleID:          gqlid.IDPtrFromPulidPtr(option.RuleID),
			Note:            option.Note,
		})
	}
	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	return &gqlmodel.RateShopResult{
		Strategy:       result.Strategy,
		Options:        options,
		RoutingGuideID: gqlid.IDPtrFromPulidPtr(result.RoutingGuideID),
		SellTotal:      gqldecimal.NullDecimalToStringPtr(result.SellTotal),
		Warnings:       warnings,
	}, nil
}

func rateMarginVerdictToModel(verdict *ratetypes.MarginVerdict) *gqlmodel.RateMarginVerdict {
	if verdict == nil {
		return nil
	}

	return &gqlmodel.RateMarginVerdict{
		Amount:          verdict.Amount.String(),
		Percent:         verdict.Percent.String(),
		PayPercent:      verdict.PayPercent.String(),
		FloorApplies:    verdict.FloorApplies,
		BelowFloor:      verdict.BelowFloor,
		CeilingApplies:  verdict.CeilingApplies,
		AbovePayCeiling: verdict.AbovePayCeiling,
		Explanation:     verdict.Explanation,
	}
}
