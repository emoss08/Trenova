package accessorialchargeresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func accessorialChargeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AccessorialChargeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func accessorialChargeConnectionToModel(
	result *pagination.CursorListResult[*accessorialcharge.AccessorialCharge],
) (*gqlmodel.AccessorialChargeConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *accessorialcharge.AccessorialCharge, cursor string) *gqlmodel.AccessorialChargeEdge {
			return &gqlmodel.AccessorialChargeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.AccessorialChargeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccessorialChargeConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func accessorialChargeFromInput(
	input *gqlmodel.AccessorialChargeInput,
	authCtx *authctx.AuthContext,
) (*accessorialcharge.AccessorialCharge, error) {
	entity := &accessorialcharge.AccessorialCharge{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyAccessorialChargeInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyAccessorialChargeInput(
	entity *accessorialcharge.AccessorialCharge,
	input *gqlmodel.AccessorialChargeInput,
) error {
	amount, err := gqldecimal.ParseDecimalField("amount", input.Amount, true)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	entity.Code = input.Code
	entity.Description = input.Description
	entity.Method = input.Method
	entity.RateUnit = ""
	if input.RateUnit != nil {
		entity.RateUnit = *input.RateUnit
	}
	entity.Amount = amount
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func applyAccessorialChargePatch(
	entity *accessorialcharge.AccessorialCharge,
	input *gqlmodel.AccessorialChargePatchInput,
) error {
	if err := gqlscalar.ApplyRequired(
		"status",
		"Status",
		input.Status,
		&entity.Status,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("code", "Code", input.Code, &entity.Code); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"description",
		"Description",
		input.Description,
		&entity.Description,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"method",
		"Method",
		input.Method,
		&entity.Method,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.RateUnit, &entity.RateUnit)

	var amount string
	if err := gqlscalar.ApplyRequired("amount", "Amount", input.Amount, &amount); err != nil {
		return err
	}
	if input.Amount.IsSet() {
		parsed, err := gqldecimal.ParseDecimalField("amount", amount, true)
		if err != nil {
			return err
		}
		entity.Amount = parsed
	}

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
