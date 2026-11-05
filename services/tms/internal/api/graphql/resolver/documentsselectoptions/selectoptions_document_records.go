package documentsselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *Sources) Document(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceDocument, permission.OpRead)
	if err != nil {
		return nil, err
	}

	query := req.SelectQuery
	if len(req.IDs) > 0 {
		query.Query = ""
		query.Pagination.Offset = 0
	}

	result, err := r.DocumentService.SelectOptions(ctx, &documentports.DocumentSelectOptionsRequest{
		SelectQueryRequest: query,
		ResourceID:         selectoption.StringFilter(req.Filters, "resourceId"),
		ResourceType:       selectoption.StringFilter(req.Filters, "resourceType"),
		IDs:                req.IDs,
		Viewer:             actorutil.FromAuthContext(authCtx).PersonUserID(),
	})
	if err != nil {
		return nil, err
	}

	if len(req.IDs) > 0 {
		items := selectoption.OrderedItems(
			req.IDs,
			result.Items,
			(*document.Document).GetID,
			documentSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	return selectoption.ListConnection(result, query.Pagination, documentSelectOptionItem)
}

func documentSelectOptionItem(entity *document.Document) selectoption.Item {
	return selectoption.ItemFor(documentSelectOption(entity), entity.CreatedAt, entity.ID)
}

func documentSelectOption(entity *document.Document) *gqlmodel.SelectOption {
	option := &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: documentLabel(entity),
		Meta: map[string]any{
			"fileType": entity.FileType,
			"fileSize": entity.FileSize,
		},
	}
	if entity.DocumentType != nil && entity.DocumentType.Name != "" {
		name := entity.DocumentType.Name
		option.Description = &name
	}
	return option
}

func documentLabel(entity *document.Document) string {
	if entity.OriginalName != "" {
		return entity.OriginalName
	}
	return entity.ID.String()
}
