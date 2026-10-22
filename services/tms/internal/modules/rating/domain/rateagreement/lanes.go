package rateagreement

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
)

const RadiusLaneKey = rules.LaneRadiusKey

var (
	ErrLaneNotApplied = errors.New(
		"a rate agreement rule is written before its lane key and specificity were applied",
	)

	errLaneShape = errors.New("the rules module answered for a different number of lanes")
)

type laneState struct {
	applied     bool
	origin      bool
	destination bool
}

func (rar *RateAgreementRule) laneRule() rules.LaneRule {
	return rules.LaneRule{
		Status:                string(rar.Status),
		Direction:             string(rar.Direction),
		OriginScopeType:       string(rar.OriginScopeType),
		OriginScopeValue:      rar.OriginScopeValue,
		OriginCity:            rar.OriginCity,
		DestinationScopeType:  string(rar.DestinationScopeType),
		DestinationScopeValue: rar.DestinationScopeValue,
		DestinationCity:       rar.DestinationCity,
		CommodityIDs:          stringsOf(rar.CommodityIDs),
		FreightClasses:        stringsOf(rar.FreightClasses),
		TractorTypeIDs:        stringsOf(rar.TractorTypeIDs),
		TrailerTypeIDs:        stringsOf(rar.TrailerTypeIDs),
		EquipmentClasses:      stringsOf(rar.EquipmentClasses),
		ServiceTypeIDs:        stringsOf(rar.ServiceTypeIDs),
		ShipmentTypeIDs:       stringsOf(rar.ShipmentTypeIDs),
		ServiceModels:         stringsOf(rar.ServiceModels),
		MinWeight:             rar.MinWeight,
		MaxWeight:             rar.MaxWeight,
		MinDistance:           rar.MinDistance,
		MaxDistance:           rar.MaxDistance,
		HazmatOnly:            rar.HazmatOnly,
		TempControlOnly:       rar.TempControlOnly,
	}
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

func ApplyLanes(ctx context.Context, entities []*RateAgreementRule) error {
	batch := make([]*RateAgreementRule, 0, len(entities))
	for _, rule := range entities {
		if rule != nil {
			rule.applyDefaults()
			batch = append(batch, rule)
		}
	}
	if len(batch) == 0 {
		return nil
	}

	lanes := make([]rules.LaneRule, 0, len(batch))
	for _, rule := range batch {
		lanes = append(lanes, rule.laneRule())
	}

	keys, err := keyLanes(ctx, lanes)
	if err != nil {
		return err
	}
	scores, err := scoreLanes(ctx, lanes)
	if err != nil {
		return err
	}
	if len(keys) != len(batch) || len(scores) != len(batch) {
		return fmt.Errorf(
			"%w: %d keys and %d scores for %d rules",
			errLaneShape,
			len(keys),
			len(scores),
			len(batch),
		)
	}

	for i, rule := range batch {
		rule.LaneKey = keys[i].Stored
		rule.SpecificityScore = scores[i]
		rule.lane = laneState{
			applied:     true,
			origin:      keys[i].Origin != nil,
			destination: keys[i].Destination != nil,
		}
	}

	return nil
}

func keyLanes(ctx context.Context, lanes []rules.LaneRule) ([]rules.LaneRuleKeys, error) {
	out, err := rules.KeyLanes(ctx, &rules.LaneKeysRequest{Rules: lanes})
	if err != nil {
		return nil, fmt.Errorf("key rate agreement lanes: %w", err)
	}

	return out.Rules, nil
}

func scoreLanes(ctx context.Context, lanes []rules.LaneRule) ([]int32, error) {
	scores, err := rules.ScoreLanes(ctx, lanes)
	if err != nil {
		return nil, fmt.Errorf("score rate agreement lanes: %w", err)
	}

	return scores, nil
}

func (rar *RateAgreementRule) KeyedEnds() (origin, destination bool) {
	if !rar.lane.applied {
		return true, true
	}

	return rar.lane.origin, rar.lane.destination
}
