package shipmenttype

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentType_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  ShipmentType
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: ShipmentType{
				ID:             pulid.MustNew("sht_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
			},
			wantErr: false,
		},
		{
			name: "code empty fails",
			entity: ShipmentType{
				ID:             pulid.MustNew("sht_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
			},
			wantErr: true,
		},
		{
			name: "valid with color passes",
			entity: ShipmentType{
				ID:             pulid.MustNew("sht_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Color:          "#FF0000",
			},
			wantErr: false,
		},
		{
			name: "invalid color fails",
			entity: ShipmentType{
				ID:             pulid.MustNew("sht_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Color:          "notacolor",
			},
			wantErr: true,
		},
		{
			name: "code single character passes",
			entity: ShipmentType{
				ID:             pulid.MustNew("sht_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "A",
			},
			wantErr: false,
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

func TestShipmentType_GetTableName(t *testing.T) {
	t.Parallel()

	sht := &ShipmentType{}
	assert.Equal(t, "shipment_types", sht.GetTableName())
}

func TestShipmentType_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("sht_")
	sht := &ShipmentType{ID: id}
	assert.Equal(t, id, sht.GetID())
}

func TestShipmentType_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	sht := &ShipmentType{OrganizationID: orgID}
	assert.Equal(t, orgID, sht.GetOrganizationID())
}

func TestShipmentType_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	sht := &ShipmentType{BusinessUnitID: buID}
	assert.Equal(t, buID, sht.GetBusinessUnitID())
}

func TestShipmentType_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	sht := &ShipmentType{}
	config := sht.GetPostgresSearchConfig()

	assert.Equal(t, "sht", config.TableAlias)
	assert.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "code", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
}
