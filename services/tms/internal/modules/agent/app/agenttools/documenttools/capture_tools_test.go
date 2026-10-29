package documenttools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCaptures struct {
	guard      *agenttoolstest.WriteGuard
	item       *capture.CaptureItem
	batch      *capture.CaptureBatch
	filed      []captureservice.FileItemInput
	discarded  *captureservice.DiscardItemInput
	discardAll *captureservice.DiscardBatchInput
}

func (f *fakeCaptures) GetItem(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*capture.CaptureItem, error) {
	copied := *f.item
	copied.ID = id

	return &copied, nil
}

func (f *fakeCaptures) GetBatch(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*capture.CaptureBatch, error) {
	return f.batch, nil
}

func (f *fakeCaptures) PlanFileItem(
	_ context.Context,
	in *captureservice.FileItemInput,
) (*captureservice.FilePlan, error) {
	if in.Version != f.item.Version {
		return nil, errortypes.NewConflictError("Somebody else changed this document")
	}
	item := *f.item
	item.ID = in.ItemID

	return &captureservice.FilePlan{Item: &item, Batch: f.batch, Target: capture.Target{
		ResourceType:   in.TargetType,
		ResourceID:     &in.TargetID,
		DocumentTypeID: in.DocumentTypeID,
	}}, nil
}

func (f *fakeCaptures) FileItem(
	_ context.Context,
	in *captureservice.FileItemInput,
) (*capture.CaptureItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.filed = append(f.filed, *in)

	return f.item, nil
}

func (f *fakeCaptures) FileItems(
	_ context.Context,
	in *captureservice.FileItemsInput,
) (*captureservice.FileItemsResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.filed = append(f.filed, in.Items...)

	return &captureservice.FileItemsResult{
		Filed: []*capture.CaptureItem{f.item},
		Failures: []captureservice.FileItemFailure{
			{ItemID: in.Items[1].ItemID, Message: "Record not found"},
		},
	}, nil
}

func (f *fakeCaptures) PlanDiscardItem(
	_ context.Context,
	in *captureservice.DiscardItemInput,
) (*captureservice.DiscardItemPlan, error) {
	return &captureservice.DiscardItemPlan{Item: f.item, Batch: f.batch}, nil
}

func (f *fakeCaptures) DiscardItem(
	_ context.Context,
	in *captureservice.DiscardItemInput,
) (*capture.CaptureBatch, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.discarded = in

	return f.batch, nil
}

func (f *fakeCaptures) PlanDiscardBatch(
	context.Context,
	*captureservice.DiscardBatchInput,
) (*captureservice.DiscardBatchPlan, error) {
	return &captureservice.DiscardBatchPlan{
		Batch:     f.batch,
		Open:      []*capture.CaptureItem{f.item},
		Unchanged: f.batch.Status.Terminal(),
	}, nil
}

func (f *fakeCaptures) DiscardBatch(
	_ context.Context,
	in *captureservice.DiscardBatchInput,
) (*capture.CaptureBatch, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.discardAll = in

	return f.batch, nil
}

func captureFixture() *fakeCaptures {
	return &fakeCaptures{
		guard: &agenttoolstest.WriteGuard{},
		item: &capture.CaptureItem{
			ID:       pulid.MustNew("citm_"),
			Position: 1,
			Status:   capture.ItemProposed,
			PageIDs:  []pulid.ID{pulid.MustNew("cpg_"), pulid.MustNew("cpg_")},
			Version:  6,
		},
		batch: &capture.CaptureBatch{
			ID:      pulid.MustNew("cbat_"),
			Status:  capture.BatchReady,
			JobName: "Morning PODs",
			Version: 9,
		},
	}
}

func TestFileCaptureItems_FilesWithTheVersionItReadAndReportsWhatDidNotFile(t *testing.T) {
	t.Parallel()

	captures := captureFixture()
	tool := NewFileCaptureItemsTool(captures)
	shipmentID := pulid.MustNew("shp_")
	entry := func(id pulid.ID) map[string]any {
		return map[string]any{
			paramCaptureItemID:   id.String(),
			paramCaptureTarget:   "shipment",
			paramCaptureTargetID: shipmentID.String(),
		}
	}
	one := agenttoolstest.ExecuteParams(
		map[string]any{paramCaptureItems: []any{entry(captures.item.ID)}},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		captures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), one)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceCaptureBatch, change.Resource)
	assert.Equal(
		t,
		string(capture.ItemFiled),
		agenttoolstest.FieldByPath(t, change, "status").After,
	)
	require.NoError(t, tool.Execute(t.Context(), one))
	require.Len(t, captures.filed, 1)
	assert.Equal(t, int64(6), captures.filed[0].Version)
	assert.Equal(t, shipmentID, captures.filed[0].TargetID)

	second := pulid.MustNew("citm_")
	two := agenttoolstest.ExecuteParams(map[string]any{
		paramCaptureItems: []any{entry(captures.item.ID), entry(second)},
	})
	err := tool.Execute(t.Context(), two)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not filed: "+second.String()+": Record not found")
	assert.Len(t, captures.filed, 3)

	for name, bad := range map[string][]any{
		"an unknown record kind": {map[string]any{
			paramCaptureItemID:   captures.item.ID.String(),
			paramCaptureTarget:   "invoice",
			paramCaptureTargetID: shipmentID.String(),
		}},
		"the same document twice": {entry(captures.item.ID), entry(captures.item.ID)},
	} {
		require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
			agenttoolstest.ExecuteParams(map[string]any{paramCaptureItems: bad})), name)
	}
}

func TestDiscardCaptureTools_SendTheVersionTheyRead(t *testing.T) {
	t.Parallel()

	captures := captureFixture()
	item := NewDiscardCaptureItemTool(captures)
	itemParams := agenttoolstest.ExecuteParams(
		map[string]any{paramCaptureItemIDOne: captures.item.ID.String()},
	)
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		captures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return item.(agenttoolports.ToolPreviewer).Preview(t.Context(), itemParams)
		},
	)
	assert.Equal(
		t,
		agentpreview.PreviewOperationArchive,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, item.Execute(t.Context(), itemParams))
	assert.Equal(t, int64(6), captures.discarded.Version)

	batch := NewDiscardCaptureBatchTool(captures)
	batchParams := agenttoolstest.ExecuteParams(
		map[string]any{paramCaptureBatchID: captures.batch.ID.String()},
	)
	batchPreview := agenttoolstest.PreviewWithoutWrites(
		t,
		captures.guard,
		func() (*agentpreview.ToolPreview, error) {
			return batch.(agenttoolports.ToolPreviewer).Preview(t.Context(), batchParams)
		},
	)
	assert.Contains(t, batchPreview.Summary, "Would discard 1 unfiled document of Morning PODs")
	require.NoError(t, batch.Execute(t.Context(), batchParams))
	assert.Equal(t, int64(9), captures.discardAll.Version)
	assert.Equal(t, agent.TierPropose, batch.Policy().MaxTier)

	captures.batch.Status = capture.BatchDiscarded
	require.Error(t, batch.(agenttoolports.ToolValidator).Validate(t.Context(), batchParams))
}
