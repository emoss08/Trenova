package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func failedSyncRecord(
	status accountingsync.SyncStatus,
	category accountingsync.SyncErrorCategory,
) *accountingsync.AccountingSyncRecord {
	record := syncRecord(status)
	record.ErrorCategory = category
	return record
}

func retryPinner(t *testing.T, tool serviceports.AgentTool) serviceports.ToolProposalSelectionResolver {
	t.Helper()

	pinner, pins := tool.(serviceports.ToolProposalSelectionResolver)
	require.True(t, pins, "a proposed retry pins the records its categories name")

	return pinner
}

func pinnedIDs(t *testing.T, params map[string]any) []string {
	t.Helper()

	raw, ok := params[paramSyncRecordIDs].([]any)
	require.True(t, ok, "the pinned call names its records")
	ids := make([]string, 0, len(raw))
	for _, value := range raw {
		id, isText := value.(string)
		require.True(t, isText)
		ids = append(ids, id)
	}

	return ids
}

func approvedFrom(params serviceports.ToolExecuteParams) serviceports.ToolExecuteParams {
	params.ProposalID = pulid.MustNew("ap_")
	return params
}

func TestRetryAccountingSync_PinsTheRecordsItsCategoriesNameWhenProposed(t *testing.T) {
	t.Parallel()

	dead := failedSyncRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient)
	limited := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorRateLimited)
	retrying := failedSyncRecord(accountingsync.SyncStatusRetrying, accountingsync.SyncErrorTransient)
	mapping := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	synced := failedSyncRecord(accountingsync.SyncStatusSynced, accountingsync.SyncErrorTransient)
	operator := newSyncOperator(dead, limited, retrying, mapping, synced)
	tool := newRetryAccountingSyncTool(operator)
	params := syncParams(map[string]any{
		"system":          "QuickBooksOnline",
		"errorCategories": []any{"Transient", "RateLimited"},
	})

	pinned, err := retryPinner(t, tool).ResolveProposalSelection(t.Context(), params)
	require.NoError(t, err)

	want := []string{dead.ID.String(), limited.ID.String(), retrying.ID.String()}
	assert.ElementsMatch(t, want, pinnedIDs(t, pinned))
	assert.Equal(t, "QuickBooksOnline", pinned["system"])
	assert.Equal(t, []any{"Transient", "RateLimited"}, pinned[paramSyncErrorCategories],
		"the categories stay beside the records to say why they were chosen")
	assert.Equal(t,
		map[string]any{
			"system":          "QuickBooksOnline",
			"errorCategories": []any{"Transient", "RateLimited"},
		},
		params.Params,
		"the model's own call is left as it sent it",
	)
	require.Len(t, operator.listed, 1)
	assert.ElementsMatch(t,
		[]accountingsync.SyncStatus{
			accountingsync.SyncStatusBlocked,
			accountingsync.SyncStatusDeadLettered,
			accountingsync.SyncStatusRetrying,
		},
		operator.listed[0].Statuses,
	)
	assert.Equal(t, maxSyncRetryIDs, operator.listed[0].Cursor.Limit)
	assert.Nil(t, operator.retried, "pinning changes nothing")
}

func TestRetryAccountingSync_PreviewShowsThePinnedRecordsAndRunsOnlyThose(t *testing.T) {
	t.Parallel()

	dead := failedSyncRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient)
	blocked := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorTransient)
	operator := newSyncOperator(dead, blocked)
	tool := newRetryAccountingSyncTool(operator)

	pinned, err := retryPinner(t, tool).ResolveProposalSelection(
		t.Context(),
		syncParams(map[string]any{"system": "QuickBooksOnline", "errorCategories": []any{"Transient"}}),
	)
	require.NoError(t, err)
	approved := approvedFrom(syncParams(pinned))

	preview := previewWithoutWrites(t, &operator.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), approved)
	})
	assert.Empty(t, preview.Warnings)
	assert.False(t, preview.Partial, "every record that failed that way is pinned")
	assert.Contains(t, preview.Summary, "Would send 2 documents to QuickBooks Online again")
	require.Len(t, preview.Changes, 2)
	previewed := []pulid.ID{preview.Changes[0].EntityID, preview.Changes[1].EntityID}
	assert.ElementsMatch(t, []pulid.ID{dead.ID, blocked.ID}, previewed)

	late := failedSyncRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient)
	operator.records[late.ID] = late

	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, operator.retried)
	assert.ElementsMatch(t, []pulid.ID{dead.ID, blocked.ID}, operator.retried.IDs,
		"a record that failed after the proposal is not retried by its approval")
	assert.NotContains(t, operator.retried.IDs, late.ID)
	require.Len(t, operator.savedRecords, 2)
}

func TestRetryAccountingSync_CapsAPinnedSelectionAndSaysWhatIsLeft(t *testing.T) {
	t.Parallel()

	records := make([]*accountingsync.AccountingSyncRecord, 0, maxSyncRetryIDs+2)
	for range maxSyncRetryIDs + 2 {
		records = append(records,
			failedSyncRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient))
	}
	operator := newSyncOperator(records...)
	tool := newRetryAccountingSyncTool(operator)

	pinned, err := retryPinner(t, tool).ResolveProposalSelection(
		t.Context(),
		syncParams(map[string]any{"system": "QuickBooksOnline", "errorCategories": []any{"Transient"}}),
	)
	require.NoError(t, err)
	require.Len(t, pinnedIDs(t, pinned), maxSyncRetryIDs)
	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), syncParams(pinned)),
		"a pinned selection is a call the tool takes")

	preview := previewWithoutWrites(t, &operator.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), syncParams(pinned))
	})
	assert.True(t, preview.Partial, "the selection was cut to what one retry takes")
	assert.Contains(t, preview.Summary, "Would send 50 documents to QuickBooks Online again")
	assert.Contains(t, preview.Summary, "2 other records that last failed as Transient")
	assert.Empty(t, preview.Warnings)
}

func TestRetryAccountingSync_RefusesToPinASelectionOfNothing(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator(
		failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping),
	)
	tool := newRetryAccountingSyncTool(operator)

	_, err := retryPinner(t, tool).ResolveProposalSelection(
		t.Context(),
		syncParams(map[string]any{"system": "QuickBooksOnline", "errorCategories": []any{"Transient"}}),
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "Transient")
}

func TestRetryAccountingSync_LeavesNamedRecordsAndUnpinnableCallsAsTheyAre(t *testing.T) {
	t.Parallel()

	blocked := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorTransient)
	operator := newSyncOperator(blocked)
	tool := newRetryAccountingSyncTool(operator)
	pinner := retryPinner(t, tool)
	validator := tool.(serviceports.ToolValidator)

	for _, params := range []map[string]any{
		{"system": "QuickBooksOnline", "syncRecordIds": []any{blocked.ID.String()}},
		{
			"system":          "QuickBooksOnline",
			"syncRecordIds":   []any{blocked.ID.String()},
			"errorCategories": []any{"Transient"},
		},
		{"system": "QuickBooksOnline"},
	} {
		pinned, err := pinner.ResolveProposalSelection(t.Context(), syncParams(params))
		require.NoError(t, err, params)
		assert.Equal(t, params, pinned, "only criteria alone are resolved")
	}
	assert.Empty(t, operator.listed)

	require.Error(t,
		validator.Validate(t.Context(), syncParams(map[string]any{"system": "QuickBooksOnline"})),
		"naming neither records nor categories is refused, as before",
	)
	require.NoError(t,
		validator.Validate(t.Context(), syncParams(map[string]any{
			"system":          "QuickBooksOnline",
			"syncRecordIds":   []any{blocked.ID.String()},
			"errorCategories": []any{"Transient"},
		})),
		"naming both is taken, as before",
	)
}

func TestRetryAccountingSync_AnApprovalRunsOnlyOnRecordsItNames(t *testing.T) {
	t.Parallel()

	dead := failedSyncRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient)
	operator := newSyncOperator(dead)
	tool := newRetryAccountingSyncTool(operator)
	unpinned := approvedFrom(syncParams(map[string]any{
		"system":          "QuickBooksOnline",
		"errorCategories": []any{"Transient"},
	}))

	err := tool.Execute(t.Context(), unpinned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "propose it again")
	assert.Nil(t, operator.retried)

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), unpinned)
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
}

func TestRetryAccountingSync_ANamedRecordThatNoLongerFailedThatWayIsNotShown(t *testing.T) {
	t.Parallel()

	transient := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorTransient)
	moved := failedSyncRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	operator := newSyncOperator(transient, moved)
	tool := newRetryAccountingSyncTool(operator)
	params := syncParams(map[string]any{
		"system":          "QuickBooksOnline",
		"syncRecordIds":   []any{transient.ID.String(), moved.ID.String()},
		"errorCategories": []any{"Transient"},
	})

	preview := previewWithoutWrites(t, &operator.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, transient.ID, preview.Changes[0].EntityID)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, operator.savedRecords, 1)
	assert.Equal(t, transient.ID, operator.savedRecords[0].ID)
}

func TestRetryAccountingSync_DeclaresItsRecordsASubsetAPersonMayNarrow(t *testing.T) {
	t.Parallel()

	tool := newRetryAccountingSyncTool(newSyncOperator())
	proposed := []any{pulid.MustNew("acctsr_").String(), pulid.MustNew("acctsr_").String()}

	fields := serviceports.ProposalFields(tool, map[string]any{
		"system":           "QuickBooksOnline",
		paramSyncRecordIDs: proposed,
	})
	var subset *toolschema.Field
	for idx := range fields {
		if fields[idx].Name == paramSyncRecordIDs {
			subset = &fields[idx]
		}
	}
	require.NotNil(t, subset)
	assert.Equal(t, toolschema.KindRecordSubset, subset.Kind)
	assert.Equal(t, permission.ResourceAccountingSync.String(), subset.Resource)
	assert.Len(t, subset.Choices, 2, "every record proposed is listed for the approver to untick")
}
