package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingAttacher struct {
	doc   *document.Document
	guard writeGuard
}

func (f *savingAttacher) PreviewAttachLineage(
	_ context.Context,
	req *documentservice.AttachLineageRequest,
) (*documentservice.AttachLineagePlan, error) {
	current := *f.doc
	plan := &documentservice.AttachLineagePlan{Current: &current}
	if current.ResourceType == req.ResourceType && current.ResourceID == req.ResourceID {
		plan.Unchanged = true
		plan.Attached = &current

		return plan, nil
	}
	attached := current
	documentservice.ApplyLineageMove(&attached, req.ResourceType, req.ResourceID)
	plan.Attached = &attached

	return plan, nil
}

func (f *savingAttacher) AttachLineageToResource(
	_ context.Context,
	_ pulid.ID,
	resourceType string,
	resourceID string,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*document.Document, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	documentservice.ApplyLineageMove(f.doc, resourceType, resourceID)

	return f.doc, nil
}

func TestAttachDocument_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "pod-88213.pdf",
		ResourceType: "unassigned",
		Version:      2,
	}
	before := *doc
	attacher := &savingAttacher{doc: doc}
	tool := newAttachDocumentTool(attacher).(*attachDocumentTool)
	shipmentID := pulid.MustNew("shp_")
	params := executeParams(map[string]any{
		"documentId": doc.ID.String(),
		"shipmentId": shipmentID.String(),
	})

	preview := previewWithoutWrites(t, &attacher.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceDocument, change.Resource)
	assert.Equal(t, "pod-88213.pdf", change.Label)
	shipment := fieldByPath(t, change, "shipmentId")
	require.NotNil(t, shipment.AfterRef)
	assert.Equal(t, shipmentID, shipment.AfterRef.ID)
	assert.Equal(t, "unassigned", fieldByPath(t, change, "resourceType").Before)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, documentHomeOf(&before), documentHomeOf(doc),
		documentHomeOptions()...)
}

func TestAttachDocument_PreviewSaysWhenItIsAlreadyThere(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	doc := &document.Document{
		ID:           pulid.MustNew("doc_"),
		FileName:     "bol.pdf",
		ResourceType: documentResourceTypeShipment,
		ResourceID:   shipmentID.String(),
	}
	attacher := &savingAttacher{doc: doc}
	tool := newAttachDocumentTool(attacher).(*attachDocumentTool)

	preview := previewWithoutWrites(t, &attacher.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"documentId": doc.ID.String(),
			"shipmentId": shipmentID.String(),
		}))
	})

	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "already attached")
}
