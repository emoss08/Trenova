package rateresolver

import (
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func rateMatrixFromInput(
	input *gqlmodel.RateMatrixInput,
	authCtx *authctx.AuthContext,
) *ratematrix.RateMatrix {
	entity := &ratematrix.RateMatrix{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Code:           input.Code,
		Name:           input.Name,
		Description:    gqlscalar.StringValue(input.Description),
		ValueType:      input.ValueType,
		Dimensions:     make([]*ratematrix.RateMatrixDimension, 0, len(input.Dimensions)),
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
	for _, dimension := range input.Dimensions {
		if dimension == nil {
			continue
		}
		entity.Dimensions = append(entity.Dimensions, &ratematrix.RateMatrixDimension{
			Position:  int16(dimension.Position),
			Kind:      ratematrix.DimensionKind(dimension.Kind),
			MatchMode: ratematrix.MatchMode(dimension.MatchMode),
			Label:     gqlscalar.StringValue(dimension.Label),
			KeyNormalization: ratematrix.KeyNormalization(
				gqlscalar.StringValue(dimension.KeyNormalization),
			),
			RangeOverflow: ratematrix.RangeOverflow(gqlscalar.StringValue(dimension.RangeOverflow)),
		})
	}

	return entity
}

func rateMatrixCellsFromInput(
	inputs []*gqlmodel.RateMatrixCellInput,
) ([]*ratematrixcell.RateMatrixCell, error) {
	cells := make([]*ratematrixcell.RateMatrixCell, 0, len(inputs))
	for index, input := range inputs {
		if input == nil {
			continue
		}
		cell, err := rateMatrixCellFromInput(input)
		if err != nil {
			return nil, indexedCellError(index, err)
		}
		cells = append(cells, cell)
	}

	return cells, nil
}

func indexedCellError(index int, err error) error {
	var validation *errortypes.Error
	if !errors.As(err, &validation) {
		return err
	}
	multiErr := errortypes.NewMultiError()
	multiErr.WithIndex("cells", index).AddError(validation)

	return multiErr
}

type cellBand struct {
	field string
	raw   *string
	dest  *decimal.NullDecimal
}

func rateMatrixCellFromInput(
	input *gqlmodel.RateMatrixCellInput,
) (*ratematrixcell.RateMatrixCell, error) {
	value, err := gqldecimal.ParseDecimalField("value", input.Value, true)
	if err != nil {
		return nil, err
	}
	cell := &ratematrixcell.RateMatrixCell{
		D0Key:           gqlscalar.StringValue(input.D0Key),
		D1Key:           gqlscalar.StringValue(input.D1Key),
		D2Key:           gqlscalar.StringValue(input.D2Key),
		D3Key:           gqlscalar.StringValue(input.D3Key),
		D0KeyTo:         gqlscalar.StringValue(input.D0KeyTo),
		D1KeyTo:         gqlscalar.StringValue(input.D1KeyTo),
		D2KeyTo:         gqlscalar.StringValue(input.D2KeyTo),
		D3KeyTo:         gqlscalar.StringValue(input.D3KeyTo),
		Value:           value,
		DeficitEligible: input.DeficitEligible == nil || *input.DeficitEligible,
	}

	bands := []cellBand{
		{field: "d0Min", raw: input.D0Min, dest: &cell.D0Min},
		{field: "d0Max", raw: input.D0Max, dest: &cell.D0Max},
		{field: "d1Min", raw: input.D1Min, dest: &cell.D1Min},
		{field: "d1Max", raw: input.D1Max, dest: &cell.D1Max},
		{field: "d2Min", raw: input.D2Min, dest: &cell.D2Min},
		{field: "d2Max", raw: input.D2Max, dest: &cell.D2Max},
		{field: "d3Min", raw: input.D3Min, dest: &cell.D3Min},
		{field: "d3Max", raw: input.D3Max, dest: &cell.D3Max},
		{field: "minCharge", raw: input.MinCharge, dest: &cell.MinCharge},
	}
	for _, band := range bands {
		parsed, parseErr := gqldecimal.ParseNullDecimalField(band.field, band.raw)
		if parseErr != nil {
			return nil, parseErr
		}
		*band.dest = parsed
	}

	return cell, nil
}
