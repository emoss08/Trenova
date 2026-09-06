package resolver

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func checklistTemplateFromInput(
	input *gqlmodel.WorkerChecklistTemplateInput,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerChecklistTemplate, error) {
	entity := &worker.WorkerChecklistTemplate{
		ID:             id,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           strings.TrimSpace(input.Code),
		Name:           strings.TrimSpace(input.Name),
		Description:    strings.TrimSpace(stringValue(input.Description)),
		Kind:           input.Kind,
		Trigger:        input.Trigger,
		Status:         input.Status,
		IsDefault:      input.IsDefault,
		Version:        int64(intValue(input.Version)),
		Items:          make([]*worker.WorkerChecklistTemplateItem, 0, len(input.Items)),
	}
	for i, item := range input.Items {
		if item == nil {
			continue
		}
		credentialTypeID, err := optionalID(item.CredentialTypeID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"items["+strconv.Itoa(i)+"].credentialTypeId",
				errortypes.ErrInvalid,
				"Credential type is invalid",
			)
		}
		documentTypeID, err := optionalID(item.DocumentTypeID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"items["+strconv.Itoa(i)+"].documentTypeId",
				errortypes.ErrInvalid,
				"Document type is invalid",
			)
		}
		entity.Items = append(entity.Items, &worker.WorkerChecklistTemplateItem{
			OrganizationID:   tenantInfo.OrgID,
			BusinessUnitID:   tenantInfo.BuID,
			Label:            strings.TrimSpace(item.Label),
			Description:      strings.TrimSpace(stringValue(item.Description)),
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
	result *pagination.CursorListResult[*worker.WorkerChecklistTemplate],
) (*gqlmodel.WorkerChecklistTemplateConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.WorkerChecklistTemplate, cursor string) *gqlmodel.WorkerChecklistTemplateEdge {
			return &gqlmodel.WorkerChecklistTemplateEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerChecklistTemplateConnection{
		Edges: edges,
		PageInfo: pageInfo(
			result.HasNextPage,
			lastEdgeCursor(
				edges,
				func(edge *gqlmodel.WorkerChecklistTemplateEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func checklistKindString(kind *worker.ChecklistKind) string {
	if kind == nil {
		return ""
	}
	return string(*kind)
}

func checklistTriggerString(trigger *worker.ChecklistTrigger) string {
	if trigger == nil {
		return ""
	}
	return string(*trigger)
}

func (r *mutationResolver) checklistItemRequest(
	ctx context.Context,
	input gqlmodel.WorkerChecklistItemActionInput,
) (*workerchecklistservice.ItemRequest, error) {
	authCtx, err := r.requirePermission(ctx, permission.ResourceWorkerChecklist, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	itemID, err := pulid.MustParse(input.ID)
	if err != nil {
		return nil, err
	}
	evidenceID, err := optionalID(input.EvidenceDocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"evidenceDocumentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	return &workerchecklistservice.ItemRequest{
		ID:                 itemID,
		TenantInfo:         tenantInfo(authCtx),
		Note:               stringValue(input.Note),
		EvidenceDocumentID: evidenceID,
		Version:            int64(intValue(input.Version)),
		UserID:             authCtx.UserID,
	}, nil
}
