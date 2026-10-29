package tendercreate

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CreateWaterfallTenderRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	RoutingGuideID *pulid.ID             `json:"routingGuideId"`
}

func (r *CreateWaterfallTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type SpotTenderLine struct {
	CarrierID       pulid.ID                        `json:"carrierId"`
	RateMethod      shipmentvocab.CarrierRateMethod `json:"rateMethod"`
	Rate            decimal.Decimal                 `json:"rate"`
	OfferTTLSeconds int32                           `json:"offerTtlSeconds"`
	Channel         tender.Channel                  `json:"channel"`
	Email           string                          `json:"email"`
}

type CreateSpotTenderRequest struct {
	TenantInfo                pagination.TenantInfo `json:"-"`
	ShipmentMoveID            pulid.ID              `json:"shipmentMoveId"`
	Mode                      tender.Mode           `json:"mode"`
	Lines                     []SpotTenderLine      `json:"lines"`
	OverrideInsuranceWarnings bool                  `json:"overrideInsuranceWarnings"`
}

func (r *CreateSpotTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if r.Mode != tender.ModeSpotBroadcast && r.Mode != tender.ModeSpotSequential {
		multiErr.Add(
			"mode",
			errortypes.ErrInvalid,
			"Spot tenders must be broadcast or sequential",
		)
	}
	if len(r.Lines) == 0 {
		multiErr.Add("lines", errortypes.ErrRequired, "At least one carrier line is required")
	}
	seen := make(map[pulid.ID]struct{}, len(r.Lines))
	for idx, line := range r.Lines {
		lineErr := multiErr.WithIndex("lines", idx)
		if line.CarrierID.IsNil() {
			lineErr.Add("carrierId", errortypes.ErrRequired, "Carrier is required")
		}
		if _, dup := seen[line.CarrierID]; dup {
			lineErr.Add("carrierId", errortypes.ErrInvalid, "Carrier is already on this tender")
		}
		seen[line.CarrierID] = struct{}{}
		if line.RateMethod != "" && !line.RateMethod.IsValid() {
			lineErr.Add("rateMethod", errortypes.ErrInvalid, "Rate method is invalid")
		}
		if line.Rate.IsNegative() {
			lineErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
		}
		if line.Channel != "" && !line.Channel.IsValid() {
			lineErr.Add("channel", errortypes.ErrInvalid, "Channel is invalid")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// GuideEntryScreeningResult is one routing guide entry's eligibility verdict as
// reported back to the caller that started the tender.
type GuideEntryScreeningResult struct {
	CarrierName string   `json:"carrierName"`
	Rank        int16    `json:"rank"`
	Reasons     []string `json:"reasons"`
}

// GuideScreeningSummary reports which routing guide entries never became offers
// and which were offered despite a warning.
type GuideScreeningSummary struct {
	Skipped []GuideEntryScreeningResult `json:"skipped"`
	Warned  []GuideEntryScreeningResult `json:"warned"`
}

// CreateWaterfallResult embeds the created tender so the response body stays a
// superset of the tender itself, with the screening summary attached.
type CreateWaterfallResult struct {
	*tender.Tender

	Screening *GuideScreeningSummary `json:"screening,omitempty"`
}

// guideEntryScreening captures one routing guide entry's eligibility verdict
// so the audit trail can be written after the tender row exists.
type guideEntryScreening struct {
	carrierName string
	rank        int16
	reasons     []string
}

// guideOfferPlan is the screened offer plan for a waterfall tender: the offers
// that survived the eligibility gate plus the entries skipped or warned on.
type guideOfferPlan struct {
	offers   []*tender.TenderOffer
	skipped  []guideEntryScreening
	warned   []guideEntryScreening
	carriers map[pulid.ID]string
}

func (p *guideOfferPlan) screeningSummary() *GuideScreeningSummary {
	if len(p.skipped) == 0 && len(p.warned) == 0 {
		return nil
	}
	return &GuideScreeningSummary{
		Skipped: screeningResults(p.skipped),
		Warned:  screeningResults(p.warned),
	}
}

func screeningResults(entries []guideEntryScreening) []GuideEntryScreeningResult {
	results := make([]GuideEntryScreeningResult, 0, len(entries))
	for _, entry := range entries {
		results = append(results, GuideEntryScreeningResult{
			CarrierName: entry.carrierName,
			Rank:        entry.rank,
			Reasons:     entry.reasons,
		})
	}
	return results
}

func guideEntryCarrierName(entry *tender.RoutingGuideEntry) string {
	if entry.Carrier != nil && entry.Carrier.Name != "" {
		return entry.Carrier.Name
	}
	return entry.CarrierID.String()
}
