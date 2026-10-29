package tendercreate

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// TenderPreview is a tender as creating it would start it: the offers in the
// order carriers are asked, at their rates and over their channels, and what
// the eligibility screen said about each carrier. Nothing is saved and no
// workflow starts.
type TenderPreview struct {
	Tender       *tender.Tender
	Carriers     map[pulid.ID]string
	Guide        *tender.RoutingGuide
	Screening    *GuideScreeningSummary
	Warnings     []string
	RefusalError error
}

type waterfallPlan struct {
	tender *tender.Tender
	guide  *tender.RoutingGuide
	offers *guideOfferPlan
}

type spotOfferPlan struct {
	offers   []*tender.TenderOffer
	warnings []string
	carriers map[pulid.ID]string
}

type spotPlan struct {
	tender *tender.Tender
	offers *spotOfferPlan
}

// refusal is the error a spot tender is refused with when a carrier on it
// carries insurance warnings the call did not override.
func (p *spotPlan) refusal(req *CreateSpotTenderRequest) error {
	if len(p.offers.warnings) == 0 || req.OverrideInsuranceWarnings {
		return nil
	}

	return errortypes.NewBusinessError(
		"Carrier has insurance warnings: {0}. Confirm the override to proceed",
		strings.Join(p.offers.warnings, "; "),
	).WithParam("overridable", "true")
}

func carrierNames(carriers []*carrier.Carrier) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(carriers))
	for _, entity := range carriers {
		if entity != nil {
			names[entity.ID] = entity.Name
		}
	}

	return names
}
