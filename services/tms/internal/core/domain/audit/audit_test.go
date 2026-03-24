package audit

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validEntry() *Entry {
	return &Entry{
		ID:             pulid.MustNew("ae_"),
		UserID:         pulid.MustNew("usr_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Resource:       permission.Resource("shipment"),
		ResourceID:     "shp_01HXYZ",
		Operation:      permission.OpCreate,
		Category:       CategoryUser,
	}
}

func TestEntry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(e *Entry)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *Entry) {},
			wantErr: false,
		},
		{
			name: "missing organization ID fails",
			modify: func(e *Entry) {
				e.OrganizationID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing business unit ID fails",
			modify: func(e *Entry) {
				e.BusinessUnitID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing resource fails",
			modify: func(e *Entry) {
				e.Resource = permission.Resource("")
			},
			wantErr: true,
		},
		{
			name: "missing resource ID fails",
			modify: func(e *Entry) {
				e.ResourceID = ""
			},
			wantErr: true,
		},
		{
			name: "missing operation fails",
			modify: func(e *Entry) {
				e.Operation = permission.Operation("")
			},
			wantErr: true,
		},
		{
			name: "missing user ID fails",
			modify: func(e *Entry) {
				e.UserID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing category fails",
			modify: func(e *Entry) {
				e.Category = Category("")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := validEntry()
			tt.modify(e)

			err := e.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEntry_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and Timestamp", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}
		require.True(t, e.ID.IsNil())

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, e.ID.IsNil())
		assert.NotZero(t, e.Timestamp)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("ae_")
		e := &Entry{ID: existingID}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, e.ID)
	})

	t.Run("insert does not overwrite existing Timestamp", func(t *testing.T) {
		t.Parallel()

		e := &Entry{Timestamp: 1234567890}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, int64(1234567890), e.Timestamp)
	})

	t.Run("insert sets default category if empty", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, CategorySystem, e.Category)
	})

	t.Run("insert does not overwrite existing category", func(t *testing.T) {
		t.Parallel()

		e := &Entry{Category: CategoryUser}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, CategoryUser, e.Category)
	})

	t.Run("update does not set ID or Timestamp", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.True(t, e.ID.IsNil())
		assert.Zero(t, e.Timestamp)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, e.ID.IsNil())
		assert.Zero(t, e.Timestamp)
	})
}

func TestEntry_GetTableName(t *testing.T) {
	t.Parallel()

	e := &Entry{}
	assert.Equal(t, "audit_entries", e.GetTableName())
}

func TestDLQEntry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   DLQEntry
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entry: DLQEntry{
				OriginalEntryID: pulid.MustNew("ae_"),
				EntryData:       map[string]any{"key": "value"},
				FailureTime:     1234567890,
				OrganizationID:  pulid.MustNew("org_"),
				BusinessUnitID:  pulid.MustNew("bu_"),
			},
			wantErr: false,
		},
		{
			name: "missing original entry ID fails",
			entry: DLQEntry{
				OriginalEntryID: pulid.ID(""),
				EntryData:       map[string]any{"key": "value"},
				FailureTime:     1234567890,
				OrganizationID:  pulid.MustNew("org_"),
				BusinessUnitID:  pulid.MustNew("bu_"),
			},
			wantErr: true,
		},
		{
			name: "missing entry data fails",
			entry: DLQEntry{
				OriginalEntryID: pulid.MustNew("ae_"),
				EntryData:       nil,
				FailureTime:     1234567890,
				OrganizationID:  pulid.MustNew("org_"),
				BusinessUnitID:  pulid.MustNew("bu_"),
			},
			wantErr: true,
		},
		{
			name: "zero failure time fails",
			entry: DLQEntry{
				OriginalEntryID: pulid.MustNew("ae_"),
				EntryData:       map[string]any{"key": "value"},
				FailureTime:     0,
				OrganizationID:  pulid.MustNew("org_"),
				BusinessUnitID:  pulid.MustNew("bu_"),
			},
			wantErr: true,
		},
		{
			name: "missing organization ID fails",
			entry: DLQEntry{
				OriginalEntryID: pulid.MustNew("ae_"),
				EntryData:       map[string]any{"key": "value"},
				FailureTime:     1234567890,
				OrganizationID:  pulid.ID(""),
				BusinessUnitID:  pulid.MustNew("bu_"),
			},
			wantErr: true,
		},
		{
			name: "missing business unit ID fails",
			entry: DLQEntry{
				OriginalEntryID: pulid.MustNew("ae_"),
				EntryData:       map[string]any{"key": "value"},
				FailureTime:     1234567890,
				OrganizationID:  pulid.MustNew("org_"),
				BusinessUnitID:  pulid.ID(""),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.entry.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDLQEntry_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID CreatedAt and UpdatedAt", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{}
		require.True(t, e.ID.IsNil())

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, e.ID.IsNil())
		assert.NotZero(t, e.CreatedAt)
		assert.NotZero(t, e.UpdatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("dlq_")
		e := &DLQEntry{ID: existingID}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, e.ID)
	})

	t.Run("insert sets default status if empty", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, DLQStatusPending, e.Status)
	})

	t.Run("insert does not overwrite existing status", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{Status: DLQStatusRetrying}

		err := e.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, DLQStatusRetrying, e.Status)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, e.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, e.CreatedAt)
		assert.NotZero(t, e.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		e := &DLQEntry{}

		err := e.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, e.ID.IsNil())
		assert.Zero(t, e.CreatedAt)
		assert.Zero(t, e.UpdatedAt)
	})
}

func TestDLQEntry_CanRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entry      DLQEntry
		maxRetries int
		want       bool
	}{
		{
			name:       "pending with retries remaining returns true",
			entry:      DLQEntry{RetryCount: 0, Status: DLQStatusPending},
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "retrying with retries remaining returns true",
			entry:      DLQEntry{RetryCount: 1, Status: DLQStatusRetrying},
			maxRetries: 3,
			want:       true,
		},
		{
			name:       "retry count at max returns false",
			entry:      DLQEntry{RetryCount: 3, Status: DLQStatusPending},
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "recovered status returns false",
			entry:      DLQEntry{RetryCount: 0, Status: DLQStatusRecovered},
			maxRetries: 3,
			want:       false,
		},
		{
			name:       "failed status returns false",
			entry:      DLQEntry{RetryCount: 0, Status: DLQStatusFailed},
			maxRetries: 3,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.entry.CanRetry(tt.maxRetries))
		})
	}
}

func TestDLQEntry_IncrementRetry(t *testing.T) {
	t.Parallel()

	e := &DLQEntry{RetryCount: 0, Status: DLQStatusPending}
	e.IncrementRetry()

	assert.Equal(t, 1, e.RetryCount)
	assert.Equal(t, DLQStatusRetrying, e.Status)

	e.IncrementRetry()
	assert.Equal(t, 2, e.RetryCount)
}

func TestDLQEntry_MarkRecovered(t *testing.T) {
	t.Parallel()

	e := &DLQEntry{Status: DLQStatusRetrying}
	e.MarkRecovered()

	assert.Equal(t, DLQStatusRecovered, e.Status)
}

func TestDLQEntry_MarkFailed(t *testing.T) {
	t.Parallel()

	e := &DLQEntry{Status: DLQStatusRetrying}
	e.MarkFailed("connection timeout")

	assert.Equal(t, DLQStatusFailed, e.Status)
	assert.Equal(t, "connection timeout", e.LastError)
}

func TestEntry_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	e := &Entry{}
	config := e.GetPostgresSearchConfig()

	assert.Equal(t, "ae", config.TableAlias)
	assert.False(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 10)
	assert.Equal(t, "comment", config.SearchableFields[0].Name)
	assert.Equal(t, "ip_address", config.SearchableFields[1].Name)
	assert.Equal(t, "user_agent", config.SearchableFields[2].Name)
	assert.Equal(t, "correlation_id", config.SearchableFields[3].Name)
	assert.Equal(t, "resource", config.SearchableFields[4].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[4].Type)
	assert.Equal(t, "resource_id", config.SearchableFields[5].Name)
	assert.Equal(t, "operation", config.SearchableFields[6].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[6].Type)
	assert.Equal(t, "category", config.SearchableFields[7].Name)
	assert.Equal(t, "critical", config.SearchableFields[8].Name)
	assert.Equal(t, "sensitive_data", config.SearchableFields[9].Name)
}
