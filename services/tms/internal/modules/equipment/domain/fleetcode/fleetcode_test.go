package fleetcode

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
