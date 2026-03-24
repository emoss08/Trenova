package accessorialcharge

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestAccessorialCharge_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  AccessorialCharge
		wantErr bool
	}{
		{
			name: "valid flat without rate unit passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Description:    "test",
				Method:         MethodFlat,
				Amount:         decimal.NewFromFloat(75.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid per unit with mile passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "DIS",
				Description:    "per mile charge",
				Method:         MethodPerUnit,
				RateUnit:       RateUnitMile,
				Amount:         decimal.NewFromFloat(1.50),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid per unit with hour passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "DET",
				Description:    "detention charge",
				Method:         MethodPerUnit,
				RateUnit:       RateUnitHour,
				Amount:         decimal.NewFromFloat(25.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid per unit with day passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "STG",
				Description:    "storage charge",
				Method:         MethodPerUnit,
				RateUnit:       RateUnitDay,
				Amount:         decimal.NewFromFloat(50.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid per unit with stop passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "STP",
				Description:    "stop-off charge",
				Method:         MethodPerUnit,
				RateUnit:       RateUnitStop,
				Amount:         decimal.NewFromFloat(15.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid percentage without rate unit passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "PCT",
				Description:    "percentage charge",
				Method:         MethodPercentage,
				Amount:         decimal.NewFromFloat(5.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "per unit without rate unit fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "DIS",
				Description:    "per unit charge",
				Method:         MethodPerUnit,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "flat with rate unit set fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Description:    "test",
				Method:         MethodFlat,
				RateUnit:       RateUnitMile,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "percentage with rate unit set fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "PCT",
				Description:    "percentage charge",
				Method:         MethodPercentage,
				RateUnit:       RateUnitHour,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "code too short fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "AB",
				Description:    "test",
				Method:         MethodFlat,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "code too long fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJK",
				Description:    "test",
				Method:         MethodFlat,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "code empty fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "",
				Description:    "test",
				Method:         MethodFlat,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "description empty fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Description:    "",
				Method:         MethodFlat,
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "invalid method fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Description:    "test",
				Method:         Method("Invalid"),
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "method empty fails",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABC",
				Description:    "test",
				Method:         Method(""),
				Status:         domaintypes.StatusActive,
			},
			wantErr: true,
		},
		{
			name: "code at min length passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "XYZ",
				Description:    "test",
				Method:         MethodFlat,
				Amount:         decimal.NewFromFloat(75.00),
				Status:         domaintypes.StatusActive,
			},
			wantErr: false,
		},
		{
			name: "code at max length passes",
			entity: AccessorialCharge{
				ID:             pulid.MustNew("acc_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				OrganizationID: pulid.MustNew("org_"),
				Code:           "ABCDEFGHIJ",
				Description:    "test",
				Method:         MethodFlat,
				Amount:         decimal.NewFromFloat(75.00),
				Status:         domaintypes.StatusActive,
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

func TestAccessorialCharge_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		ac := &AccessorialCharge{}
		require.True(t, ac.ID.IsNil())

		err := ac.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, ac.ID.IsNil())
		assert.NotZero(t, ac.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("acc_")
		ac := &AccessorialCharge{ID: existingID}

		err := ac.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, ac.ID)
		assert.NotZero(t, ac.CreatedAt)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		ac := &AccessorialCharge{}

		err := ac.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, ac.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		ac := &AccessorialCharge{}

		err := ac.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, ac.CreatedAt)
		assert.NotZero(t, ac.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		ac := &AccessorialCharge{}

		err := ac.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, ac.ID.IsNil())
		assert.Zero(t, ac.CreatedAt)
		assert.Zero(t, ac.UpdatedAt)
	})
}

func TestAccessorialCharge_GetTableName(t *testing.T) {
	t.Parallel()

	ac := &AccessorialCharge{}
	assert.Equal(t, "accessorial_charges", ac.GetTableName())
}

func TestAccessorialCharge_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("acc_")
	ac := &AccessorialCharge{ID: id}
	assert.Equal(t, id, ac.GetID())
}

func TestAccessorialCharge_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	ac := &AccessorialCharge{OrganizationID: orgID}
	assert.Equal(t, orgID, ac.GetOrganizationID())
}

func TestAccessorialCharge_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	ac := &AccessorialCharge{BusinessUnitID: buID}
	assert.Equal(t, buID, ac.GetBusinessUnitID())
}

func TestAccessorialCharge_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	ac := &AccessorialCharge{}
	config := ac.GetPostgresSearchConfig()

	assert.Equal(t, "acc", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "code", config.SearchableFields[0].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
	assert.Equal(t, domaintypes.SearchWeightA, config.SearchableFields[0].Weight)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[1].Type)
	assert.Equal(t, domaintypes.SearchWeightB, config.SearchableFields[1].Weight)
}
