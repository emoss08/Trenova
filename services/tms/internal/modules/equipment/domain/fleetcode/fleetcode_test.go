package fleetcode

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestFleetCode_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  FleetCode
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: false,
		},
		{
			name: "code empty fails",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: true,
		},
		{
			name: "code too long fails",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJK",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: true,
		},
		{
			name: "manager ID empty fails",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				ManagerID:      pulid.ID(""),
			},
			wantErr: true,
		},
		{
			name: "code at max length passes",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJ",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: false,
		},
		{
			name: "code single character passes",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "A",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: false,
		},
		{
			name: "code 11 chars fails",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "12345678901",
				ManagerID:      pulid.MustNew("usr_"),
			},
			wantErr: true,
		},
		{
			name: "both code empty and manager nil fails",
			entity: FleetCode{
				ID:             pulid.MustNew("fc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
				ManagerID:      pulid.ID(""),
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

func TestFleetCode_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		fc := &FleetCode{}
		require.True(t, fc.ID.IsNil())

		err := fc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, fc.ID.IsNil())
		assert.NotZero(t, fc.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("fc_")
		fc := &FleetCode{ID: existingID}

		err := fc.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, fc.ID)
		assert.NotZero(t, fc.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		fc := &FleetCode{}

		err := fc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, fc.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		fc := &FleetCode{}

		err := fc.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, fc.CreatedAt)
		assert.NotZero(t, fc.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		fc := &FleetCode{}

		err := fc.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, fc.ID.IsNil())
		assert.Zero(t, fc.CreatedAt)
		assert.Zero(t, fc.UpdatedAt)
	})
}

func TestFleetCode_GetTableName(t *testing.T) {
	t.Parallel()

	fc := &FleetCode{}
	assert.Equal(t, "fleet_codes", fc.GetTableName())
}

func TestFleetCode_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fc_")
	fc := &FleetCode{ID: id}
	assert.Equal(t, id, fc.GetID())
}

func TestFleetCode_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	fc := &FleetCode{OrganizationID: orgID}
	assert.Equal(t, orgID, fc.GetOrganizationID())
}

func TestFleetCode_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	fc := &FleetCode{BusinessUnitID: buID}
	assert.Equal(t, buID, fc.GetBusinessUnitID())
}

func TestFleetCode_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	fc := &FleetCode{}
	config := fc.GetPostgresSearchConfig()

	assert.Equal(t, "fc", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "code", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
}
