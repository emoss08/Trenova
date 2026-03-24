package document

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestDocument_GetTableName(t *testing.T) {
	t.Parallel()

	d := &Document{}
	assert.Equal(t, "documents", d.GetTableName())
}

func TestDocument_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("doc_")
	d := &Document{ID: id}
	assert.Equal(t, id, d.GetID())
}

func TestDocument_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	d := &Document{OrganizationID: orgID}
	assert.Equal(t, orgID, d.GetOrganizationID())
}

func TestDocument_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	d := &Document{BusinessUnitID: buID}
	assert.Equal(t, buID, d.GetBusinessUnitID())
}

func TestDocument_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID CreatedAt and UpdatedAt", func(t *testing.T) {
		t.Parallel()

		d := &Document{}
		require.True(t, d.ID.IsNil())

		err := d.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, d.ID.IsNil())
		assert.NotZero(t, d.CreatedAt)
		assert.NotZero(t, d.UpdatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("doc_")
		d := &Document{ID: existingID}

		err := d.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, d.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		d := &Document{}

		err := d.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, d.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		d := &Document{}

		err := d.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, d.CreatedAt)
		assert.NotZero(t, d.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		d := &Document{}

		err := d.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, d.ID.IsNil())
		assert.Zero(t, d.CreatedAt)
		assert.Zero(t, d.UpdatedAt)
	})
}

func TestStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{name: "draft is valid", status: StatusDraft, want: true},
		{name: "active is valid", status: StatusActive, want: true},
		{name: "archived is valid", status: StatusArchived, want: true},
		{name: "expired is valid", status: StatusExpired, want: true},
		{name: "pending is valid", status: StatusPending, want: true},
		{name: "rejected is valid", status: StatusRejected, want: true},
		{name: "pending approval is valid", status: StatusPendingApproval, want: true},
		{name: "empty is invalid", status: Status(""), want: false},
		{name: "unknown is invalid", status: Status("Unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestStatus_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Active", StatusActive.String())
	assert.Equal(t, "Draft", StatusDraft.String())
	assert.Equal(t, "Archived", StatusArchived.String())
	assert.Equal(t, "Expired", StatusExpired.String())
	assert.Equal(t, "Pending", StatusPending.String())
	assert.Equal(t, "Rejected", StatusRejected.String())
	assert.Equal(t, "PendingApproval", StatusPendingApproval.String())
}

func TestDocument_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	d := &Document{}
	config := d.GetPostgresSearchConfig()

	assert.Equal(t, "doc", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 3)
	assert.Equal(t, "file_name", config.SearchableFields[0].Name)
	assert.Equal(t, "original_name", config.SearchableFields[1].Name)
	assert.Equal(t, "description", config.SearchableFields[2].Name)
}
