package rateengine

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

// ErrMatrixNotFound is returned when a rule points at a matrix that is gone or
// inactive. The foreign key stops it being deleted, so in practice this means
// deactivated.
var ErrMatrixNotFound = errors.New("rate matrix is unavailable")

// matrixCharge prices a rule against a multi-axis tariff.
//
// The axes are filled from the shipment in the order the matrix declares them,
// the database narrows to the handful of cells that could match, and the domain
// picks the winner. Nothing loads the matrix whole; a class tariff runs to
// hundreds of thousands of cells and this is the hot path of every shipment
// write.
func (s *Service) matrixCharge(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	matrix, err := s.matrixRepo.GetByID(ctx, &repositories.GetRateMatrixByIDRequest{
		RateMatrixID:      *rule.RateMatrixID,
		TenantInfo:        rateCtx.TenantInfo,
		IncludeDimensions: true,
	})
	if err != nil {
		return decimal.Zero, err
	}

	dimensions := matrix.OrderedDimensions()
	if len(dimensions) == 0 {
		return decimal.Zero, ErrMatrixNotFound
	}

	values, axes := s.buildAxes(ctx, rateCtx, rule, dimensions, trace)

	cells, err := s.matrixRepo.LookupCells(ctx, &repositories.LookupRateMatrixCellsRequest{
		TenantInfo:   rateCtx.TenantInfo,
		RateMatrixID: matrix.ID,
		Axes:         axes,
	})
	if err != nil {
		return decimal.Zero, err
	}

	match, err := ratematrix.SelectCell(dimensions, cells, &values)
	if err != nil {
		return decimal.Zero, err
	}

	return s.applyCellValue(ctx, rateCtx, matrix, match, trace)
}

// buildAxes fills each declared axis from the shipment.
//
// Exact axes get the candidate keys in preference order, because a place can
// belong to several zones and the narrower one has to win. Banded axes get the
// quantity, and the band is settled after the fetch.
func (s *Service) buildAxes(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	dimensions []*ratematrix.RateMatrixDimension,
	trace *ratetypes.Trace,
) (ratematrix.LookupValues, []repositories.MatrixAxisQuery) {
	var values ratematrix.LookupValues
	axes := make([]repositories.MatrixAxisQuery, 0, len(dimensions))

	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}

		value := s.axisValue(ctx, rateCtx, rule, dimension, trace)
		values[dimension.Position] = value

		axes = append(axes, repositories.MatrixAxisQuery{
			Position: dimension.Position,
			Keys:     value.Keys,
			Quantity: value.Quantity,
		})
	}

	return values, axes
}

func (s *Service) axisValue(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	dimension *ratematrix.RateMatrixDimension,
	trace *ratetypes.Trace,
) ratematrix.DimensionValue {
	switch dimension.Kind {
	case ratematrix.DimensionKindZone:
		return ratematrix.DimensionValue{Keys: zoneKeys(rateCtx, dimension.Position)}
	case ratematrix.DimensionKindZip3, ratematrix.DimensionKindZip5,
		ratematrix.DimensionKindState, ratematrix.DimensionKindCountry:
		return ratematrix.DimensionValue{
			Keys: placeKeys(rateCtx, dimension),
		}
	case ratematrix.DimensionKindWeightBreak:
		return ratematrix.DimensionValue{
			Quantity: decimal.NewNullDecimal(rateCtx.Weight),
		}
	case ratematrix.DimensionKindDistance:
		return ratematrix.DimensionValue{
			Quantity: decimal.NewNullDecimal(rateCtx.Distance),
		}
	case ratematrix.DimensionKindPieceCount:
		return ratematrix.DimensionValue{
			Quantity: decimal.NewNullDecimal(decimal.NewFromInt(rateCtx.Pieces)),
		}
	case ratematrix.DimensionKindLinearFeet:
		return ratematrix.DimensionValue{
			Quantity: decimal.NewNullDecimal(rateCtx.LinearFeet),
		}
	case ratematrix.DimensionKindFreightClass:
		class := s.resolveFreightClass(ctx, rateCtx, rule, trace)
		if class == "" {
			return ratematrix.DimensionValue{}
		}
		return ratematrix.DimensionValue{Keys: []string{class.String()}}
	case ratematrix.DimensionKindEquipmentType:
		return ratematrix.DimensionValue{Keys: equipmentKeys(rateCtx)}
	case ratematrix.DimensionKindServiceType:
		return ratematrix.DimensionValue{Keys: []string{rateCtx.ServiceTypeID.String()}}
	case ratematrix.DimensionKindCustom:
		return ratematrix.DimensionValue{}
	default:
		return ratematrix.DimensionValue{}
	}
}

// zoneKeys reads the zones of whichever end of the lane the axis describes.
//
// A matrix keyed by zone on both ends declares the origin axis before the
// destination one, which is the convention every rate sheet follows: origin
// reads left to right before destination.
func zoneKeys(rateCtx *RateContext, position int16) []string {
	place := rateCtx.Origin
	if position > 0 {
		place = rateCtx.Destination
	}

	keys := make([]string, 0, len(place.ZoneIDs))
	for _, zoneID := range place.ZoneIDs {
		keys = append(keys, zoneID.String())
	}

	return keys
}

func placeKeys(
	rateCtx *RateContext,
	dimension *ratematrix.RateMatrixDimension,
) []string {
	place := rateCtx.Origin
	if dimension.Position > 0 {
		place = rateCtx.Destination
	}

	scope := rategeo.Scope{Value: place.PostalCode}

	switch dimension.Kind { //nolint:exhaustive // the other kinds are not geographic
	case ratematrix.DimensionKindZip5:
		scope.Type = rategeo.ScopeTypeZip5
	case ratematrix.DimensionKindZip3:
		scope.Type = rategeo.ScopeTypeZip3
	case ratematrix.DimensionKindState:
		scope = rategeo.Scope{Type: rategeo.ScopeTypeState, Value: place.StateID.String()}
	case ratematrix.DimensionKindCountry:
		scope = rategeo.Scope{Type: rategeo.ScopeTypeCountry, Value: place.CountryISO3}
	default:
		return nil
	}

	key, ok := scope.Key()
	if !ok {
		return nil
	}

	return []string{key}
}

// equipmentKeys offers the trailer before the tractor. A tariff that varies by
// equipment is nearly always describing the trailer that carries the freight.
func equipmentKeys(rateCtx *RateContext) []string {
	keys := make([]string, 0, 2)

	if !rateCtx.TrailerTypeID.IsNil() {
		keys = append(keys, rateCtx.TrailerTypeID.String())
	}
	if !rateCtx.TractorTypeID.IsNil() {
		keys = append(keys, rateCtx.TractorTypeID.String())
	}

	return keys
}

// applyCellValue turns a cell's number into money by evaluating the matrix's
// formula template with the cell's value bound as its base rate. What a cell
// means — a per-mile rate, a flat amount — is whatever the matrix's template
// does with it.
func (s *Service) applyCellValue(
	ctx context.Context,
	rateCtx *RateContext,
	matrix *ratematrix.RateMatrix,
	match *ratematrix.Match,
	trace *ratetypes.Trace,
) (decimal.Decimal, error) {
	cell := match.Cell

	resp, err := s.formula.Calculate(ctx, &formulatemplatetypes.CalculateRequest{
		TemplateID: matrix.FormulaTemplateID,
		Entity:     rateCtx.Entity,
		Overrides:  map[string]any{"baseRate": cell.Value.InexactFloat64()},
		TenantInfo: rateCtx.TenantInfo,
		RatingDate: rateCtx.AsOf,
	})
	if err != nil {
		return decimal.Zero, err
	}

	amount := resp.Amount
	if cell.MinCharge.Valid && amount.LessThan(cell.MinCharge.Decimal) {
		trace.Guardrails = append(trace.Guardrails, ratetypes.Guardrail{
			Kind:    ratetypes.ComponentKindMinimumCharge,
			Applied: true,
			Bound:   cell.MinCharge.Decimal,
			Raw:     amount,
			Result:  cell.MinCharge.Decimal,
		})
		amount = cell.MinCharge.Decimal
	}

	// The keys actually read are recorded, not just the cell id, so a reader
	// can see which zone and which break the rate came from without going and
	// looking the cell up.
	detail := map[string]any{
		"cellId":          cell.ID.String(),
		"matchedKeys":     nonEmpty(match.MatchedKeys[:]),
		"formulaTemplate": resp.FormulaTemplateName,
	}
	recordFormulaResponse(resp, detail, trace)

	trace.AddComponent(&ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      linehaulLabel,
		Basis:      resp.FormulaTemplateName + " @ " + cell.Value.String(),
		Rate:       decimal.NewNullDecimal(cell.Value),
		Amount:     amount,
		Source:     ratetypes.ComponentSourceRateMatrix,
		SourceID:   matrix.ID.String(),
		SourceName: matrix.Name,
		Detail:     detail,
	})

	return amount, nil
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}

	return out
}

// resolveFreightClass decides which class a class-rated rule prices at.
//
// The 2025 NMFC restructure moved most freight off fixed classifications onto a
// density scale, so a class is now something derived from weight and cube as
// often as it is something read off a commodity. Whichever way it was reached,
// the class and its provenance are recorded: a class dispute is the most common
// LTL billing adjustment, and "we measured 8.2 pcf, which classifies as 92.5"
// is the whole of the argument.
func (s *Service) resolveFreightClass(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) commodity.FreightClass {
	switch rule.FreightClassSource { //nolint:exhaustive // the commodity source is the default arm
	case rateagreement.FreightClassSourceFixed:
		trace.Inputs.FreightClassUsed = rule.FixedFreightClass.String()
		trace.Inputs.FreightClassSource = rule.FreightClassSource.String()

		return rule.FixedFreightClass

	case rateagreement.FreightClassSourceDensity:
		if class, ok := s.densityClass(ctx, rateCtx, rule, trace); ok {
			return class
		}

		// Dimensions go missing on entry all the time. Falling back to the
		// commodity's own class is what a rater would do by hand, and the
		// warning says so rather than leaving a surprising number unexplained.
		trace.Warn("Density could not be measured; classified from the commodity instead")
	}

	if len(rateCtx.FreightClasses) > 0 {
		class := rateCtx.FreightClasses[0]
		trace.Inputs.FreightClassUsed = class.String()
		trace.Inputs.FreightClassSource = rateagreement.FreightClassSourceCommodity.String()

		return class
	}

	return ""
}

func (s *Service) densityClass(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
	trace *ratetypes.Trace,
) (commodity.FreightClass, bool) {
	density, ok := rateagreement.Density(rateCtx.Weight, rateCtx.CubicFeet)
	if !ok {
		return "", false
	}

	scale, err := s.loadDensityScale(ctx, rateCtx, rule)
	if err != nil || scale == nil {
		return "", false
	}

	class, _, ok := scale.ClassFor(density)
	if !ok {
		return "", false
	}

	trace.Inputs.DensityPcf = decimal.NewNullDecimal(density)
	trace.Inputs.FreightClassUsed = class.String()
	trace.Inputs.FreightClassSource = rateagreement.FreightClassSourceDensity.String()

	return class, true
}

func (s *Service) loadDensityScale(
	ctx context.Context,
	rateCtx *RateContext,
	rule *rateagreement.RateAgreementRule,
) (*ratematrix.DensityScale, error) {
	if rule.DensityScaleID != nil && !rule.DensityScaleID.IsNil() {
		return s.densityRepo.GetByID(ctx, &repositories.GetDensityScaleRequest{
			DensityScaleID: *rule.DensityScaleID,
			TenantInfo:     rateCtx.TenantInfo,
		})
	}

	return s.densityRepo.GetOrgDefault(ctx, rateCtx.TenantInfo)
}
