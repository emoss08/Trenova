package auditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ownedRecord struct {
	ID       pulid.ID `json:"id"`
	Resource permission.Resource
	Owner    pulid.ID `json:"workerId"`
}

func (r *ownedRecord) AuditSubject() (permission.Resource, pulid.ID) {
	if r == nil {
		return "", pulid.Nil
	}
	return r.Resource, r.Owner
}

func TestWithSubject_WritesTheMetadataAndTheColumnsTogether(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	entry := &audit.Entry{Metadata: map[string]any{"existing": "kept"}}

	require.NoError(t, WithSubject(permission.ResourceWorker, workerID)(entry))

	assert.Equal(t, "worker", entry.Metadata[audit.MetadataSubjectResource])
	assert.Equal(t, workerID.String(), entry.Metadata[audit.MetadataSubjectID])
	assert.Equal(t, "kept", entry.Metadata["existing"])
	assert.Equal(t, permission.ResourceWorker, entry.SubjectResource)
	assert.Equal(t, workerID, entry.SubjectID)
}

func TestWithSubject_RefusesAKindThatKeepsNoFileAndAMissingID(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	require.Error(t, WithSubject(permission.ResourceShipment, pulid.MustNew("shp_"))(entry))
	require.Error(t, WithSubject(permission.ResourceWorker, pulid.Nil)(entry))
	assert.Empty(t, entry.Metadata)
	assert.True(t, entry.SubjectID.IsNil())
}

func TestWithSubjectOf_TakesTheFirstRecordThatBelongsToAFile(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	var missing *ownedRecord
	entry := &audit.Entry{}

	opt := WithSubjectOf(
		nil,
		map[string]any{"workerId": "wrk_ignored"},
		missing,
		&ownedRecord{Resource: permission.ResourceShipment, Owner: pulid.MustNew("shp_")},
		&ownedRecord{Resource: permission.ResourceWorker, Owner: workerID},
	)
	require.NoError(t, opt(entry))

	assert.Equal(t, permission.ResourceWorker, entry.SubjectResource)
	assert.Equal(t, workerID, entry.SubjectID)
}

func TestWithSubjectOf_LeavesAnEntryWithoutAnOwnerUnattributed(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	require.NoError(t, WithSubjectOf(&testEntity{ID: pulid.MustNew("tst_")}, nil)(entry))

	assert.Nil(t, entry.Metadata)
	assert.Empty(t, entry.SubjectResource)
	assert.True(t, entry.SubjectID.IsNil())
}

func TestLogAction_StoresTheSubjectSealedInMetadataAndProjectedToColumns(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	workerID := pulid.MustNew("wrk_")
	credential := &ownedRecord{
		ID:       pulid.MustNew("wcr_"),
		Resource: permission.ResourceWorker,
		Owner:    workerID,
	}

	params := validLogActionParams()
	params.Resource = permission.ResourceWorkerCredential
	params.ResourceID = credential.ID.String()
	require.NoError(t, srv.LogAction(t.Context(), params, WithSubjectOf(credential)))

	entries := repo.entries()
	require.Len(t, entries, 1)
	entry := entries[0]
	assert.Equal(t, permission.ResourceWorker, entry.SubjectResource)
	assert.Equal(t, workerID, entry.SubjectID)
	assert.Equal(t, workerID.String(), entry.Metadata[audit.MetadataSubjectID])
	assert.Equal(t, "worker", entry.Metadata[audit.MetadataSubjectResource])
}

func TestLogAction_ProjectsTheColumnsFromMetadataWhateverSetThem(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	workerID := pulid.MustNew("wrk_")

	params := validLogActionParams()
	require.NoError(t, srv.LogAction(t.Context(), params,
		WithMetadata(map[string]any{
			audit.MetadataSubjectResource: "worker",
			audit.MetadataSubjectID:       workerID.String(),
		}),
		func(entry *audit.Entry) error {
			entry.SubjectID = pulid.MustNew("wrk_")
			return nil
		},
	))

	entries := repo.entries()
	require.Len(t, entries, 1)
	assert.Equal(t, workerID, entries[0].SubjectID,
		"the column is the sealed metadata's projection, never set apart from it")

	repo = &fakeRepo{}
	srv = newTestService(repo, &noopRealtimeService{})
	require.NoError(t, srv.LogAction(t.Context(), validLogActionParams(),
		WithMetadata(map[string]any{
			audit.MetadataSubjectResource: "shipment",
			audit.MetadataSubjectID:       "shp_1",
		}),
	))
	entries = repo.entries()
	require.Len(t, entries, 1)
	assert.True(t, entries[0].SubjectID.IsNil(), "a kind that keeps no file is not projected")
	assert.Empty(t, entries[0].SubjectResource)
}

func TestBuildBulkLogEntries_NamesEachRecordsSubject(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	built := BuildBulkLogEntries(&BulkLogEntriesParams[*bulkOwned]{
		Resource:  permission.ResourceWorkerCredential,
		Operation: permission.OpUpdate,
		UserID:    pulid.MustNew("usr_"),
		Updated: []*bulkOwned{{
			testEntity: testEntity{
				ID:             pulid.MustNew("wcr_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			},
			WorkerID: workerID,
		}},
	})
	require.Len(t, built, 1)

	entry := &audit.Entry{}
	for _, opt := range built[0].Options {
		require.NoError(t, opt(entry))
	}
	assert.Equal(t, workerID, entry.SubjectID)
}

type bulkOwned struct {
	testEntity

	WorkerID pulid.ID `json:"workerId"`
}

func (b *bulkOwned) AuditSubject() (permission.Resource, pulid.ID) {
	return permission.ResourceWorker, b.WorkerID
}
