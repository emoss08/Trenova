package documenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
)

var documentHomeLabels = map[string]string{
	"resourceType":                "Filed under",
	changepreview.FieldShipmentID: changepreview.LabelShipment,
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
			changepreview.FieldShipmentID: permission.ResourceShipment,
		}),
		toolpreview.Labels(documentHomeLabels),
	}
}

func (t *AttachDocumentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.documents.PreviewAttachLineage(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
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
		Version:  changepreview.PreviewVersion(plan.Current.Version),
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
