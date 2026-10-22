package shipmentprovider

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// linkedValue is one value a card shows about a record other than the
// shipments the page is checked for: the kind it comes from and the fields it
// is made of. A figure about a linked record that no reference view carries
// needs that kind's read.
type linkedValue struct {
	kind   permission.Resource
	fields []string
	figure bool
}

// The values the shipment page's cards show about other records. Each is
// decided through the reference view rule, so a reader who may read shipments
// sees what identifies a customer, a location or a driver, and the rest only
// with that kind's read.
var (
	customerName = linkedValue{
		kind:   permission.ResourceCustomer,
		fields: []string{"name"},
	}
	customerFigures = linkedValue{kind: permission.ResourceCustomer, figure: true}
	locationLabel   = linkedValue{
		kind:   permission.ResourceLocation,
		fields: []string{"code", "city", "name"},
	}
	locationRegion = linkedValue{
		kind:   permission.ResourceLocation,
		fields: []string{"state"},
	}
	driverName = linkedValue{
		kind:   permission.ResourceWorker,
		fields: []string{"firstName", "lastName"},
	}
)

// cardAccess is what one reader may be shown on the page's cards.
type cardAccess struct {
	linked permissionports.LinkedFields
	groups permission.FieldGroupSet
}

func (p *Provider) accessFor(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
) *cardAccess {
	actor := analyticsActor(opts)
	access := &cardAccess{}
	if p.views != nil {
		access.linked = p.views.Person(actor, permissionports.PersonGrants{})
	}
	if p.fieldAccess != nil {
		access.groups = p.fieldAccess.Visible(ctx, actor)
	}

	return access
}

func analyticsActor(opts *kernelports.AnalyticsRequestOptions) *permissionports.RequestActor {
	actor := &permissionports.RequestActor{
		PrincipalType:  opts.Principal.Type,
		PrincipalID:    opts.Principal.ID,
		UserID:         opts.Principal.UserID,
		APIKeyID:       opts.Principal.APIKeyID,
		BusinessUnitID: opts.BuID,
		OrganizationID: opts.OrgID,
	}
	if actor.PrincipalType == "" {
		actor.PrincipalType = permissionports.PrincipalTypeUser
		actor.PrincipalID = opts.UserID
		actor.UserID = opts.UserID
	}

	return actor
}

func (a *cardAccess) shows(ctx context.Context, value linkedValue) bool {
	if a.linked == nil {
		return false
	}
	if value.figure {
		return a.linked.Reads(ctx, value.kind)
	}
	for _, field := range value.fields {
		if a.linked.Decide(ctx, value.kind, field) != permissionports.LinkedFieldVisible {
			return false
		}
	}

	return true
}

func (a *cardAccess) showsMargins() bool {
	return a.groups.Has(permission.FieldGroupMargins)
}

func (a *cardAccess) gate(ctx context.Context, data map[string]any) {
	for key, value := range data {
		switch card := value.(type) {
		case *DetentionWatchlistCard:
			a.gateDetentionWatchlist(ctx, card)
		case *CustomerMixCard:
			if card != nil && !a.shows(ctx, customerFigures) {
				data[key] = (*CustomerMixCard)(nil)
			}
		case *TomorrowsPickupsCard:
			a.gateTomorrowsPickups(ctx, card)
		case *LaneHeatmapCard:
			if card != nil && !a.shows(ctx, locationRegion) {
				data[key] = (*LaneHeatmapCard)(nil)
			}
		case *ProfitabilityCard:
			if card != nil && !a.showsMargins() {
				card.AvgMarginPct = nil
			}
		}
	}
}

func (a *cardAccess) gateDetentionWatchlist(ctx context.Context, card *DetentionWatchlistCard) {
	if card == nil || a.shows(ctx, customerName) {
		return
	}
	for _, item := range card.Items {
		item.Customer = nil
	}
}

func (a *cardAccess) gateTomorrowsPickups(ctx context.Context, card *TomorrowsPickupsCard) {
	if card == nil {
		return
	}
	customers := a.shows(ctx, customerName)
	locations := a.shows(ctx, locationLabel)
	drivers := a.shows(ctx, driverName)
	for _, pickup := range card.Pickups {
		if !customers {
			pickup.Customer = nil
		}
		if !locations {
			pickup.Origin = nil
			pickup.Destination = nil
		}
		if !drivers {
			pickup.Driver = nil
		}
	}
}
