package hazmatsegregationruleresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func hazmatSegregationRuleColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HazmatSegregationRuleSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func hazmatSegregationRuleConnectionToModel(
	result *pagination.CursorListResult[*hazmatsegregationrule.HazmatSegregationRule],
) (*gqlmodel.HazmatSegregationRuleConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *hazmatsegregationrule.HazmatSegregationRule, cursor string) *gqlmodel.HazmatSegregationRuleEdge {
			return &gqlmodel.HazmatSegregationRuleEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HazmatSegregationRuleEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HazmatSegregationRuleConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func hazmatSegregationRuleFromInput(
	input *gqlmodel.HazmatSegregationRuleInput,
	authCtx *authctx.AuthContext,
) (*hazmatsegregationrule.HazmatSegregationRule, error) {
	entity := &hazmatsegregationrule.HazmatSegregationRule{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyHazmatSegregationRuleInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyHazmatSegregationRuleInput(
	entity *hazmatsegregationrule.HazmatSegregationRule,
	input *gqlmodel.HazmatSegregationRuleInput,
) error {
	hazmatAID, err := gqlid.OptionalScopedID("hazmatAId", input.HazmatAId)
	if err != nil {
		return err
	}
	hazmatBID, err := gqlid.OptionalScopedID("hazmatBId", input.HazmatBId)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.ClassA = input.ClassA
	entity.ClassB = input.ClassB
	entity.HazmatAID = gqlid.IDValuePtr(hazmatAID)
	entity.HazmatBID = gqlid.IDValuePtr(hazmatBID)
	entity.SegregationType = input.SegregationType
	entity.MinimumDistance = input.MinimumDistance
	entity.DistanceUnit = gqlscalar.StringValue(input.DistanceUnit)
	entity.HasExceptions = input.HasExceptions
	entity.ExceptionNotes = gqlscalar.StringValue(input.ExceptionNotes)
	entity.ReferenceCode = gqlscalar.StringValue(input.ReferenceCode)
	entity.RegulationSource = gqlscalar.StringValue(input.RegulationSource)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}
