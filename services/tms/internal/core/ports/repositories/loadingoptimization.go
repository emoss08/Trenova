package repositories

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LoadingCommodityInput struct {
	CommodityID pulid.ID `json:"commodityId"`
	Pieces      int64    `json:"pieces"`
	Weight      int64    `json:"weight"`
}

type StopInfo struct {
	Sequence     int    `json:"sequence"`
	LocationName string `json:"locationName"`
	LocationCity string `json:"locationCity"`
}

type LoadingOptimizationRequest struct {
	TenantInfo      pagination.TenantInfo   `json:"-"`
	Commodities     []LoadingCommodityInput `json:"commodities"`
	EquipmentTypeID *pulid.ID               `json:"equipmentTypeId,omitempty"`
	Stops           []StopInfo              `json:"stops,omitempty"`
}

func (r *LoadingOptimizationRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if len(r.Commodities) == 0 {
		multiErr.Add("commodities", errortypes.ErrRequired, "At least one commodity is required")
	}

	for i, c := range r.Commodities {
		if c.CommodityID.IsNil() {
			multiErr.WithIndex("commodities", i).
				Add("commodityId", errortypes.ErrRequired, "Commodity ID is required")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type CommodityPlacement struct {
	CommodityID         pulid.ID `json:"commodityId"`
	CommodityName       string   `json:"commodityName"`
	PositionFeet        float64  `json:"positionFeet"`
	LengthFeet          float64  `json:"lengthFeet"`
	Weight              int64    `json:"weight"`
	Pieces              int64    `json:"pieces"`
	Stackable           bool     `json:"stackable"`
	Fragile             bool     `json:"fragile"`
	IsHazmat            bool     `json:"isHazmat"`
	HazmatClass         string   `json:"hazmatClass,omitempty"`
	MinTemp             *int     `json:"minTemp,omitempty"`
	MaxTemp             *int     `json:"maxTemp,omitempty"`
	LoadingInstructions string   `json:"loadingInstructions,omitempty"`
	EstimatedLength     bool     `json:"estimatedLength"`
	StopNumber          int      `json:"stopNumber,omitempty"`
}

type HazmatZoneResult struct {
	CommodityAID         pulid.ID `json:"commodityAId"`
	CommodityBID         pulid.ID `json:"commodityBId"`
	CommodityAName       string   `json:"commodityAName"`
	CommodityBName       string   `json:"commodityBName"`
	RuleName             string   `json:"ruleName"`
	SegregationType      string   `json:"segregationType"`
	RequiredDistanceFeet *float64 `json:"requiredDistanceFeet,omitempty"`
	ActualDistanceFeet   float64  `json:"actualDistanceFeet"`
	Satisfied            bool     `json:"satisfied"`
}

type LoadingWarning struct {
	Type         string   `json:"type"`
	Message      string   `json:"message"`
	Severity     string   `json:"severity"`
	CommodityIDs []string `json:"commodityIds,omitempty"`
}

type AxleWeight struct {
	Axle       string  `json:"axle"`
	Weight     int64   `json:"weight"`
	Limit      int64   `json:"limit"`
	Percentage float64 `json:"percentage"`
	Compliant  bool    `json:"compliant"`
}

type LoadingRecommendation struct {
	Type         string   `json:"type"`
	Priority     string   `json:"priority"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Impact       string   `json:"impact,omitempty"`
	CommodityIDs []string `json:"commodityIds,omitempty"`
}

type StopDivider struct {
	PositionFeet float64 `json:"positionFeet"`
	StopNumber   int     `json:"stopNumber"`
	Label        string  `json:"label"`
}

type LoadingOptimizationResult struct {
	TrailerLengthFeet float64                 `json:"trailerLengthFeet"`
	TotalLinearFeet   float64                 `json:"totalLinearFeet"`
	TotalWeight       int64                   `json:"totalWeight"`
	MaxWeight         int64                   `json:"maxWeight"`
	LinearFeetUtil    float64                 `json:"linearFeetUtil"`
	WeightUtil        float64                 `json:"weightUtil"`
	UtilizationScore  int                     `json:"utilizationScore"`
	UtilizationGrade  string                  `json:"utilizationGrade"`
	Placements        []CommodityPlacement    `json:"placements"`
	HazmatZones       []HazmatZoneResult      `json:"hazmatZones"`
	Warnings          []LoadingWarning        `json:"warnings"`
	AxleWeights       []AxleWeight            `json:"axleWeights"`
	Recommendations   []LoadingRecommendation `json:"recommendations"`
	StopDividers      []StopDivider           `json:"stopDividers,omitempty"`
	AIAnalysis        string                  `json:"aiAnalysis,omitempty"`
}
