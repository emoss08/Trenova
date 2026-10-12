package tableconfiguration

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestTableConfiguration_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  TableConfiguration
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: false,
		},
		{
			name: "name empty fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: true,
		},
		{
			name: "name too long fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           strings.Repeat("a", 256),
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: true,
		},
		{
			name: "resource empty fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: true,
		},
		{
			name: "resource too long fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       strings.Repeat("a", 101),
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: true,
		},
		{
			name: "table config nil fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    nil,
				Visibility:     VisibilityPrivate,
			},
			wantErr: true,
		},
		{
			name: "invalid visibility fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     Visibility("Invalid"),
			},
			wantErr: true,
		},
		{
			name: "public visibility passes",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPublic,
			},
			wantErr: false,
		},
		{
			name: "shared visibility passes",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityShared,
			},
			wantErr: false,
		},
		{
			name: "name at max length passes",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           strings.Repeat("a", 255),
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: false,
		},
		{
			name: "resource at max length passes",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       strings.Repeat("r", 100),
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     VisibilityPrivate,
			},
			wantErr: false,
		},
		{
			name: "visibility empty fails",
			entity: TableConfiguration{
				ID:             pulid.MustNew("tc_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         pulid.MustNew("usr_"),
				Name:           "My Config",
				Resource:       "shipments",
				TableConfig:    &TableConfig{PageSize: 20},
				Visibility:     Visibility(""),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.entity.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestTableConfiguration_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		tc := &TableConfiguration{}
		require.True(t, tc.ID.IsNil())

		err := tc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, tc.ID.IsNil())
		assert.NotZero(t, tc.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("tc_")
		tc := &TableConfiguration{ID: existingID}

		err := tc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, tc.ID)
		assert.NotZero(t, tc.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		tc := &TableConfiguration{}

		err := tc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, tc.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		tc := &TableConfiguration{}

		err := tc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, tc.CreatedAt)
		assert.NotZero(t, tc.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		tc := &TableConfiguration{}

		err := tc.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, tc.ID.IsNil())
		assert.Zero(t, tc.CreatedAt)
		assert.Zero(t, tc.UpdatedAt)
	})
}

func TestTableConfiguration_GetTableName(t *testing.T) {
	t.Parallel()

	tc := &TableConfiguration{}
	assert.Equal(t, "table_configurations", tc.GetTableName())
}

func TestTableConfiguration_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tc_")
	tc := &TableConfiguration{ID: id}
	assert.Equal(t, id, tc.GetID())
}

func TestTableConfiguration_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	tc := &TableConfiguration{OrganizationID: orgID}
	assert.Equal(t, orgID, tc.GetOrganizationID())
}

func TestTableConfiguration_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	tc := &TableConfiguration{BusinessUnitID: buID}
	assert.Equal(t, buID, tc.GetBusinessUnitID())
}

func TestTableConfiguration_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	tc := &TableConfiguration{}
	config := tc.GetPostgresSearchConfig()

	assert.Equal(t, "tc", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "name", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
}

func TestVisibility_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		visibility Visibility
		want       string
	}{
		{name: "private", visibility: VisibilityPrivate, want: "Private"},
		{name: "public", visibility: VisibilityPublic, want: "Public"},
		{name: "shared", visibility: VisibilityShared, want: "Shared"},
		{name: "empty", visibility: Visibility(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.visibility.String())
		})
	}
}

func TestVisibilityFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Visibility
		wantErr bool
	}{
		{name: "private", input: "Private", want: VisibilityPrivate, wantErr: false},
		{name: "public", input: "Public", want: VisibilityPublic, wantErr: false},
		{name: "shared", input: "Shared", want: VisibilityShared, wantErr: false},
		{name: "invalid", input: "Unknown", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
		{name: "lowercase", input: "private", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := VisibilityFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
