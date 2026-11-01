package fleetcoderesolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func fleetCodeColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FleetCodeSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func fleetCodeConnectionToModel(
	result *pagination.CursorListResult[*fleetcode.FleetCode],
) (*gqlmodel.FleetCodeConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *fleetcode.FleetCode, cursor string) *gqlmodel.FleetCodeEdge {
			return &gqlmodel.FleetCodeEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FleetCodeEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FleetCodeConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func fleetCodeFromInput(
	input *gqlmodel.FleetCodeInput,
	authCtx *authctx.AuthContext,
) (*fleetcode.FleetCode, error) {
	entity := &fleetcode.FleetCode{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyFleetCodeInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyFleetCodeInput(entity *fleetcode.FleetCode, input *gqlmodel.FleetCodeInput) error {
	managerID, err := gqlid.RequiredID("managerId", input.ManagerID)
	if err != nil {
		return err
	}

	entity.Status = input.Status
	entity.Code = input.Code
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.ManagerID = managerID
	entity.RevenueGoal = input.RevenueGoal
	entity.DeadheadGoal = input.DeadheadGoal
	entity.MileageGoal = input.MileageGoal
	entity.Color = gqlscalar.StringValue(input.Color)
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func applyFleetCodePatch(entity *fleetcode.FleetCode, input *gqlmodel.FleetCodePatchInput) error {
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
	manager, err := gqlscalar.RequiredOmittable("managerId", "Manager", input.ManagerID)
	if err != nil {
		return err
	}
	if manager.Set {
		managerID, idErr := gqlid.RequiredID("managerId", manager.Value)
		if idErr != nil {
			return idErr
		}
		entity.ManagerID = managerID
	}
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	gqlscalar.ApplyNullable(input.RevenueGoal, &entity.RevenueGoal)
	gqlscalar.ApplyNullable(input.DeadheadGoal, &entity.DeadheadGoal)
	gqlscalar.ApplyNullable(input.MileageGoal, &entity.MileageGoal)
	gqlscalar.ApplyClearable(input.Color, &entity.Color)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
