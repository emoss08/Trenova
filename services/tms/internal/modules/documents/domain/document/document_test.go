package document

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
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
