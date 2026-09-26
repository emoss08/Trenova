package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/stringutils"
)

var _ serviceports.ToolPreviewer = (*attachDocumentTool)(nil)

var documentHomeLabels = map[string]string{
	"resourceType":  "Filed under",
	fieldShipmentID: labelShipment,
}

type documentHome struct {
	ResourceType string `json:"resourceType"`
	ShipmentID   string `json:"shipmentId,omitempty"`
}

func documentHomeOf(entity *document.Document) *documentHome {
	home := &documentHome{ResourceType: entity.ResourceType}
	if entity.ResourceType == documentResourceTypeShipment {
		home.ShipmentID = entity.ResourceID
	}

	return home
}

func documentHomeOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldShipmentID: permission.ResourceShipment,
		}),
		toolpreview.Labels(documentHomeLabels),
	}
}

func (t *attachDocumentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.documents.PreviewAttachLineage(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
				toolpreview.Build("Would attach a document to a shipment."),
				err,
			), nil
		}

		return nil, err
	}

	name := stringutils.FirstNonEmpty(plan.Current.OriginalName, plan.Current.FileName)
	if plan.Unchanged {
		return toolpreview.Build(fmt.Sprintf(
			"%s is already attached to this shipment, so nothing would change.",
			name,
		)), nil
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDocument,
		ID:       plan.Current.ID,
		Label:    name,
		Version:  previewVersion(plan.Current.Version),
	}, documentHomeOf(plan.Current), documentHomeOf(plan.Attached), documentHomeOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would attach %s, with every version of it, to the shipment so it rides with the load "+
			"and its billing.",
		name,
	), change), nil
}
