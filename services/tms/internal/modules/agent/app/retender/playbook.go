package retender

import (
	"context"
	"sort"
	"strings"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/haulerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Params struct {
	fx.In

	Moves   haulerports.Reader
	Tenders *tendercreate.Service
}

// Playbook is a person offering a move down its routing guide again after a
// tender ran the guide dry. It is the step that follows from the record
// alone: a carrier that passed an hour ago often has a truck now, and the
// offer is the guide's own order and rates. The person is shown which guide
// and which carriers, in order, and sends it under their own name; no agent
// takes part. Shopping other carriers or putting an own driver on the load
// is judgment, and stays with the person or an agent they name.
type Playbook struct {
	moves   haulerports.Reader
	tenders *tendercreate.Service
}

var _ watchtowerports.Playbook = (*Playbook)(nil)

func New(p Params) *Playbook { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Playbook{moves: p.Moves, tenders: p.Tenders}
}

func (p *Playbook) Kind() watchtowerplay.Kind { return watchtowerplay.TenderAgain }

func (p *Playbook) Plan(
	ctx context.Context,
	req *watchtowerports.PlanRequest,
) (*watchtowerports.PlannedPlay, error) {
	tenderID, err := pulid.Parse(req.Item.SourceID)
	if err != nil {
		return nil, errortypes.NewBusinessError("This item is not about a tender")
	}
	moveID, err := p.moves.TenderMove(ctx, req.TenantInfo, tenderID)
	if err != nil {
		return nil, err
	}
	if moveID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"The tender this is about is gone, so there is nothing to offer again.",
		)
	}

	request := &tendercreate.CreateWaterfallTenderRequest{
		TenantInfo:     req.TenantInfo,
		ShipmentMoveID: moveID,
	}
	planned, err := p.tenders.PreviewWaterfall(ctx, request)
	if err != nil {
		return nil, err
	}
	if planned.RefusalError != nil {
		return nil, errortypes.NewBusinessError("{0}", planned.RefusalError.Error())
	}
	carriers := carriersInOrder(planned)
	if len(carriers) == 0 {
		return nil, errortypes.NewBusinessError(
			"The routing guide has no carrier left to offer this move to. Shop other carriers, or put your own driver on it.",
		)
	}

	return &watchtowerports.PlannedPlay{
		Play: watchtowerplay.Play{
			Kind:      watchtowerplay.TenderAgain,
			Channel:   watchtowerplay.ChannelTender,
			Recipient: strings.Join(carriers, ", "),
			Summary:   summary(planned, carriers),
		},
		Needs: []watchtowerports.PlayGrant{
			{Resource: permission.ResourceTender, Operation: permission.OpCreate},
		},
		Asked: "Offered the move down " + guideName(planned) + " again",
		Send: func(ctx context.Context, _ *watchtowerports.PlaySending) error {
			_, cErr := p.tenders.CreateWaterfall(ctx, request)

			return cErr
		},
	}, nil
}

func guideName(planned *tendercreate.TenderPreview) string {
	if planned.Guide != nil && strings.TrimSpace(planned.Guide.Name) != "" {
		return "routing guide " + planned.Guide.Name
	}

	return "its routing guide"
}

// carriersInOrder names the carriers the move would be offered to, in the
// order the guide asks them.
func carriersInOrder(planned *tendercreate.TenderPreview) []string {
	if planned.Tender == nil {
		return nil
	}
	offers := append(planned.Tender.Offers[:0:0], planned.Tender.Offers...)
	sort.SliceStable(offers, func(i, j int) bool { return offers[i].Rank < offers[j].Rank })

	names := make([]string, 0, len(offers))
	for _, offer := range offers {
		if name := strings.TrimSpace(planned.Carriers[offer.CarrierID]); name != "" {
			names = append(names, name)
		}
	}

	return names
}

// summary says what sending does, in the words the person confirms.
func summary(planned *tendercreate.TenderPreview, carriers []string) string {
	text := "Offers the move down " + guideName(planned) + ", one carrier at a time in this " +
		"order: " + strings.Join(carriers, ", ") + ". Each has the guide's time to accept " +
		"before the next is asked."
	if screening := planned.Screening; screening != nil && len(screening.Skipped) > 0 {
		skipped := make([]string, 0, len(screening.Skipped))
		for _, entry := range screening.Skipped {
			skipped = append(skipped, entry.CarrierName)
		}
		text += " Left out as ineligible: " + strings.Join(skipped, ", ") + "."
	}

	return text
}
