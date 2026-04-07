package shipmentservice

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	defaultTrailerLengthFeet = 53.0
	defaultMaxWeight         = int64(80_000)
	minEstimatedFt           = 2.0
	kingpinOffsetFt          = 2.0
	rearAxleFromKingpinFt    = 40.0
	steerAxleLimitLbs        = int64(12_000)
	driveAxleLimitLbs        = int64(34_000)
	trailerAxleLimitLbs      = int64(34_000)
)

func (s *service) CalculateLoadingOptimization(
	ctx context.Context,
	req *repositories.LoadingOptimizationRequest,
) (*repositories.LoadingOptimizationResult, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	commodityIDs := make([]pulid.ID, 0, len(req.Commodities))
	for _, c := range req.Commodities {
		commodityIDs = append(commodityIDs, c.CommodityID)
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
		TenantInfo:   req.TenantInfo,
		CommodityIDs: commodityIDs,
	})
	if err != nil {
		return nil, err
	}

	trailerLength := defaultTrailerLengthFeet
	if req.EquipmentTypeID != nil && !req.EquipmentTypeID.IsNil() {
		et, etErr := s.equipmentTypeRepo.GetByID(ctx, repositories.GetEquipmentTypeByIDRequest{
			ID:         *req.EquipmentTypeID,
			TenantInfo: req.TenantInfo,
		})
		if etErr == nil && et.InteriorLength != nil && *et.InteriorLength > 0 {
			trailerLength = *et.InteriorLength
		}
	}

	control, _ := s.getShipmentControl(ctx, req.TenantInfo)
	maxWeight := defaultMaxWeight
	if control != nil && control.MaxShipmentWeightLimit > 0 {
		maxWeight = int64(control.MaxShipmentWeightLimit)
	}

	checkHazmat := control != nil && control.CheckHazmatSegregation

	var rules []*hazmatsegregationrule.HazmatSegregationRule
	if checkHazmat {
		rules, _ = s.hazmatRuleRepo.ListActiveByTenant(ctx, req.TenantInfo)
	}

	commodityByID := make(map[pulid.ID]*commodity.Commodity, len(commodities))
	for _, c := range commodities {
		commodityByID[c.ID] = c
	}

	return buildLoadingPlan(&loadPlanParams{
		commodityByID: commodityByID,
		inputs:        req.Commodities,
		stops:         req.Stops,
		rules:         rules,
		trailerLength: trailerLength,
		maxWeight:     maxWeight,
		checkHazmat:   checkHazmat,
	}), nil
}

type loadPlanParams struct {
	commodityByID map[pulid.ID]*commodity.Commodity
	inputs        []repositories.LoadingCommodityInput
	stops         []repositories.StopInfo
	rules         []*hazmatsegregationrule.HazmatSegregationRule
	trailerLength float64
	maxWeight     int64
	checkHazmat   bool
}

type placementCandidate struct {
	commodity  *commodity.Commodity
	input      repositories.LoadingCommodityInput
	lengthFt   float64
	isHazmat   bool
	estimated  bool
	stopNumber int
}

func buildLoadingPlan(p *loadPlanParams) *repositories.LoadingOptimizationResult {
	candidates := buildCandidates(p.commodityByID, p.inputs, p.trailerLength, p.maxWeight)
	assignStopNumbers(candidates, p.stops)
	sortCandidates(candidates, len(p.stops) > 0)

	placements := placeCommodities(candidates, p.rules, p.trailerLength, p.checkHazmat)

	totalLinearFeet, totalWeight := computeTotals(placements)
	hazmatZones := evaluateHazmatZones(placements, p.commodityByID, p.rules)
	warnings := generateWarnings(placements, p.commodityByID, totalLinearFeet, totalWeight, p.trailerLength, p.maxWeight, hazmatZones)
	axleWeights := computeAxleWeights(placements)
	stopDividers := computeStopDividers(placements, p.stops)

	linearFeetUtil := safePercent(totalLinearFeet, p.trailerLength)
	weightUtil := safePercent(float64(totalWeight), float64(p.maxWeight))
	score, grade := computeUtilizationScore(linearFeetUtil, weightUtil)
	recommendations := generateRecommendations(placements, axleWeights, hazmatZones, totalLinearFeet, totalWeight, p.trailerLength, p.maxWeight)

	return &repositories.LoadingOptimizationResult{
		TrailerLengthFeet: p.trailerLength,
		TotalLinearFeet:   roundTo2(totalLinearFeet),
		TotalWeight:       totalWeight,
		MaxWeight:         p.maxWeight,
		LinearFeetUtil:    roundTo2(linearFeetUtil),
		WeightUtil:        roundTo2(weightUtil),
		UtilizationScore:  score,
		UtilizationGrade:  grade,
		Placements:        placements,
		HazmatZones:       hazmatZones,
		Warnings:          warnings,
		AxleWeights:       axleWeights,
		Recommendations:   recommendations,
		StopDividers:      stopDividers,
	}
}

// --- Candidate Building ---

func buildCandidates(
	commodityByID map[pulid.ID]*commodity.Commodity,
	inputs []repositories.LoadingCommodityInput,
	trailerLength float64,
	maxWeight int64,
) []placementCandidate {
	candidates := make([]placementCandidate, 0, len(inputs))

	for _, input := range inputs {
		com, ok := commodityByID[input.CommodityID]
		if !ok {
			continue
		}

		lengthFt, estimated := estimateCommodityLength(com, input, trailerLength, maxWeight)
		isHazmat := com.HazardousMaterial != nil && !com.HazardousMaterialID.IsNil()

		candidates = append(candidates, placementCandidate{
			commodity: com,
			input:     input,
			lengthFt:  lengthFt,
			isHazmat:  isHazmat,
			estimated: estimated,
		})
	}

	return candidates
}

func estimateCommodityLength(
	com *commodity.Commodity,
	input repositories.LoadingCommodityInput,
	trailerLength float64,
	maxWeight int64,
) (float64, bool) {
	if com.LinearFeetPerUnit != nil && *com.LinearFeetPerUnit > 0 {
		return *com.LinearFeetPerUnit * float64(input.Pieces), false
	}

	lengthFt := minEstimatedFt
	if maxWeight > 0 {
		lengthFt = (float64(input.Weight) / float64(maxWeight)) * trailerLength
	}

	return max(lengthFt, minEstimatedFt), true
}

func assignStopNumbers(candidates []placementCandidate, stops []repositories.StopInfo) {
	if len(stops) <= 1 {
		return
	}

	perStop := max(len(candidates)/len(stops), 1)
	for i := range candidates {
		stopIdx := min(i/perStop, len(stops)-1)
		candidates[i].stopNumber = stops[stopIdx].Sequence + 1
	}
}

// --- Sorting ---

func sortCandidates(candidates []placementCandidate, hasStops bool) {
	slices.SortStableFunc(candidates, func(a, b placementCandidate) int {
		// LIFO: reverse delivery order (highest stop number = nose, lowest = doors)
		if hasStops && a.stopNumber != b.stopNumber {
			return b.stopNumber - a.stopNumber
		}

		// Hazmat items first for maximum separation
		if a.isHazmat != b.isHazmat {
			if a.isHazmat {
				return -1
			}
			return 1
		}

		// Group temperature-compatible items together
		if tempCmp := compareTemperatureGroup(a, b); tempCmp != 0 {
			return tempCmp
		}

		// Fragile items last (loaded last = near doors = unloaded first)
		if a.commodity.Fragile != b.commodity.Fragile {
			if a.commodity.Fragile {
				return 1
			}
			return -1
		}

		// Heaviest first for axle distribution
		if a.input.Weight != b.input.Weight {
			if a.input.Weight > b.input.Weight {
				return -1
			}
			return 1
		}

		return 0
	})
}

func compareTemperatureGroup(a, b placementCandidate) int {
	aHasTemp := a.commodity.MinTemperature != nil && a.commodity.MaxTemperature != nil
	bHasTemp := b.commodity.MinTemperature != nil && b.commodity.MaxTemperature != nil

	if !aHasTemp && !bHasTemp {
		return 0
	}
	if aHasTemp && !bHasTemp {
		return -1
	}
	if !aHasTemp && bHasTemp {
		return 1
	}

	return *a.commodity.MinTemperature - *b.commodity.MinTemperature
}

// --- Placement ---

func placeCommodities(
	candidates []placementCandidate,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
	trailerLength float64,
	checkHazmat bool,
) []repositories.CommodityPlacement {
	placements := make([]repositories.CommodityPlacement, 0, len(candidates))
	cursor := 0.0
	separatedPairs := make(map[pulid.ID]bool)

	for _, cand := range candidates {
		if separatedPairs[cand.commodity.ID] {
			continue
		}

		position := cursor
		if checkHazmat && cand.isHazmat && len(rules) > 0 {
			position = adjustForHazmatDistance(cand, placements, rules, cursor)
		}

		placements = append(placements, buildPlacement(cand, position))
		cursor = position + cand.lengthFt

		if checkHazmat && cand.isHazmat {
			for _, other := range candidates {
				if other.commodity.ID == cand.commodity.ID || !other.isHazmat {
					continue
				}
				rule := findMatchingHazmatRule(rules, cand.commodity, other.commodity)
				if rule != nil && rule.SegregationType == hazmatsegregationrule.SegregationTypeSeparated {
					otherPos := max(trailerLength-other.lengthFt, cursor)
					placements = append(placements, buildPlacement(other, otherPos))
					separatedPairs[other.commodity.ID] = true
				}
			}
		}
	}

	return placements
}

func adjustForHazmatDistance(
	cand placementCandidate,
	placed []repositories.CommodityPlacement,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
	cursor float64,
) float64 {
	position := cursor

	for _, existing := range placed {
		if !existing.IsHazmat {
			continue
		}
		rule := findMatchingHazmatRuleByClass(rules, cand.commodity, existing.HazmatClass)
		if rule == nil || rule.SegregationType != hazmatsegregationrule.SegregationTypeDistance || rule.MinimumDistance == nil {
			continue
		}

		requiredFt := convertDistanceToFeet(*rule.MinimumDistance, rule.DistanceUnit)
		minStart := existing.PositionFeet + existing.LengthFeet + requiredFt
		position = max(position, minStart)
	}

	return position
}

func buildPlacement(cand placementCandidate, position float64) repositories.CommodityPlacement {
	p := repositories.CommodityPlacement{
		CommodityID:         cand.commodity.ID,
		CommodityName:       cand.commodity.Name,
		PositionFeet:        roundTo2(position),
		LengthFeet:          roundTo2(cand.lengthFt),
		Weight:              cand.input.Weight,
		Pieces:              cand.input.Pieces,
		Stackable:           cand.commodity.Stackable,
		Fragile:             cand.commodity.Fragile,
		IsHazmat:            cand.isHazmat,
		MinTemp:             cand.commodity.MinTemperature,
		MaxTemp:             cand.commodity.MaxTemperature,
		LoadingInstructions: cand.commodity.LoadingInstructions,
		EstimatedLength:     cand.estimated,
		StopNumber:          cand.stopNumber,
	}

	if cand.isHazmat && cand.commodity.HazardousMaterial != nil {
		p.HazmatClass = string(cand.commodity.HazardousMaterial.Class)
	}

	return p
}

// --- Stop Dividers ---

func computeStopDividers(placements []repositories.CommodityPlacement, stops []repositories.StopInfo) []repositories.StopDivider {
	if len(stops) == 0 {
		return nil
	}

	stopLabelMap := make(map[int]string, len(stops))
	for _, s := range stops {
		label := s.LocationCity
		if label == "" {
			label = s.LocationName
		}
		stopLabelMap[s.Sequence+1] = label
	}

	type stopBound struct {
		maxEnd float64
		label  string
	}
	bounds := make(map[int]*stopBound)

	for _, p := range placements {
		if p.StopNumber == 0 {
			continue
		}
		end := p.PositionFeet + p.LengthFeet
		b, ok := bounds[p.StopNumber]
		if !ok {
			bounds[p.StopNumber] = &stopBound{maxEnd: end, label: stopLabelMap[p.StopNumber]}
		} else if end > b.maxEnd {
			b.maxEnd = end
		}
	}

	dividers := make([]repositories.StopDivider, 0, len(bounds))
	for stopNum, b := range bounds {
		dividers = append(dividers, repositories.StopDivider{
			PositionFeet: roundTo2(b.maxEnd),
			StopNumber:   stopNum,
			Label:        fmt.Sprintf("Stop %d: %s", stopNum, b.label),
		})
	}

	slices.SortFunc(dividers, func(a, b repositories.StopDivider) int {
		if a.PositionFeet < b.PositionFeet {
			return -1
		}
		if a.PositionFeet > b.PositionFeet {
			return 1
		}
		return 0
	})

	return dividers
}

// --- Hazmat Evaluation ---

func findMatchingHazmatRuleByClass(
	rules []*hazmatsegregationrule.HazmatSegregationRule,
	com *commodity.Commodity,
	otherClass string,
) *hazmatsegregationrule.HazmatSegregationRule {
	if com.HazardousMaterial == nil || otherClass == "" {
		return nil
	}

	comClass := string(com.HazardousMaterial.Class)
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		classA := string(rule.ClassA)
		classB := string(rule.ClassB)

		if (classA == comClass && classB == otherClass) ||
			(classA == otherClass && classB == comClass) {
			return rule
		}
	}

	return nil
}

func evaluateHazmatZones(
	placements []repositories.CommodityPlacement,
	commodityByID map[pulid.ID]*commodity.Commodity,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
) []repositories.HazmatZoneResult {
	hazmatPlacements := filterHazmatPlacements(placements)
	zones := make([]repositories.HazmatZoneResult, 0)

	for i := range hazmatPlacements {
		for j := i + 1; j < len(hazmatPlacements); j++ {
			a, b := hazmatPlacements[i], hazmatPlacements[j]

			comA := commodityByID[a.CommodityID]
			comB := commodityByID[b.CommodityID]
			if comA == nil || comB == nil {
				continue
			}

			rule := findMatchingHazmatRule(rules, comA, comB)
			if rule == nil {
				continue
			}

			actualDist := distanceBetweenPlacements(a, b)
			zone := repositories.HazmatZoneResult{
				CommodityAID:       a.CommodityID,
				CommodityBID:       b.CommodityID,
				CommodityAName:     a.CommodityName,
				CommodityBName:     b.CommodityName,
				RuleName:           rule.Name,
				SegregationType:    string(rule.SegregationType),
				ActualDistanceFeet: roundTo2(actualDist),
				Satisfied:          true,
			}

			switch rule.SegregationType {
			case hazmatsegregationrule.SegregationTypeProhibited:
				zone.Satisfied = false
			case hazmatsegregationrule.SegregationTypeDistance:
				if rule.MinimumDistance != nil {
					requiredFt := convertDistanceToFeet(*rule.MinimumDistance, rule.DistanceUnit)
					zone.RequiredDistanceFeet = &requiredFt
					zone.Satisfied = actualDist >= requiredFt
				}
			case hazmatsegregationrule.SegregationTypeSeparated:
				zone.Satisfied = actualDist > 0
			case hazmatsegregationrule.SegregationTypeBarrier:
				zone.Satisfied = hasBarrierBetween(placements, a, b)
			}

			zones = append(zones, zone)
		}
	}

	return zones
}

func filterHazmatPlacements(placements []repositories.CommodityPlacement) []repositories.CommodityPlacement {
	result := make([]repositories.CommodityPlacement, 0)
	for _, p := range placements {
		if p.IsHazmat {
			result = append(result, p)
		}
	}
	return result
}

func distanceBetweenPlacements(a, b repositories.CommodityPlacement) float64 {
	aEnd := a.PositionFeet + a.LengthFeet
	bStart := b.PositionFeet
	dist := bStart - aEnd
	if dist < 0 {
		bEnd := b.PositionFeet + b.LengthFeet
		dist = a.PositionFeet - bEnd
	}
	return max(dist, 0)
}

func hasBarrierBetween(placements []repositories.CommodityPlacement, a, b repositories.CommodityPlacement) bool {
	for _, p := range placements {
		if p.IsHazmat || p.CommodityID == a.CommodityID || p.CommodityID == b.CommodityID {
			continue
		}
		pEnd := p.PositionFeet + p.LengthFeet
		if (p.PositionFeet > a.PositionFeet && pEnd < b.PositionFeet) ||
			(p.PositionFeet > b.PositionFeet && pEnd < a.PositionFeet) {
			return true
		}
	}
	return false
}

// --- Warnings ---

func generateWarnings(
	placements []repositories.CommodityPlacement,
	commodityByID map[pulid.ID]*commodity.Commodity,
	totalLinearFeet float64,
	totalWeight int64,
	trailerLength float64,
	maxWeight int64,
	hazmatZones []repositories.HazmatZoneResult,
) []repositories.LoadingWarning {
	warnings := make([]repositories.LoadingWarning, 0)

	if totalWeight > maxWeight {
		warnings = append(warnings, repositories.LoadingWarning{
			Type:     "overweight",
			Message:  fmt.Sprintf("Total weight %s lbs exceeds maximum %s lbs", intutils.FormatWithCommas(totalWeight), intutils.FormatWithCommas(maxWeight)),
			Severity: "error",
		})
	}

	if totalLinearFeet > trailerLength {
		warnings = append(warnings, repositories.LoadingWarning{
			Type:     "over_length",
			Message:  fmt.Sprintf("Total linear feet %.1f exceeds trailer capacity %.1f ft", totalLinearFeet, trailerLength),
			Severity: "error",
		})
	}

	for _, zone := range hazmatZones {
		if !zone.Satisfied {
			warnings = append(warnings, repositories.LoadingWarning{
				Type: "hazmat_violation",
				Message: fmt.Sprintf(
					"Hazmat rule %q (%s) violated between %s and %s",
					zone.RuleName, zone.SegregationType, zone.CommodityAName, zone.CommodityBName,
				),
				Severity:     "error",
				CommodityIDs: []string{zone.CommodityAID.String(), zone.CommodityBID.String()},
			})
		}
	}

	warnings = append(warnings, checkTemperatureConflicts(placements)...)
	warnings = append(warnings, checkFragileStacking(placements)...)
	warnings = append(warnings, checkEstimatedLengths(placements, commodityByID)...)

	return warnings
}

func checkTemperatureConflicts(placements []repositories.CommodityPlacement) []repositories.LoadingWarning {
	var warnings []repositories.LoadingWarning

	for i := range placements {
		for j := i + 1; j < len(placements); j++ {
			a, b := placements[i], placements[j]
			if a.MinTemp == nil || a.MaxTemp == nil || b.MinTemp == nil || b.MaxTemp == nil {
				continue
			}

			dist := distanceBetweenPlacements(a, b)
			if dist > 1 {
				continue
			}

			if *a.MaxTemp < *b.MinTemp || *b.MaxTemp < *a.MinTemp {
				warnings = append(warnings, repositories.LoadingWarning{
					Type: "temperature_conflict",
					Message: fmt.Sprintf(
						"%s (%d\u2013%d\u00b0F) and %s (%d\u2013%d\u00b0F) have incompatible temperature ranges and are placed adjacent",
						a.CommodityName, *a.MinTemp, *a.MaxTemp,
						b.CommodityName, *b.MinTemp, *b.MaxTemp,
					),
					Severity:     "warning",
					CommodityIDs: []string{a.CommodityID.String(), b.CommodityID.String()},
				})
			}
		}
	}

	return warnings
}

func checkFragileStacking(placements []repositories.CommodityPlacement) []repositories.LoadingWarning {
	var warnings []repositories.LoadingWarning

	for i := range placements {
		if !placements[i].Fragile {
			continue
		}

		for j := i + 1; j < len(placements); j++ {
			if !placements[j].Stackable && placements[j].Weight > 2000 {
				dist := distanceBetweenPlacements(placements[i], placements[j])
				if dist < 1 {
					warnings = append(warnings, repositories.LoadingWarning{
						Type: "fragile_stacking",
						Message: fmt.Sprintf(
							"Fragile item %q is adjacent to heavy non-stackable item %q (%s lbs) \u2014 risk of damage during transit",
							placements[i].CommodityName, placements[j].CommodityName, intutils.FormatWithCommas(placements[j].Weight),
						),
						Severity:     "warning",
						CommodityIDs: []string{placements[i].CommodityID.String(), placements[j].CommodityID.String()},
					})
				}
			}
		}
	}

	return warnings
}

func checkEstimatedLengths(placements []repositories.CommodityPlacement, commodityByID map[pulid.ID]*commodity.Commodity) []repositories.LoadingWarning {
	var warnings []repositories.LoadingWarning

	for _, p := range placements {
		if !p.EstimatedLength {
			continue
		}
		name := p.CommodityName
		if com := commodityByID[p.CommodityID]; com != nil {
			name = com.Name
		}
		warnings = append(warnings, repositories.LoadingWarning{
			Type:         "missing_linear_feet",
			Message:      fmt.Sprintf("Linear feet per unit not configured for %q \u2014 using estimated length", name),
			Severity:     "info",
			CommodityIDs: []string{p.CommodityID.String()},
		})
	}

	return warnings
}

// --- Axle Weights ---

func computeAxleWeights(placements []repositories.CommodityPlacement) []repositories.AxleWeight {
	var driveWeight, trailerWeight float64

	for _, p := range placements {
		cog := p.PositionFeet + (p.LengthFeet / 2)
		distFromKingpin := max(cog-kingpinOffsetFt, 0)

		trailerPortion := min(distFromKingpin/rearAxleFromKingpinFt, 1)
		w := float64(p.Weight)
		trailerWeight += w * trailerPortion
		driveWeight += w * (1 - trailerPortion)
	}

	driveW := int64(math.Round(driveWeight))
	trailerW := int64(math.Round(trailerWeight))

	return []repositories.AxleWeight{
		{Axle: "steer", Weight: 0, Limit: steerAxleLimitLbs, Percentage: 0, Compliant: true},
		{Axle: "drive", Weight: driveW, Limit: driveAxleLimitLbs, Percentage: safePercent(float64(driveW), float64(driveAxleLimitLbs)), Compliant: driveW <= driveAxleLimitLbs},
		{Axle: "trailer", Weight: trailerW, Limit: trailerAxleLimitLbs, Percentage: safePercent(float64(trailerW), float64(trailerAxleLimitLbs)), Compliant: trailerW <= trailerAxleLimitLbs},
	}
}

// --- Utilization ---

func computeUtilizationScore(linearFeetUtil, weightUtil float64) (int, string) {
	score := intutils.Clamp(int(min(linearFeetUtil, weightUtil)), 0, 100)

	var grade string
	switch {
	case score >= 85:
		grade = "Excellent"
	case score >= 70:
		grade = "Good"
	case score >= 50:
		grade = "Fair"
	default:
		grade = "Poor"
	}

	return score, grade
}

// --- Recommendations ---

func generateRecommendations(
	placements []repositories.CommodityPlacement,
	axleWeights []repositories.AxleWeight,
	hazmatZones []repositories.HazmatZoneResult,
	totalLinearFeet float64,
	totalWeight int64,
	trailerLength float64,
	maxWeight int64,
) []repositories.LoadingRecommendation {
	var recs []repositories.LoadingRecommendation

	recs = append(recs, checkOverweight(placements, totalWeight, maxWeight)...)
	recs = append(recs, checkOverLength(placements, totalLinearFeet, trailerLength)...)
	recs = append(recs, checkAxleViolations(axleWeights)...)
	recs = append(recs, checkHazmatViolations(hazmatZones)...)
	recs = append(recs, checkWeightBalance(axleWeights, totalWeight)...)
	recs = append(recs, checkUtilization(totalWeight, maxWeight, totalLinearFeet, trailerLength)...)
	recs = append(recs, checkMissingDimensions(placements)...)

	return recs
}

func checkOverweight(placements []repositories.CommodityPlacement, totalWeight, maxWeight int64) []repositories.LoadingRecommendation {
	if totalWeight <= maxWeight {
		return nil
	}

	excess := totalWeight - maxWeight
	heaviest := findHeaviest(placements)

	return []repositories.LoadingRecommendation{{
		Type:     "split",
		Priority: "critical",
		Title:    "Shipment exceeds weight limit",
		Description: fmt.Sprintf(
			"Remove %q (%s lbs) to reduce weight by %s lbs, or split into multiple trailers.",
			heaviest.CommodityName, intutils.FormatWithCommas(heaviest.Weight), intutils.FormatWithCommas(excess),
		),
		Impact:       "Prevents DOT overweight fine ($1,000\u2013$16,000+)",
		CommodityIDs: []string{heaviest.CommodityID.String()},
	}}
}

func checkOverLength(placements []repositories.CommodityPlacement, totalLinearFeet, trailerLength float64) []repositories.LoadingRecommendation {
	if totalLinearFeet <= trailerLength {
		return nil
	}

	longest := findLongest(placements)

	return []repositories.LoadingRecommendation{{
		Type:     "split",
		Priority: "critical",
		Title:    "Cargo exceeds trailer length",
		Description: fmt.Sprintf(
			"Remove %q (%.1f ft) to fit within %.0f ft trailer capacity.",
			longest.CommodityName, longest.LengthFeet, trailerLength,
		),
		Impact:       "Required \u2014 cargo physically cannot fit",
		CommodityIDs: []string{longest.CommodityID.String()},
	}}
}

func checkAxleViolations(axleWeights []repositories.AxleWeight) []repositories.LoadingRecommendation {
	var recs []repositories.LoadingRecommendation

	for _, aw := range axleWeights {
		if aw.Compliant || aw.Weight == 0 {
			continue
		}
		excess := aw.Weight - aw.Limit
		recs = append(recs, repositories.LoadingRecommendation{
			Type:     "reorder",
			Priority: "critical",
			Title:    fmt.Sprintf("%s axle overweight", stringutils.CapitalizeFirst(aw.Axle)),
			Description: fmt.Sprintf(
				"%s axle at %s lbs exceeds %s lb limit by %s lbs. Shift heavy items toward the opposite end of the trailer.",
				stringutils.CapitalizeFirst(aw.Axle), intutils.FormatWithCommas(aw.Weight), intutils.FormatWithCommas(aw.Limit), intutils.FormatWithCommas(excess),
			),
			Impact: "Prevents weigh station fine and improves vehicle handling",
		})
	}

	return recs
}

func checkHazmatViolations(hazmatZones []repositories.HazmatZoneResult) []repositories.LoadingRecommendation {
	var recs []repositories.LoadingRecommendation

	for _, zone := range hazmatZones {
		if zone.Satisfied {
			continue
		}
		desc := fmt.Sprintf(
			"%s and %s violate %q (%s).",
			zone.CommodityAName, zone.CommodityBName, zone.RuleName, zone.SegregationType,
		)
		if zone.RequiredDistanceFeet != nil {
			if gap := *zone.RequiredDistanceFeet - zone.ActualDistanceFeet; gap > 0 {
				desc += fmt.Sprintf(" Increase separation by %.1f ft.", gap)
			}
		}
		recs = append(recs, repositories.LoadingRecommendation{
			Type:         "reorder",
			Priority:     "critical",
			Title:        "Hazmat segregation violation",
			Description:  desc,
			Impact:       "Required by DOT 49 CFR 177.848",
			CommodityIDs: []string{zone.CommodityAID.String(), zone.CommodityBID.String()},
		})
	}

	return recs
}

func checkWeightBalance(axleWeights []repositories.AxleWeight, totalWeight int64) []repositories.LoadingRecommendation {
	if len(axleWeights) < 3 || totalWeight == 0 {
		return nil
	}

	drive, trailer := axleWeights[1], axleWeights[2]
	if drive.Weight == 0 || trailer.Weight == 0 {
		return nil
	}

	ratio := float64(drive.Weight) / float64(drive.Weight+trailer.Weight)

	switch {
	case ratio > 0.65:
		return []repositories.LoadingRecommendation{{
			Type:        "reorder",
			Priority:    "suggested",
			Title:       "Weight is front-heavy",
			Description: "Most cargo weight is near the nose. Move heavier items toward the rear to improve axle balance and tire wear.",
			Impact:      "Improves fuel efficiency and tire longevity",
		}}
	case ratio < 0.35:
		return []repositories.LoadingRecommendation{{
			Type:        "reorder",
			Priority:    "suggested",
			Title:       "Weight is rear-heavy",
			Description: "Most cargo weight is near the doors. Move heavier items toward the nose to improve traction on the drive axle.",
			Impact:      "Improves traction and braking performance",
		}}
	default:
		return nil
	}
}

func checkUtilization(totalWeight, maxWeight int64, totalLinearFeet, trailerLength float64) []repositories.LoadingRecommendation {
	if totalWeight == 0 || maxWeight == 0 {
		return nil
	}

	weightUtil := safePercent(float64(totalWeight), float64(maxWeight))
	linearUtil := safePercent(totalLinearFeet, trailerLength)

	if weightUtil < 50 && linearUtil < 50 {
		return []repositories.LoadingRecommendation{{
			Type:        "mode_change",
			Priority:    "optimization",
			Title:       "Trailer underutilized",
			Description: fmt.Sprintf("Only %.0f%% of weight and %.0f%% of space used. Consider consolidating with another shipment or switching to LTL.", weightUtil, linearUtil),
			Impact:      "Potential cost savings through consolidation",
		}}
	}

	return nil
}

func checkMissingDimensions(placements []repositories.CommodityPlacement) []repositories.LoadingRecommendation {
	var names []string
	var ids []string

	for _, p := range placements {
		if p.EstimatedLength {
			names = append(names, p.CommodityName)
			ids = append(ids, p.CommodityID.String())
		}
	}

	if len(names) == 0 {
		return nil
	}

	desc := fmt.Sprintf(
		"Linear feet per unit is not configured for %s. Space is estimated from weight \u2014 configure actual values in commodity settings for accurate planning.",
		joinNames(names),
	)

	return []repositories.LoadingRecommendation{{
		Type:         "configure",
		Priority:     "optimization",
		Title:        "Missing commodity dimensions",
		Description:  desc,
		CommodityIDs: ids,
	}}
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return fmt.Sprintf("%s and %d more", names[0], len(names)-1)
	}
}

// --- Helpers ---

func findHeaviest(placements []repositories.CommodityPlacement) repositories.CommodityPlacement {
	var heaviest repositories.CommodityPlacement
	for _, p := range placements {
		if p.Weight > heaviest.Weight {
			heaviest = p
		}
	}
	return heaviest
}

func findLongest(placements []repositories.CommodityPlacement) repositories.CommodityPlacement {
	var longest repositories.CommodityPlacement
	for _, p := range placements {
		if p.LengthFeet > longest.LengthFeet {
			longest = p
		}
	}
	return longest
}

func computeTotals(placements []repositories.CommodityPlacement) (totalLinearFeet float64, totalWeight int64) {
	for _, p := range placements {
		end := p.PositionFeet + p.LengthFeet
		totalLinearFeet = max(totalLinearFeet, end)
		totalWeight += p.Weight
	}
	return totalLinearFeet, totalWeight
}

func convertDistanceToFeet(distance float64, unit string) float64 {
	switch unit {
	case "M":
		return distance * 3.28084
	case "IN":
		return distance / 12.0
	case "CM":
		return distance / 30.48
	default:
		return distance
	}
}

func safePercent(value, total float64) float64 {
	if total == 0 {
		return 0
	}
	return roundTo2((value / total) * 100)
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}
