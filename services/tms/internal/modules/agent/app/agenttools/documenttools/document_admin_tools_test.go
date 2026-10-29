package documenttools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDocumentKeeper struct {
	guard     *agenttoolstest.WriteGuard
	docs      []*document.Document
	versions  []*document.Document
	deleted   pulid.ID
	bulk      *documentservice.BulkDeleteRequest
	restored  pulid.ID
	unchanged bool
}

func (f *fakeDocumentKeeper) PlanDelete(
	_ context.Context,
	req *documentservice.BulkDeleteRequest,
) (*documentservice.DeletePlan, error) {
	if len(req.IDs) > len(f.docs) {
		return nil, errortypes.NewValidationError("ids", errortypes.ErrInvalid,
			"These are not documents of this organization")
	}

	return &documentservice.DeletePlan{
		Documents: f.docs[:len(req.IDs)],
		Versions:  map[pulid.ID][]*document.Document{},
	}, nil
}

func (f *fakeDocumentKeeper) Delete(
	_ context.Context,
	req documentports.DeleteDocumentRequest,
	_ pulid.ID,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = req.ID

	return nil
}

func (f *fakeDocumentKeeper) BulkDelete(
	_ context.Context,
	req *documentservice.BulkDeleteRequest,
) (*documentservice.BulkDeleteResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.bulk = req

	return &documentservice.BulkDeleteResult{DeletedCount: len(req.IDs)}, nil
}

func (f *fakeDocumentKeeper) ListVersions(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*document.Document, error) {
	return f.versions, nil
}

func (f *fakeDocumentKeeper) PlanRestoreVersion(
	_ context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
) (*documentservice.RestorePlan, error) {
	var target, current *document.Document
	for _, version := range f.versions {
		if version.ID == id {
			target = version
		}
		if version.IsCurrentVersion {
			current = version
		}
	}

	return &documentservice.RestorePlan{
		Target:    target,
		Current:   current,
		Unchanged: f.unchanged,
	}, nil
}

func (f *fakeDocumentKeeper) RestoreVersion(
	_ context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
	_ pulid.ID,
) (*document.Document, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.restored = id

	return f.versions[0], nil
}

func TestDeleteDocuments_OneGoesThroughDeleteAndMoreThroughBulk(t *testing.T) {
	t.Parallel()

	docs := []*document.Document{
		{ID: pulid.MustNew("doc_"), OriginalName: "pod.pdf", Version: 1},
		{ID: pulid.MustNew("doc_"), OriginalName: "bol.pdf", Version: 1},
	}
	documents := &fakeDocumentKeeper{guard: &agenttoolstest.WriteGuard{}, docs: docs}
	tool := NewDeleteDocumentsTool(documents)

	one := agenttoolstest.ExecuteParams(
		map[string]any{paramDocumentIDs: []any{docs[0].ID.String()}},
	)
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		documents.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), one)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), one))
	assert.Equal(t, docs[0].ID, documents.deleted)
	assert.Nil(t, documents.bulk)

	two := agenttoolstest.ExecuteParams(map[string]any{
		paramDocumentIDs: []any{docs[0].ID.String(), docs[1].ID.String()},
	})
	require.NoError(t, tool.Execute(t.Context(), two))
	require.NotNil(t, documents.bulk)
	assert.Len(t, documents.bulk.IDs, 2)

	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.False(t, tool.Policy().Reversible)
}

func TestRestoreDocumentVersion_FindsTheVersionByItsNumber(t *testing.T) {
	t.Parallel()

	lineage := pulid.MustNew("doc_")
	versions := []*document.Document{
		{ID: pulid.MustNew("doc_"), LineageID: lineage, VersionNumber: 2, IsCurrentVersion: true,
			OriginalName: "pod-v2.pdf"},
		{ID: pulid.MustNew("doc_"), LineageID: lineage, VersionNumber: 1, OriginalName: "pod.pdf"},
	}
	documents := &fakeDocumentKeeper{guard: &agenttoolstest.WriteGuard{}, versions: versions}
	tool := NewRestoreDocumentVersionTool(documents)
	params := agenttoolstest.ExecuteParams(map[string]any{
		paramDocumentID:    versions[0].ID.String(),
		paramVersionNumber: 1,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		documents.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would make version 1 of pod.pdf the current one")
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, versions[1].ID, documents.restored)

	missing := agenttoolstest.ExecuteParams(map[string]any{
		paramDocumentID:    versions[0].ID.String(),
		paramVersionNumber: 5,
	})
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), missing))

	documents.unchanged = true
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}
