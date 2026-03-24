package trailer

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

//go:fix inline
func intPtr(v int) *int {
	return new(v)
}

func validTrailer() *Trailer {
	return &Trailer{
		ID:                      pulid.MustNew("tr_"),
		BusinessUnitID:          pulid.MustNew("bu_"),
		OrganizationID:          pulid.MustNew("org_"),
		EquipmentTypeID:         pulid.MustNew("et_"),
		EquipmentManufacturerID: pulid.MustNew("em_"),
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "TRL-001",
		Year:                    new(2022),
	}
}

func TestTrailer_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(tr *Trailer)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *Trailer) {},
			wantErr: false,
		},
		{
			name: "missing code fails",
			modify: func(tr *Trailer) {
				tr.Code = ""
			},
			wantErr: true,
		},
		{
			name: "code too long fails",
			modify: func(tr *Trailer) {
				tr.Code = string(make([]byte, 51))
			},
			wantErr: true,
		},
		{
			name: "missing equipment type ID fails",
			modify: func(tr *Trailer) {
				tr.EquipmentTypeID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing equipment manufacturer ID fails",
			modify: func(tr *Trailer) {
				tr.EquipmentManufacturerID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "year below 1900 fails",
			modify: func(tr *Trailer) {
				tr.Year = new(1899)
			},
			wantErr: true,
		},
		{
			name: "year above 2099 fails",
			modify: func(tr *Trailer) {
				tr.Year = new(2100)
			},
			wantErr: true,
		},
		{
			name: "year at 1900 passes",
			modify: func(tr *Trailer) {
				tr.Year = new(1900)
			},
			wantErr: false,
		},
		{
			name: "year at 2099 passes",
			modify: func(tr *Trailer) {
				tr.Year = new(2099)
			},
			wantErr: false,
		},
		{
			name: "nil year passes",
			modify: func(tr *Trailer) {
				tr.Year = nil
			},
			wantErr: false,
		},
		{
			name: "valid VIN passes",
			modify: func(tr *Trailer) {
				tr.Vin = "1HGCM82633A004352"
			},
			wantErr: false,
		},
		{
			name: "invalid VIN fails",
			modify: func(tr *Trailer) {
				tr.Vin = "INVALIDVIN"
			},
			wantErr: true,
		},
		{
			name: "empty VIN passes",
			modify: func(tr *Trailer) {
				tr.Vin = ""
			},
			wantErr: false,
		},
		{
			name: "VIN with lowercase fails",
			modify: func(tr *Trailer) {
				tr.Vin = "1hgcm82633a004352"
			},
			wantErr: true,
		},
		{
			name: "VIN with invalid characters fails",
			modify: func(tr *Trailer) {
				tr.Vin = "1HGCM8263IA00435O"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := validTrailer()
			tt.modify(tr)

			multiErr := errortypes.NewMultiError()
			tr.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestTrailer_GetTableName(t *testing.T) {
	t.Parallel()

	tr := &Trailer{}
	assert.Equal(t, "trailers", tr.GetTableName())
}

func TestTrailer_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tr_")
	tr := &Trailer{ID: id}
	assert.Equal(t, id, tr.GetID())
}

func TestTrailer_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	tr := &Trailer{OrganizationID: orgID}
	assert.Equal(t, orgID, tr.GetOrganizationID())
}

func TestTrailer_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	tr := &Trailer{BusinessUnitID: buID}
	assert.Equal(t, buID, tr.GetBusinessUnitID())
}

func TestTrailer_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		tr := &Trailer{}
		require.True(t, tr.ID.IsNil())

		err := tr.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, tr.ID.IsNil())
		assert.NotZero(t, tr.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("tr_")
		tr := &Trailer{ID: existingID}

		err := tr.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, tr.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		tr := &Trailer{}

		err := tr.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, tr.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		tr := &Trailer{}

		err := tr.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, tr.CreatedAt)
		assert.NotZero(t, tr.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		tr := &Trailer{}

		err := tr.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, tr.ID.IsNil())
		assert.Zero(t, tr.CreatedAt)
		assert.Zero(t, tr.UpdatedAt)
	})
}

func TestTrailer_GetResourceType(t *testing.T) {
	t.Parallel()

	tr := &Trailer{}
	assert.Equal(t, "trailer", tr.GetResourceType())
}

func TestTrailer_GetResourceID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tr_")
	tr := &Trailer{ID: id}
	assert.Equal(t, id.String(), tr.GetResourceID())
}

func TestTrailer_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	tr := &Trailer{}
	config := tr.GetPostgresSearchConfig()

	assert.Equal(t, "tr", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 4)
	assert.Equal(t, "code", config.SearchableFields[0].Name)
	assert.Equal(t, "vin", config.SearchableFields[1].Name)
	assert.Equal(t, "license_plate_number", config.SearchableFields[2].Name)
	assert.Equal(t, "registration_number", config.SearchableFields[3].Name)
	assert.Len(t, config.Relationships, 3)
}
