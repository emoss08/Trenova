package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
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
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), one)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationDelete,
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
		func() (*agent.ToolPreview, error) {
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

type fakeInsightRestorer struct {
	guard    *agenttoolstest.WriteGuard
	found    *insight.Insight
	hidden   bool
	restored bool
}

func (f *fakeInsightRestorer) GetDetail(
	context.Context,
	insightports.GetInsightDetailRequest,
) (*insightports.InsightDetail, error) {
	if f.hidden {
		return nil, errortypes.NewAuthorizationError("You cannot see this insight")
	}

	return &insightports.InsightDetail{Insight: f.found}, nil
}

func (f *fakeInsightRestorer) PlanRestore(
	context.Context,
	insightports.RestoreInsightRequest,
) (*kernelports.RecordChange[insight.Insight], error) {
	restored := *f.found
	if err := restored.Restore(); err != nil {
		return nil, err
	}

	return &kernelports.RecordChange[insight.Insight]{Before: f.found, After: &restored}, nil
}

func (f *fakeInsightRestorer) Restore(
	context.Context,
	insightports.RestoreInsightRequest,
) (*insight.Insight, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.restored = true

	return f.found, nil
}

func TestRestoreInsight_OnlyADismissedFindingTheReaderMaySee(t *testing.T) {
	t.Parallel()

	dismissedAt := int64(1_789_000_000)
	insights := &fakeInsightRestorer{guard: &agenttoolstest.WriteGuard{}, found: &insight.Insight{
		ID:            pulid.MustNew("inst_"),
		Headline:      "Late deliveries for Acme",
		Status:        insight.StatusDismissed,
		DismissedAt:   &dismissedAt,
		DismissReason: "Seasonal",
		Version:       2,
	}}
	tool := insighttools.NewRestoreInsightTool(insights)
	params := agenttoolstest.ExecuteParams(
		map[string]any{insighttools.ParamInsightID: insights.found.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		insights.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(insight.StatusActive),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, insights.restored)

	insights.found.Status = insight.StatusActive
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	insights.hidden = true
	insights.restored = false
	require.Error(t, tool.Execute(t.Context(), params))
	assert.False(t, insights.restored)
}

type fakeAlerts struct {
	guard      *agenttoolstest.WriteGuard
	stored     *tablechangealert.TCASubscription
	getRequest tablechangealertports.GetTCASubscriptionByIDRequest
	updated    *tablechangealert.TCASubscription
	paused     bool
	resumed    bool
	deleted    bool
}

func (f *fakeAlerts) GetSubscriptionByID(
	_ context.Context,
	req tablechangealertports.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	f.getRequest = req
	copied := *f.stored

	return &copied, nil
}

func (f *fakeAlerts) PlanUpdateSubscription(
	_ context.Context,
	entity *tablechangealert.TCASubscription,
) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
	if entity.TableName == "secrets" {
		return nil, errortypes.NewValidationError("tableName", errortypes.ErrInvalid,
			"Table is not eligible for change alerts")
	}

	return &kernelports.RecordChange[tablechangealert.TCASubscription]{
		Before: f.stored,
		After:  entity,
	}, nil
}

func (f *fakeAlerts) UpdateSubscription(
	_ context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeAlerts) PlanSetSubscriptionStatus(
	_ context.Context,
	_ pulid.ID,
	_ pagination.TenantInfo,
	status tablechangealert.SubscriptionStatus,
) (*kernelports.RecordChange[tablechangealert.TCASubscription], error) {
	changed := *f.stored
	changed.Status = status

	return &kernelports.RecordChange[tablechangealert.TCASubscription]{
		Before: f.stored,
		After:  &changed,
	}, nil
}

func (f *fakeAlerts) PauseSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	f.paused = true

	return f.stored, f.guard.Write()
}

func (f *fakeAlerts) ResumeSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	f.resumed = true

	return f.stored, f.guard.Write()
}

func (f *fakeAlerts) PlanDeleteSubscription(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return f.stored, nil
}

func (f *fakeAlerts) DeleteSubscription(context.Context, pulid.ID, pagination.TenantInfo) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.deleted = true

	return nil
}

func TestTableChangeAlertTools_ActOnlyOnTheCallersOwnAlert(t *testing.T) {
	t.Parallel()

	alerts := &fakeAlerts{
		guard: &agenttoolstest.WriteGuard{},
		stored: &tablechangealert.TCASubscription{
			ID:             pulid.MustNew("tcas_"),
			Name:           "Delayed loads",
			TableName:      "shipments",
			EventTypes:     []string{"UPDATE"},
			ConditionMatch: "all",
			Status:         tablechangealert.SubscriptionStatusActive,
			Version:        3,
		},
	}
	update := alerttools.NewUpdateTableChangeAlertTool(alerts)
	params := agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID: alerts.stored.ID.String(),
		"customMessage":         "A load slipped",
		"eventTypes":            []any{"update", "insert"},
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		alerts.guard,
		func() (*agent.ToolPreview, error) {
			return update.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Would change the alert")
	require.NoError(t, update.Execute(t.Context(), params))
	assert.Equal(t, params.Actor.UserID, alerts.getRequest.TenantInfo.UserID)
	assert.Equal(t, "A load slipped", alerts.updated.CustomMessage)
	assert.Equal(t, []string{"UPDATE", "INSERT"}, alerts.updated.EventTypes)
	assert.Equal(t, "Delayed loads", alerts.updated.Name)

	require.Error(
		t,
		update.(agenttoolports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				alerttools.ParamAlertID: alerts.stored.ID.String(),
				"tableName":             "secrets",
			},
		)),
	)

	status := alerttools.NewSetTableChangeAlertStatusTool(alerts)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID:   alerts.stored.ID.String(),
		changepreview.FieldStatus: "Paused",
	})))
	assert.True(t, alerts.paused)
	require.NoError(t, status.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		alerttools.ParamAlertID:   alerts.stored.ID.String(),
		changepreview.FieldStatus: "Active",
	})))
	assert.True(t, alerts.resumed)

	remove := alerttools.NewDeleteTableChangeAlertTool(alerts)
	deletePreview := agenttoolstest.PreviewWithoutWrites(
		t,
		alerts.guard,
		func() (*agent.ToolPreview, error) {
			return remove.(agenttoolports.ToolPreviewer).Preview(
				t.Context(),
				agenttoolstest.ExecuteParams(
					map[string]any{alerttools.ParamAlertID: alerts.stored.ID.String()}),
			)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationDelete,
		agenttoolstest.PreviewChange(t, deletePreview, 0).Operation,
	)
	require.NoError(t, remove.Execute(t.Context(), agenttoolstest.ExecuteParams(
		map[string]any{alerttools.ParamAlertID: alerts.stored.ID.String()})))
	assert.True(t, alerts.deleted)
}

type fakeWatchtower struct {
	guard     *agenttoolstest.WriteGuard
	item      *watchtower.Item
	dismissed bool
}

func (f *fakeWatchtower) PlanDismiss(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	*permissionports.RequestActor,
) (*kernelports.RecordChange[watchtower.Item], error) {
	after := *f.item
	now := int64(1_790_000_000)
	after.ResolvedAt = &now

	return &kernelports.RecordChange[watchtower.Item]{Before: f.item, After: &after}, nil
}

func (f *fakeWatchtower) Dismiss(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	*permissionports.RequestActor,
) (*watchtower.Item, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.dismissed = true

	return f.item, nil
}

func TestDismissWatchtowerItem_LeavesAgentOversightForAPerson(t *testing.T) {
	t.Parallel()

	items := &fakeWatchtower{guard: &agenttoolstest.WriteGuard{}, item: &watchtower.Item{
		ID:         pulid.MustNew("wti_"),
		SourceKind: watchtower.SourceWeatherAlert,
		Severity:   watchtower.Severity("Warning"),
		Title:      "Ice storm in Amarillo",
		Version:    1,
	}}
	tool := NewDismissWatchtowerItemTool(items)
	params := agenttoolstest.ExecuteParams(
		map[string]any{paramWatchtowerItemID: items.item.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		items.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationArchive,
		agenttoolstest.PreviewChange(t, preview, 0).Operation,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, items.dismissed)

	items.dismissed = false
	items.item.SourceKind = watchtower.SourceAgentRunFailed
	require.Error(t, tool.Execute(t.Context(), params))
	assert.False(t, items.dismissed)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

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
		func() (*agent.ToolPreview, error) {
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
		func() (*agent.ToolPreview, error) {
			return item.(agenttoolports.ToolPreviewer).Preview(t.Context(), itemParams)
		},
	)
	assert.Equal(
		t,
		agent.PreviewOperationArchive,
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
		func() (*agent.ToolPreview, error) {
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
