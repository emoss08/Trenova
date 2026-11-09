package auditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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

	_ = auditports.BulkLogEntry{}
}

func TestWithAction_RefusesACodeThatIsNotDottedLowerCase(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	require.NoError(t, WithAction("role.assigned")(entry))
	assert.Equal(t, "role.assigned", entry.Action)

	require.Error(t, WithAction("Role Assigned")(&audit.Entry{}))
	require.Error(t, WithAction("role")(&audit.Entry{}))
}

func TestWithSecurity_MakesTheEntryACriticalSecurityEntry(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{Category: audit.CategoryUser}
	require.NoError(t, WithSecurity()(entry))
	assert.Equal(t, audit.CategorySecurity, entry.Category)
	assert.True(t, entry.Critical)
}

func TestWithChannel_RefusesAnUnknownChannel(t *testing.T) {
	t.Parallel()

	entry := &audit.Entry{}
	require.NoError(t, WithChannel(audit.ChannelSCIM)(entry))
	assert.Equal(t, audit.ChannelSCIM, entry.Channel)
	require.Error(t, WithChannel(audit.Channel("fax"))(&audit.Entry{}))
}
