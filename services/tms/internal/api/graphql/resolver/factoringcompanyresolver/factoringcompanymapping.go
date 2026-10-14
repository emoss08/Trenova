package factoringcompanyresolver

import (
	"context"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/pagination"
)

func factoringCompanyColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FactoringCompanySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func factoringCompanyConnectionToModel(
	result *pagination.CursorListResult[*factoringcompany.FactoringCompany],
) (*gqlmodel.FactoringCompanyConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoringcompany.FactoringCompany, cursor string) *gqlmodel.FactoringCompanyEdge {
			return &gqlmodel.FactoringCompanyEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FactoringCompanyEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FactoringCompanyConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func applyFactoringCompanyInput(
	entity *factoringcompany.FactoringCompany,
	in *gqlmodel.FactoringCompanyInput,
) error {
	stateID, err := pulid.MustParse(in.StateID)
	if err != nil {
		return errortypes.NewValidationError("stateId", errortypes.ErrInvalid, "Choose a state")
	}
	entity.Status = in.Status
	entity.Code = in.Code
	entity.Name = in.Name
	entity.AddressLine1 = in.AddressLine1
	entity.AddressLine2 = stringutils.FromPtr(in.AddressLine2)
	entity.City = in.City
	entity.StateID = stateID
	entity.PostalCode = in.PostalCode
	entity.Email = stringutils.FromPtr(in.Email)
	entity.PhoneNumber = stringutils.FromPtr(in.PhoneNumber)
	entity.RemittanceInstructions = stringutils.FromPtr(in.RemittanceInstructions)
	entity.NoticeOfAssignment = stringutils.FromPtr(in.NoticeOfAssignment)
	if in.NonRecourse != nil {
		entity.NonRecourse = *in.NonRecourse
	}
	entity.Notes = stringutils.FromPtr(in.Notes)
	return nil
}
