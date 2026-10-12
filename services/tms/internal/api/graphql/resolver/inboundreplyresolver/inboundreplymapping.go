package inboundreplyresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *TriageActionResolver) visibleShipment(
	ctx context.Context,
	shipmentID pulid.ID,
) (*shipmentports.ShipmentSummary, error) {
	if shipmentID.IsNil() {
		return nil, nil //nolint:nilnil // no shipment is linked
	}

	authCtx, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !r.HasPermission(ctx, authCtx, permission.ResourceShipment, permission.OpRead) {
		return nil, nil //nolint:nilnil // the viewer cannot read shipments, so none is shown
	}
	if err = base.RequireLoaders(ctx); err != nil {
		return nil, err
	}

	return base.OptionalMatch(loaders.ShipmentSummaryByID(ctx, shipmentID.String()))
}

func templateInput(input gqlmodel.TriageReplyTemplateInput) inboundreplyservice.TemplateInput {
	return inboundreplyservice.TemplateInput{Name: input.Name, Body: input.Body}
}

func loadFieldInputs(fields []*gqlmodel.TriageLoadFieldInput) []triageworkservice.LoadFieldInput {
	out := make([]triageworkservice.LoadFieldInput, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		out = append(out, triageworkservice.LoadFieldInput{Key: field.Key, Value: field.Value})
	}

	return out
}

func fileRequest(
	input gqlmodel.FileInboundAttachmentInput,
) (*triageworkservice.FileRequest, error) {
	attachmentID, err := pulid.MustParse(input.AttachmentID)
	if err != nil {
		return nil, err
	}
	shipmentID, err := pulid.MustParse(input.ShipmentID)
	if err != nil {
		return nil, err
	}
	documentTypeID, err := base.OptionalID(input.DocumentTypeID)
	if err != nil {
		return nil, err
	}

	req := &triageworkservice.FileRequest{
		AttachmentID:   attachmentID,
		ShipmentID:     shipmentID,
		DocumentTypeID: documentTypeID,
		Pages:          make([]triageworkservice.PageRange, 0, len(input.Pages)),
	}
	for _, page := range input.Pages {
		if page == nil {
			continue
		}
		typeID, parseErr := pulid.MustParse(page.DocumentTypeID)
		if parseErr != nil {
			return nil, parseErr
		}
		req.Pages = append(req.Pages, triageworkservice.PageRange{
			FromPage:       page.FromPage,
			ToPage:         page.ToPage,
			DocumentTypeID: typeID,
		})
	}

	return req, nil
}
