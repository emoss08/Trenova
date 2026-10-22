package rules

import (
	"context"

	"github.com/shopspring/decimal"
)

const (
	OpLaneKeys        = "lane.keys"
	OpLaneSpecificity = "lane.specificity"

	LaneRadiusKey = "RADIUS"
)

type LaneRule struct {
	Status                string              `json:"status,omitempty"`
	Direction             string              `json:"direction,omitempty"`
	OriginScopeType       string              `json:"originScopeType,omitempty"`
	OriginScopeValue      string              `json:"originScopeValue,omitempty"`
	OriginCity            string              `json:"originCity,omitempty"`
	DestinationScopeType  string              `json:"destinationScopeType,omitempty"`
	DestinationScopeValue string              `json:"destinationScopeValue,omitempty"`
	DestinationCity       string              `json:"destinationCity,omitempty"`
	CommodityIDs          []string            `json:"commodityIds,omitempty"`
	FreightClasses        []string            `json:"freightClasses,omitempty"`
	TractorTypeIDs        []string            `json:"tractorTypeIds,omitempty"`
	TrailerTypeIDs        []string            `json:"trailerTypeIds,omitempty"`
	EquipmentClasses      []string            `json:"equipmentClasses,omitempty"`
	ServiceTypeIDs        []string            `json:"serviceTypeIds,omitempty"`
	ShipmentTypeIDs       []string            `json:"shipmentTypeIds,omitempty"`
	ServiceModels         []string            `json:"serviceModels,omitempty"`
	MinWeight             decimal.NullDecimal `json:"minWeight"`
	MaxWeight             decimal.NullDecimal `json:"maxWeight"`
	MinDistance           decimal.NullDecimal `json:"minDistance"`
	MaxDistance           decimal.NullDecimal `json:"maxDistance"`
	HazmatOnly            bool                `json:"hazmatOnly"`
	TempControlOnly       bool                `json:"tempControlOnly"`
}

type LanePlace struct {
	LocationID  string   `json:"locationId,omitempty"`
	PostalCode  string   `json:"postalCode,omitempty"`
	City        string   `json:"city,omitempty"`
	StateID     string   `json:"stateId,omitempty"`
	CountryISO3 string   `json:"countryIso3,omitempty"`
	ZoneIDs     []string `json:"zoneIds,omitempty"`
}

type LaneShipment struct {
	Origin      LanePlace `json:"origin"`
	Destination LanePlace `json:"destination"`
}

type LaneKeysRequest struct {
	Rules     []LaneRule     `json:"rules"`
	Shipments []LaneShipment `json:"shipments"`
}

type LaneRuleKeys struct {
	Origin      *string `json:"origin"`
	Destination *string `json:"destination"`
	Key         *string `json:"key"`
	Stored      string  `json:"stored"`
	Reverse     *string `json:"reverse"`
}

type LaneShipmentKeys struct {
	OriginKeys      []string `json:"originKeys"`
	DestinationKeys []string `json:"destinationKeys"`
	Candidates      []string `json:"candidates"`
}

type LaneKeys struct {
	Rules     []LaneRuleKeys     `json:"rules"`
	Shipments []LaneShipmentKeys `json:"shipments"`
}

type laneRulesRequest struct {
	Rules []LaneRule `json:"rules"`
}

type laneScores struct {
	Scores []int32 `json:"scores"`
}

func KeyLanes(ctx context.Context, req *LaneKeysRequest) (*LaneKeys, error) {
	var out LaneKeys
	if err := Call(ctx, OpLaneKeys, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func ScoreLanes(ctx context.Context, lanes []LaneRule) ([]int32, error) {
	var out laneScores
	if err := Call(ctx, OpLaneSpecificity, &laneRulesRequest{Rules: lanes}, &out); err != nil {
		return nil, err
	}
	return out.Scores, nil
}
