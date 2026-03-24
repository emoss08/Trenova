package auditjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessAuditBatchPayload(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	batchID := pulid.MustNew("aeb_")

	payload := &ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Timestamp:      12345,
		},
		Entries: []*audit.Entry{
			{ID: pulid.MustNew("ael_")},
		},
		BatchID: batchID,
	}

	assert.Equal(t, orgID, payload.OrganizationID)
	assert.Equal(t, buID, payload.BusinessUnitID)
	assert.Equal(t, int64(12345), payload.Timestamp)
	assert.Equal(t, batchID, payload.BatchID)
	assert.Len(t, payload.Entries, 1)
}

func TestProcessAuditBatchPayload_Empty(t *testing.T) {
	t.Parallel()

	payload := &ProcessAuditBatchPayload{}

	assert.Empty(t, payload.OrganizationID)
	assert.Empty(t, payload.BusinessUnitID)
	assert.Zero(t, payload.Timestamp)
	assert.Empty(t, payload.BatchID)
	assert.Nil(t, payload.Entries)
}

func TestProcessAuditBatchResult(t *testing.T) {
	t.Parallel()

	batchID := pulid.MustNew("aeb_")
	result := &ProcessAuditBatchResult{
		ProcessedCount: 10,
		FailedCount:    2,
		BatchID:        batchID,
		ProcessedAt:    99999,
		Errors:         []string{"error1", "error2"},
		Metadata: map[string]any{
			"key": "value",
		},
	}

	assert.Equal(t, 10, result.ProcessedCount)
	assert.Equal(t, 2, result.FailedCount)
	assert.Equal(t, batchID, result.BatchID)
	assert.Equal(t, int64(99999), result.ProcessedAt)
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, "value", result.Metadata["key"])
}

func TestProcessAuditBatchResult_NoErrors(t *testing.T) {
	t.Parallel()

	result := &ProcessAuditBatchResult{
		ProcessedCount: 5,
		FailedCount:    0,
	}

	assert.Nil(t, result.Errors)
	assert.Nil(t, result.Metadata)
}

func TestAuditBufferStatus(t *testing.T) {
	t.Parallel()

	status := &AuditBufferStatus{
		BufferedEntries: 100,
		DLQEntries:      5,
		LastFlush:       1700000000,
	}

	assert.Equal(t, 100, status.BufferedEntries)
	assert.Equal(t, 5, status.DLQEntries)
	assert.Equal(t, int64(1700000000), status.LastFlush)
}

func TestAuditBufferStatus_Zero(t *testing.T) {
	t.Parallel()

	status := &AuditBufferStatus{}

	assert.Equal(t, 0, status.BufferedEntries)
	assert.Equal(t, 0, status.DLQEntries)
	assert.Equal(t, int64(0), status.LastFlush)
}

func TestDeleteAuditEntriesResult(t *testing.T) {
	t.Parallel()

	result := &DeleteAuditEntriesResult{
		TotalDeleted: 42,
		Result:       "Deleted 42 entries",
	}

	assert.Equal(t, 42, result.TotalDeleted)
	assert.Equal(t, "Deleted 42 entries", result.Result)
}

func TestDeleteAuditEntriesResult_Zero(t *testing.T) {
	t.Parallel()

	result := &DeleteAuditEntriesResult{}

	assert.Equal(t, 0, result.TotalDeleted)
	assert.Empty(t, result.Result)
}

func TestFlushFromRedisResult(t *testing.T) {
	t.Parallel()

	entries := []*audit.Entry{
		{ID: pulid.MustNew("ael_")},
		{ID: pulid.MustNew("ael_")},
	}

	result := &FlushFromRedisResult{
		Batches:    [][]*audit.Entry{entries},
		EntryCount: 2,
	}

	assert.Len(t, result.Batches, 1)
	assert.Len(t, result.Batches[0], 2)
	assert.Equal(t, 2, result.EntryCount)
}

func TestFlushFromRedisResult_MultipleBatches(t *testing.T) {
	t.Parallel()

	batch1 := []*audit.Entry{{ID: pulid.MustNew("ael_")}}
	batch2 := []*audit.Entry{{ID: pulid.MustNew("ael_")}, {ID: pulid.MustNew("ael_")}}

	result := &FlushFromRedisResult{
		Batches:    [][]*audit.Entry{batch1, batch2},
		EntryCount: 3,
	}

	assert.Len(t, result.Batches, 2)
	assert.Equal(t, 3, result.EntryCount)
}

func TestFlushFromRedisResult_Empty(t *testing.T) {
	t.Parallel()

	result := &FlushFromRedisResult{
		Batches:    make([][]*audit.Entry, 0),
		EntryCount: 0,
	}

	assert.Empty(t, result.Batches)
	assert.Equal(t, 0, result.EntryCount)
}

func TestMoveToDLQPayload(t *testing.T) {
	t.Parallel()

	entries := []*audit.Entry{
		{ID: pulid.MustNew("ael_")},
	}

	payload := &MoveToDLQPayload{
		Entries:      entries,
		ErrorMessage: "insertion failed",
	}

	assert.Len(t, payload.Entries, 1)
	assert.Equal(t, "insertion failed", payload.ErrorMessage)
}

func TestMoveToDLQPayload_Empty(t *testing.T) {
	t.Parallel()

	payload := &MoveToDLQPayload{}

	assert.Nil(t, payload.Entries)
	assert.Empty(t, payload.ErrorMessage)
}

func TestDLQRetryResult(t *testing.T) {
	t.Parallel()

	id1 := pulid.MustNew("dlq_")
	id2 := pulid.MustNew("dlq_")

	result := &DLQRetryResult{
		RetryCount:     10,
		SuccessCount:   7,
		FailedCount:    2,
		ExhaustedCount: 1,
		RecoveredIDs:   []pulid.ID{id1},
		FailedIDs:      []pulid.ID{id2},
	}

	assert.Equal(t, 10, result.RetryCount)
	assert.Equal(t, 7, result.SuccessCount)
	assert.Equal(t, 2, result.FailedCount)
	assert.Equal(t, 1, result.ExhaustedCount)
	assert.Len(t, result.RecoveredIDs, 1)
	assert.Len(t, result.FailedIDs, 1)
}

func TestDLQRetryResult_Empty(t *testing.T) {
	t.Parallel()

	result := &DLQRetryResult{}

	assert.Equal(t, 0, result.RetryCount)
	assert.Equal(t, 0, result.SuccessCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Equal(t, 0, result.ExhaustedCount)
	assert.Nil(t, result.RecoveredIDs)
	assert.Nil(t, result.FailedIDs)
}

func TestEntryToMap_Success(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		ID:             pulid.MustNew("ael_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Timestamp:      1700000000,
		Resource:       "shipment",
		ResourceID:     "shp_123",
		Operation:      permission.OpCreate,
	}

	result, err := entryToMap(entry)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, string(entry.ID), result["id"])
	assert.Equal(t, string(entry.OrganizationID), result["organizationId"])
	assert.Equal(t, string(entry.BusinessUnitID), result["businessUnitId"])
	assert.Equal(t, "shipment", result["resource"])
	assert.Equal(t, "shp_123", result["resourceId"])
}

func TestEntryToMap_WithMetadata(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		ID:       pulid.MustNew("ael_"),
		Metadata: map[string]any{"key": "value", "count": float64(42)},
	}

	result, err := entryToMap(entry)

	require.NoError(t, err)
	require.NotNil(t, result)
	metadata, ok := result["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", metadata["key"])
}

func TestEntryToMap_EmptyEntry(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}

	result, err := entryToMap(entry)

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestMapToEntry_Success(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ael_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	data := map[string]any{
		"id":             string(id),
		"organizationId": string(orgID),
		"businessUnitId": string(buID),
		"resource":       "worker",
		"resourceId":     "wkr_456",
		"operation":      "update",
		"timestamp":      float64(1700000000),
	}

	entry, err := mapToEntry(data)

	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, id, entry.ID)
	assert.Equal(t, orgID, entry.OrganizationID)
	assert.Equal(t, buID, entry.BusinessUnitID)
	assert.Equal(t, "wkr_456", entry.ResourceID)
}

func TestMapToEntry_EmptyMap(t *testing.T) {
	t.Parallel()

	data := map[string]any{}

	entry, err := mapToEntry(data)

	require.NoError(t, err)
	require.NotNil(t, entry)
}

func TestEntryToMap_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &audit.Entry{
		ID:             pulid.MustNew("ael_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Timestamp:      1700000000,
		Resource:       "location",
		ResourceID:     "loc_789",
		Operation:      permission.OpUpdate,
		Comment:        "test comment",
		IPAddress:      "192.168.1.1",
		UserAgent:      "TestAgent/1.0",
	}

	m, err := entryToMap(original)
	require.NoError(t, err)

	restored, err := mapToEntry(m)
	require.NoError(t, err)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.OrganizationID, restored.OrganizationID)
	assert.Equal(t, original.BusinessUnitID, restored.BusinessUnitID)
	assert.Equal(t, original.UserID, restored.UserID)
	assert.Equal(t, original.Resource, restored.Resource)
	assert.Equal(t, original.ResourceID, restored.ResourceID)
	assert.Equal(t, original.Comment, restored.Comment)
	assert.Equal(t, original.IPAddress, restored.IPAddress)
	assert.Equal(t, original.UserAgent, restored.UserAgent)
}

func TestEntryToMap_NilEntry(t *testing.T) {
	t.Parallel()

	_, err := entryToMap(nil)
	require.NoError(t, err)
}

func TestMapToEntry_NilMap(t *testing.T) {
	t.Parallel()

	entry, err := mapToEntry(nil)
	require.NoError(t, err)
	require.NotNil(t, entry)
}

func TestEntryToMap_WithChanges(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{
		ID: pulid.MustNew("ael_"),
		Changes: map[string]any{
			"name": map[string]any{
				"old": "OldName",
				"new": "NewName",
			},
		},
		PreviousState: map[string]any{"name": "OldName"},
		CurrentState:  map[string]any{"name": "NewName"},
	}

	result, err := entryToMap(entry)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result["changes"])
	assert.NotNil(t, result["previousState"])
	assert.NotNil(t, result["currentState"])
}

func TestMapToEntry_WithAllFields(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"id":             "ael_test123",
		"organizationId": "org_test456",
		"businessUnitId": "bu_test789",
		"userId":         "usr_testabc",
		"timestamp":      float64(1700000000),
		"resource":       "shipment",
		"resourceId":     "shp_xyz",
		"operation":      "create",
		"comment":        "test comment",
		"ipAddress":      "10.0.0.1",
		"userAgent":      "Mozilla/5.0",
		"correlationId":  "corr_123",
		"sensitiveData":  true,
		"critical":       true,
	}

	entry, err := mapToEntry(data)

	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, pulid.ID("ael_test123"), entry.ID)
	assert.Equal(t, "test comment", entry.Comment)
	assert.Equal(t, "10.0.0.1", entry.IPAddress)
	assert.Equal(t, "corr_123", entry.CorrelationID)
	assert.True(t, entry.SensitiveData)
	assert.True(t, entry.Critical)
}

func TestErrors(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, ErrBufferFull)
	assert.Equal(t, "buffer is full", ErrBufferFull.Error())

	assert.NotNil(t, ErrBufferCircuitBreakerOpen)
	assert.Equal(t, "buffer circuit breaker is open", ErrBufferCircuitBreakerOpen.Error())
}
