package rategeo

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
)

var errKeysShape = errors.New("the rules module answered lane.keys with the wrong shape")

// Scope is one side of a lane: a way of naming a place. Value carries whatever
// the type needs; City is only read for CityState scopes, where Value holds the
// state id. Its key is computed by the rules module (lane.keys).
type Scope struct {
	Type  ScopeType
	Value string
	City  string
}

// Place is everything the engine knows about one end of a shipment's lane.
type Place struct {
	LocationID  pulid.ID
	PostalCode  string
	City        string
	StateID     pulid.ID
	CountryISO3 string
	ZoneIDs     []pulid.ID
}

func (p *Place) Lane() rules.LanePlace {
	zones := make([]string, 0, len(p.ZoneIDs))
	for _, zoneID := range p.ZoneIDs {
		if zoneID.IsNotNil() {
			zones = append(zones, zoneID.String())
		}
	}

	return rules.LanePlace{
		LocationID:  p.LocationID.String(),
		PostalCode:  p.PostalCode,
		City:        p.City,
		StateID:     p.StateID.String(),
		CountryISO3: p.CountryISO3,
		ZoneIDs:     zones,
	}
}

type LaneMatch struct {
	OriginKeys      []string
	DestinationKeys []string
	Candidates      []string
}

func MatchLane(ctx context.Context, origin, destination *Place) (*LaneMatch, error) {
	keys, err := rules.KeyLanes(ctx, &rules.LaneKeysRequest{
		Shipments: []rules.LaneShipment{{Origin: origin.Lane(), Destination: destination.Lane()}},
	})
	if err != nil {
		return nil, fmt.Errorf("key the shipment's lane: %w", err)
	}
	if len(keys.Shipments) != 1 {
		return nil, fmt.Errorf("%w: %d lanes for 1", errKeysShape, len(keys.Shipments))
	}

	lane := keys.Shipments[0]
	return &LaneMatch{
		OriginKeys:      lane.OriginKeys,
		DestinationKeys: lane.DestinationKeys,
		Candidates:      lane.Candidates,
	}, nil
}

func ScopeKeys(ctx context.Context, scopes []Scope) ([]string, error) {
	if len(scopes) == 0 {
		return nil, nil
	}

	req := &rules.LaneKeysRequest{Rules: make([]rules.LaneRule, 0, len(scopes))}
	for _, scope := range scopes {
		req.Rules = append(req.Rules, rules.LaneRule{
			OriginScopeType:  string(scope.Type),
			OriginScopeValue: scope.Value,
			OriginCity:       scope.City,
		})
	}

	keys, err := rules.KeyLanes(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("key scopes: %w", err)
	}
	if len(keys.Rules) != len(scopes) {
		return nil, fmt.Errorf(
			"%w: %d keys for %d scopes",
			errKeysShape,
			len(keys.Rules),
			len(scopes),
		)
	}

	out := make([]string, 0, len(scopes))
	for _, rule := range keys.Rules {
		key := ""
		if rule.Origin != nil {
			key = *rule.Origin
		}
		out = append(out, key)
	}

	return out, nil
}
