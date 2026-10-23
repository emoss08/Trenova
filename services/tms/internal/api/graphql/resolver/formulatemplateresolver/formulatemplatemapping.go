package formulatemplateresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
)

func formulaTemplateColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FormulaTemplateSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func formulaTemplateConnectionToModel(
	result *pagination.CursorListResult[*formulatemplate.FormulaTemplate],
) (*gqlmodel.FormulaTemplateConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *formulatemplate.FormulaTemplate, cursor string) *gqlmodel.FormulaTemplateEdge {
			return &gqlmodel.FormulaTemplateEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FormulaTemplateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FormulaTemplateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func formulaInputDeclarationsToModel(
	inputs []tarifftypes.InputDecl,
) ([]*gqlmodel.FormulaInputDeclaration, error) {
	items := make([]*gqlmodel.FormulaInputDeclaration, 0, len(inputs))
	for i := range inputs {
		input := &inputs[i]
		descriptor, err := decodeWire(input.Descriptor)
		if err != nil {
			return nil, err
		}
		defaultValue, err := decodeWire(input.Default)
		if err != nil {
			return nil, err
		}
		constraint, err := constraintToModel(input.Constraint)
		if err != nil {
			return nil, err
		}
		items = append(items, &gqlmodel.FormulaInputDeclaration{
			Name:       input.Name,
			Doc:        input.Doc,
			Type:       input.Type,
			Descriptor: descriptor,
			Required:   input.Required,
			Default:    defaultValue,
			Constraint: constraint,
		})
	}
	return items, nil
}

func constraintToModel(constraint *rules.TariffConstraint) (any, error) {
	if constraint == nil {
		return nil, nil //nolint:nilnil // an input without a where clause accepts every value
	}
	encoded, err := sonic.Marshal(constraint)
	if err != nil {
		return nil, err
	}
	return decodeWire(encoded)
}

func decodeWire(raw rules.RawJSON) (any, error) {
	var decoded any
	if len(raw) == 0 {
		return decoded, nil
	}
	err := sonic.Unmarshal(raw, &decoded)
	return decoded, err
}

func pinnedMatrix(
	ctx context.Context,
	pin *tarifftypes.TablePin,
) (*ratematrix.RateMatrix, error) {
	if pin == nil || pin.RateMatrixID.IsNil() || !loaders.Available(ctx) {
		return nil, nil
	}
	return loaders.RateMatrixByID(ctx, pin.RateMatrixID.String())
}

func formulaTemplateStats(
	ctx context.Context,
	obj *formulatemplate.FormulaTemplate,
) (formulatemplateports.TemplateStats, error) {
	if !loaders.Available(ctx) {
		return formulatemplateports.TemplateStats{}, errortypes.NewDatabaseError(
			"Formula template stats loader is not configured",
		)
	}

	return loaders.FormulaTemplateStatsByID(ctx, obj.ID.String())
}
