package workercredentialresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func workerCredentialTypeFromInput(
	input *gqlmodel.WorkerCredentialTypeInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *workercredential.WorkerCredentialType {
	entity := &workercredential.WorkerCredentialType{
		ID:                     id,
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Code:                   strings.TrimSpace(input.Code),
		Name:                   strings.TrimSpace(input.Name),
		Description:            strings.TrimSpace(gqlscalar.StringValue(input.Description)),
		Category:               input.Category,
		Status:                 input.Status,
		IsRequired:             input.IsRequired,
		RequiredForDriverTypes: input.RequiredForDriverTypes,
		RenewalWindowDays: int32(
			input.RenewalWindowDays,
		), //nolint:gosec // bounded by validation
		RequiresNumber:   input.RequiresNumber,
		RequiresDocument: input.RequiresDocument,
		SortOrder: int32(
			gqlscalar.IntValue(input.SortOrder),
		), //nolint:gosec // small ordinal
		Version: int64(gqlscalar.IntValue(input.Version)),
	}
	if entity.RequiredForDriverTypes == nil {
		entity.RequiredForDriverTypes = []worker.DriverType{}
	}
	if input.ValidityMonths != nil {
		months := int32(*input.ValidityMonths) //nolint:gosec // bounded by validation
		entity.ValidityMonths = &months
	}
	return entity
}

func workerCredentialFromInput(
	input *gqlmodel.WorkerCredentialInput,
	tenantInfo pagination.TenantInfo,
) (*workercredential.WorkerCredential, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	typeID, err := pulid.MustParse(input.CredentialTypeID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"credentialTypeId",
			errortypes.ErrInvalid,
			"Credential type is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workercredential.WorkerCredential{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		WorkerID:         workerID,
		CredentialTypeID: typeID,
		Status:           workercredential.CredentialStatusActive,
		Number:           gqlscalar.StringValue(input.Number),
		IssuingAuthority: gqlscalar.StringValue(input.IssuingAuthority),
		IssuedAt:         gqlscalar.Int64Ptr(input.IssuedAt),
		ExpiresAt:        gqlscalar.Int64Ptr(input.ExpiresAt),
		DocumentID:       documentID,
		Notes:            gqlscalar.StringValue(input.Notes),
	}, nil
}

func workerCredentialFromUpdateInput(
	input *gqlmodel.UpdateWorkerCredentialInput,
	tenantInfo pagination.TenantInfo,
) (*workercredential.WorkerCredential, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workercredential.WorkerCredential{
		ID:               id,
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		Number:           gqlscalar.StringValue(input.Number),
		IssuingAuthority: gqlscalar.StringValue(input.IssuingAuthority),
		IssuedAt:         gqlscalar.Int64Ptr(input.IssuedAt),
		ExpiresAt:        gqlscalar.Int64Ptr(input.ExpiresAt),
		DocumentID:       documentID,
		Notes:            gqlscalar.StringValue(input.Notes),
		Version:          int64(input.Version),
	}, nil
}

func workerCredentialTypeCursorConnectionToModel(
	result *pagination.CursorListResult[*workercredential.WorkerCredentialType],
) (*gqlmodel.WorkerCredentialTypeConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *workercredential.WorkerCredentialType, cursor string) *gqlmodel.WorkerCredentialTypeEdge {
			return &gqlmodel.WorkerCredentialTypeEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerCredentialTypeConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerCredentialTypeEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func credentialCategoryString(category *workercredential.CredentialCategory) string {
	if category == nil {
		return ""
	}
	return string(*category)
}
