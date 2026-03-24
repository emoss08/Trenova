package auditservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEntity struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Name           string   `json:"name"`
}

func (t *testEntity) GetID() pulid.ID                   { return t.ID }
func (t *testEntity) GetOrganizationID() pulid.ID       { return t.OrganizationID }
func (t *testEntity) GetBusinessUnitID() pulid.ID       { return t.BusinessUnitID }
func (t *testEntity) Validate(_ *errortypes.MultiError) {}
func (t *testEntity) GetTableName() string              { return "test_entities" }

func TestWithComment(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithComment("test comment")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "test comment", entry.Comment)
}

func TestWithMetadata(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithMetadata(map[string]any{"key": "val"})
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "val", entry.Metadata["key"])
}

func TestWithCategory(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithCategory(audit.CategorySystem)
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, audit.CategorySystem, entry.Category)
}

func TestWithCritical(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithCritical()
	err := opt(entry)

	require.NoError(t, err)
	assert.True(t, entry.Critical)
}

func TestWithIP(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithIP("192.168.1.1")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", entry.IPAddress)
}

func TestWithCorrelationID(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithCorrelationID()
	err := opt(entry)

	require.NoError(t, err)
	assert.NotEmpty(t, entry.CorrelationID)
	assert.Contains(t, entry.CorrelationID, "corr_")
}

func TestWithCustomCorrelationID(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithCustomCorrelationID("custom-id")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "custom-id", entry.CorrelationID)
}

func TestWithUserAgent(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithUserAgent("Mozilla/5.0")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "Mozilla/5.0", entry.UserAgent)
}

func TestWithLocation(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithLocation("New York")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "New York", entry.Metadata["location"])
}

func TestWithSessionID(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithSessionID("sess-123")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "sess-123", entry.Metadata["sessionId"])
}

func TestWithTags(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	opt := WithTags("tag1", "tag2")
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, "tag1,tag2", entry.Metadata["tags"])
}

func TestWithTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entry := &audit.Entry{}
	opt := WithTimestamp(now)
	err := opt(entry)

	require.NoError(t, err)
	assert.Equal(t, now.Unix(), entry.Timestamp)
}

func TestWithDiff(t *testing.T) {
	t.Parallel()

	before := &testEntity{Name: "Old Name"}
	after := &testEntity{Name: "New Name"}

	entry := &audit.Entry{}
	opt := WithDiff(before, after)
	err := opt(entry)

	require.NoError(t, err)
	assert.NotNil(t, entry.Changes)
	assert.Contains(t, entry.Changes, "name")
}

func TestWithDiff_NoDifference(t *testing.T) {
	t.Parallel()

	entity := &testEntity{Name: "Same Name"}

	entry := &audit.Entry{}
	opt := WithDiff(entity, entity)
	err := opt(entry)

	require.NoError(t, err)
	assert.Empty(t, entry.Changes)
}

func TestWithCompactDiff(t *testing.T) {
	t.Parallel()

	before := &testEntity{Name: "Old Name"}
	after := &testEntity{Name: "New Name"}

	entry := &audit.Entry{}
	opt := WithCompactDiff(before, after)
	err := opt(entry)

	require.NoError(t, err)
	assert.NotNil(t, entry.Changes)
	assert.Contains(t, entry.Changes, "name")
}

func TestNewBulkLogEntry(t *testing.T) {
	t.Parallel()

	params := &services.LogActionParams{
		Resource:   permission.Resource("test"),
		Operation:  permission.OpCreate,
		ResourceID: "test-id",
	}

	entry := NewBulkLogEntry(params, WithComment("bulk"))
	assert.Equal(t, params, entry.Params)
	assert.Len(t, entry.Options, 1)
}

func TestBuildBulkLogEntries(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	id1 := pulid.MustNew("te_")
	id2 := pulid.MustNew("te_")

	originals := []*testEntity{
		{ID: id1, OrganizationID: orgID, BusinessUnitID: buID, Name: "Original1"},
		{ID: id2, OrganizationID: orgID, BusinessUnitID: buID, Name: "Original2"},
	}

	updated := []*testEntity{
		{ID: id1, OrganizationID: orgID, BusinessUnitID: buID, Name: "Updated1"},
		{ID: id2, OrganizationID: orgID, BusinessUnitID: buID, Name: "Updated2"},
	}

	params := &BulkLogEntriesParams[*testEntity]{
		Resource:  permission.Resource("test_entity"),
		Operation: permission.OpUpdate,
		UserID:    userID,
		Updated:   updated,
		Originals: originals,
	}

	entries := BuildBulkLogEntries(params)

	require.Len(t, entries, 2)

	for i, entry := range entries {
		assert.Equal(t, permission.Resource("test_entity"), entry.Params.Resource)
		assert.Equal(t, permission.OpUpdate, entry.Params.Operation)
		assert.Equal(t, userID, entry.Params.UserID)
		assert.Equal(t, orgID, entry.Params.OrganizationID)
		assert.Equal(t, buID, entry.Params.BusinessUnitID)
		assert.Equal(t, updated[i].GetID().String(), entry.Params.ResourceID)
		assert.NotNil(t, entry.Params.CurrentState)
		assert.NotNil(t, entry.Params.PreviousState)
	}

	_ = services.BulkLogEntry{}
}
