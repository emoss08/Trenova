package workerchecklistresolver

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func checklistTemplateFromInput(
	input *gqlmodel.WorkerChecklistTemplateInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	entity := &workerchecklist.WorkerChecklistTemplate{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           strings.TrimSpace(input.Code),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(gqlscalar.StringValue(input.Description)),
		Kind:           input.Kind,
		Trigger:        input.Trigger,
		Status:         input.Status,
		IsDefault:      input.IsDefault,
		Version:        int64(gqlscalar.IntValue(input.Version)),
		Items:          make([]*workerchecklist.WorkerChecklistTemplateItem, 0, len(input.Items)),
	}
	for i, item := range input.Items {
		if item == nil {
			continue
		}
		credentialTypeID, err := gqlid.OptionalID(item.CredentialTypeID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"items["+strconv.Itoa(i)+"].credentialTypeId",
				errortypes.ErrInvalid,
				"Credential type is invalid",
			)
		}
		documentTypeID, err := gqlid.OptionalID(item.DocumentTypeID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"items["+strconv.Itoa(i)+"].documentTypeId",
				errortypes.ErrInvalid,
				"Document type is invalid",
			)
		}
		entity.Items = append(entity.Items, &workerchecklist.WorkerChecklistTemplateItem{
			OrganizationID:   tenantInfo.OrgID,
			BusinessUnitID:   tenantInfo.BuID,
			Label:            strings.TrimSpace(item.Label),
			Description:      strings.TrimSpace(gqlscalar.StringValue(item.Description)),
			Kind:             item.Kind,
			Required:         item.Required,
			DueOffsetDays:    int32(item.DueOffsetDays), //nolint:gosec // bounded by validation
			Owner:            item.Owner,
			CredentialTypeID: credentialTypeID,
			DocumentTypeID:   documentTypeID,
			SortOrder:        int32(i), //nolint:gosec // item counts are tiny
		})
	}
	return entity, nil
}

func checklistTemplateCursorConnectionToModel(
	result *pagination.CursorListResult[*workerchecklist.WorkerChecklistTemplate],
) (*gqlmodel.WorkerChecklistTemplateConnection, error) {
	edges, err := gqlconnection.EntityCursorEdges(
		result.Items,
		result,
		func(node *workerchecklist.WorkerChecklistTemplate, cursor string) *gqlmodel.WorkerChecklistTemplateEdge {
			return &gqlmodel.WorkerChecklistTemplateEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerChecklistTemplateConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			result.HasNextPage,
			gqlconnection.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerChecklistTemplateEdge) string { return edge.Cursor },
			),
		),
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func checklistKindString(kind *workerchecklist.ChecklistKind) string {
	if kind == nil {
		return ""
	}
	return string(*kind)
}

func checklistTriggerString(trigger *workerchecklist.ChecklistTrigger) string {
	if trigger == nil {
		return ""
	}
	return string(*trigger)
}

func (r *MutationResolver) checklistItemRequest(
	ctx context.Context,
	input gqlmodel.WorkerChecklistItemActionInput,
) (*workerchecklistservice.ItemRequest, error) {
	authCtx, err := r.RequirePermission(
		ctx,
		permission.ResourceWorkerChecklist,
		permission.OpUpdate,
	)
	if err != nil {
		return nil, err
	}

	itemID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	evidenceID, err := gqlid.OptionalID(input.EvidenceDocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"evidenceDocumentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workerchecklistservice.ItemRequest{
		ID:                 itemID,
		TenantInfo:         resolvercore.TenantInfo(authCtx),
		Note:               gqlscalar.StringValue(input.Note),
		EvidenceDocumentID: evidenceID,
		Version:            int64(gqlscalar.IntValue(input.Version)),
		UserID:             authCtx.UserID,
	}, nil
}
