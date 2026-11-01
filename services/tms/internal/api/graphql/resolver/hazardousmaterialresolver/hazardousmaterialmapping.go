package hazardousmaterialresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func hazardousMaterialColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.HazardousMaterialSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func hazardousMaterialConnectionToModel(
	result *pagination.CursorListResult[*hazardousmaterial.HazardousMaterial],
) (*gqlmodel.HazardousMaterialConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *hazardousmaterial.HazardousMaterial, cursor string) *gqlmodel.HazardousMaterialEdge {
			return &gqlmodel.HazardousMaterialEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.HazardousMaterialEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.HazardousMaterialConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func hazardousMaterialFromInput(
	input *gqlmodel.HazardousMaterialInput,
	authCtx *authctx.AuthContext,
) *hazardousmaterial.HazardousMaterial {
	entity := &hazardousmaterial.HazardousMaterial{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	applyHazardousMaterialInput(entity, input)

	return entity
}

func applyHazardousMaterialInput(
	entity *hazardousmaterial.HazardousMaterial,
	input *gqlmodel.HazardousMaterialInput,
) {
	entity.Status = input.Status
	entity.Code = gqlscalar.StringValue(input.Code)
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.Class = input.Class
	entity.UNNumber = gqlscalar.StringValue(input.UnNumber)
	entity.PackingGroup = ""
	if input.PackingGroup != nil {
		entity.PackingGroup = *input.PackingGroup
	}
	entity.SubsidiaryHazardClass = gqlscalar.StringValue(input.SubsidiaryHazardClass)
	entity.ErgGuideNumber = gqlscalar.StringValue(input.ErgGuideNumber)
	entity.LabelCodes = gqlscalar.StringValue(input.LabelCodes)
	entity.SpecialProvisions = gqlscalar.StringValue(input.SpecialProvisions)
	entity.ProperShippingName = gqlscalar.StringValue(input.ProperShippingName)
	entity.HandlingInstructions = gqlscalar.StringValue(input.HandlingInstructions)
	entity.EmergencyContact = gqlscalar.StringValue(input.EmergencyContact)
	entity.EmergencyContactPhoneNumber = gqlscalar.StringValue(input.EmergencyContactPhoneNumber)
	entity.QuantityThreshold = gqlscalar.StringValue(input.QuantityThreshold)
	entity.PlacardRequired = input.PlacardRequired
	entity.IsReportableQuantity = input.IsReportableQuantity
	entity.MarinePollutant = input.MarinePollutant
	entity.InhalationHazard = input.InhalationHazard
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}
}

func applyHazardousMaterialPatch(
	entity *hazardousmaterial.HazardousMaterial,
	input *gqlmodel.HazardousMaterialPatchInput,
) error {
	if err := applyHazardousMaterialPatchRequired(entity, input); err != nil {
		return err
	}

	gqlscalar.ApplyClearable(input.Code, &entity.Code)
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	gqlscalar.ApplyClearable(input.UnNumber, &entity.UNNumber)
	gqlscalar.ApplyClearable(input.PackingGroup, &entity.PackingGroup)
	gqlscalar.ApplyClearable(input.SubsidiaryHazardClass, &entity.SubsidiaryHazardClass)
	gqlscalar.ApplyClearable(input.ErgGuideNumber, &entity.ErgGuideNumber)
	gqlscalar.ApplyClearable(input.LabelCodes, &entity.LabelCodes)
	gqlscalar.ApplyClearable(input.SpecialProvisions, &entity.SpecialProvisions)
	gqlscalar.ApplyClearable(input.ProperShippingName, &entity.ProperShippingName)
	gqlscalar.ApplyClearable(input.HandlingInstructions, &entity.HandlingInstructions)
	gqlscalar.ApplyClearable(input.EmergencyContact, &entity.EmergencyContact)
	gqlscalar.ApplyClearable(
		input.EmergencyContactPhoneNumber,
		&entity.EmergencyContactPhoneNumber,
	)
	gqlscalar.ApplyClearable(input.QuantityThreshold, &entity.QuantityThreshold)

	return nil
}

func applyHazardousMaterialPatchRequired(
	entity *hazardousmaterial.HazardousMaterial,
	input *gqlmodel.HazardousMaterialPatchInput,
) error {
	if err := gqlscalar.ApplyRequired(
		"status",
		"Status",
		input.Status,
		&entity.Status,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("name", "Name", input.Name, &entity.Name); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired("class", "Class", input.Class, &entity.Class); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"placardRequired",
		"Placard required",
		input.PlacardRequired,
		&entity.PlacardRequired,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"isReportableQuantity",
		"Reportable quantity",
		input.IsReportableQuantity,
		&entity.IsReportableQuantity,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"marinePollutant",
		"Marine pollutant",
		input.MarinePollutant,
		&entity.MarinePollutant,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"inhalationHazard",
		"Inhalation hazard",
		input.InhalationHazard,
		&entity.InhalationHazard,
	); err != nil {
		return err
	}

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
