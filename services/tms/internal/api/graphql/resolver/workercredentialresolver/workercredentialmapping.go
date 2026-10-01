package workercredentialresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func workerCredentialTypeFromInput(
	input *gqlmodel.WorkerCredentialTypeInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *worker.WorkerCredentialType {
	entity := &worker.WorkerCredentialType{
		ID:                     id,
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Code:                   strings.TrimSpace(input.Code),
		Name:                   strings.TrimSpace(input.Name),
		Description:            strings.TrimSpace(base.StringValue(input.Description)),
		Category:               input.Category,
		Status:                 input.Status,
		IsRequired:             input.IsRequired,
		RequiredForDriverTypes: input.RequiredForDriverTypes,
		RenewalWindowDays: int32(
			input.RenewalWindowDays,
		), //nolint:gosec // bounded by validation
		RequiresNumber:   input.RequiresNumber,
		RequiresDocument: input.RequiresDocument,
		SortOrder:        int32(base.IntValue(input.SortOrder)), //nolint:gosec // small ordinal
		Version:          int64(base.IntValue(input.Version)),
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
) (*worker.WorkerCredential, error) {
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
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &worker.WorkerCredential{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		WorkerID:         workerID,
		CredentialTypeID: typeID,
		Status:           worker.CredentialStatusActive,
		Number:           base.StringValue(input.Number),
		IssuingAuthority: base.StringValue(input.IssuingAuthority),
		IssuedAt:         base.Int64Ptr(input.IssuedAt),
		ExpiresAt:        base.Int64Ptr(input.ExpiresAt),
		DocumentID:       documentID,
		Notes:            base.StringValue(input.Notes),
	}, nil
}

func workerCredentialFromUpdateInput(
	input *gqlmodel.UpdateWorkerCredentialInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerCredential, error) {
	id, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &worker.WorkerCredential{
		ID:               id,
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		Number:           base.StringValue(input.Number),
		IssuingAuthority: base.StringValue(input.IssuingAuthority),
		IssuedAt:         base.Int64Ptr(input.IssuedAt),
		ExpiresAt:        base.Int64Ptr(input.ExpiresAt),
		DocumentID:       documentID,
		Notes:            base.StringValue(input.Notes),
		Version:          int64(input.Version),
	}, nil
}

func workerCredentialTypeCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerCredentialType],
) (*gqlmodel.WorkerCredentialTypeConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.WorkerCredentialType, cursor string) *gqlmodel.WorkerCredentialTypeEdge {
			return &gqlmodel.WorkerCredentialTypeEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerCredentialTypeConnection{
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerCredentialTypeEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func credentialCategoryString(category *worker.CredentialCategory) string {
	if category == nil {
		return ""
	}
	return string(*category)
}
