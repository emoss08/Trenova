package dataentrycontrol

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validDataEntryControl() *DataEntryControl {
	return &DataEntryControl{
		ID:             pulid.MustNew("dec_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		CodeCase:       CaseFormatUpper,
		NameCase:       CaseFormatTitleCase,
		EmailCase:      CaseFormatLower,
		CityCase:       CaseFormatTitleCase,
	}
}

func TestDataEntryControl_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(dec *DataEntryControl)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *DataEntryControl) {},
			wantErr: false,
		},
		{
			name: "empty code case fails",
			modify: func(dec *DataEntryControl) {
				dec.CodeCase = CaseFormat("")
			},
			wantErr: true,
		},
		{
			name: "invalid code case fails",
			modify: func(dec *DataEntryControl) {
				dec.CodeCase = CaseFormat("Invalid")
			},
			wantErr: true,
		},
		{
			name: "empty name case fails",
			modify: func(dec *DataEntryControl) {
				dec.NameCase = CaseFormat("")
			},
			wantErr: true,
		},
		{
			name: "invalid name case fails",
			modify: func(dec *DataEntryControl) {
				dec.NameCase = CaseFormat("Invalid")
			},
			wantErr: true,
		},
		{
			name: "empty email case fails",
			modify: func(dec *DataEntryControl) {
				dec.EmailCase = CaseFormat("")
			},
			wantErr: true,
		},
		{
			name: "invalid email case fails",
			modify: func(dec *DataEntryControl) {
				dec.EmailCase = CaseFormat("Invalid")
			},
			wantErr: true,
		},
		{
			name: "empty city case fails",
			modify: func(dec *DataEntryControl) {
				dec.CityCase = CaseFormat("")
			},
			wantErr: true,
		},
		{
			name: "invalid city case fails",
			modify: func(dec *DataEntryControl) {
				dec.CityCase = CaseFormat("Invalid")
			},
			wantErr: true,
		},
		{
			name: "all valid case formats pass",
			modify: func(dec *DataEntryControl) {
				dec.CodeCase = CaseFormatAsEntered
				dec.NameCase = CaseFormatLower
				dec.EmailCase = CaseFormatUpper
				dec.CityCase = CaseFormatTitleCase
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dec := validDataEntryControl()
			tt.modify(dec)

			multiErr := errortypes.NewMultiError()
			dec.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestDataEntryControl_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		dec := &DataEntryControl{}
		require.True(t, dec.ID.IsNil())

		err := dec.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, dec.ID.IsNil())
		assert.NotZero(t, dec.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("dec_")
		dec := &DataEntryControl{ID: existingID}

		err := dec.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, dec.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		dec := &DataEntryControl{}

		err := dec.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, dec.UpdatedAt)
	})

	t.Run("update does not set CreatedAt", func(t *testing.T) {
		t.Parallel()

		dec := &DataEntryControl{}

		err := dec.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.Zero(t, dec.CreatedAt)
		assert.NotZero(t, dec.UpdatedAt)
	})

	t.Run("select query does nothing", func(t *testing.T) {
		t.Parallel()

		dec := &DataEntryControl{}

		err := dec.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, dec.ID.IsNil())
		assert.Zero(t, dec.CreatedAt)
		assert.Zero(t, dec.UpdatedAt)
	})
}

func TestDataEntryControl_GetTableName(t *testing.T) {
	t.Parallel()

	dec := &DataEntryControl{}
	assert.Equal(t, "data_entry_controls", dec.GetTableName())
}

func TestDataEntryControl_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dec_")
	dec := &DataEntryControl{ID: id}
	assert.Equal(t, id, dec.GetID())
}

func TestDataEntryControl_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	dec := &DataEntryControl{OrganizationID: orgID}
	assert.Equal(t, orgID, dec.GetOrganizationID())
}

func TestDataEntryControl_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	dec := &DataEntryControl{BusinessUnitID: buID}
	assert.Equal(t, buID, dec.GetBusinessUnitID())
}

func TestCaseFormat_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "AsEntered", CaseFormatAsEntered.String())
	assert.Equal(t, "Upper", CaseFormatUpper.String())
	assert.Equal(t, "Lower", CaseFormatLower.String())
	assert.Equal(t, "TitleCase", CaseFormatTitleCase.String())
}

func TestCaseFormat_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format CaseFormat
		want   bool
	}{
		{name: "AsEntered is valid", format: CaseFormatAsEntered, want: true},
		{name: "Upper is valid", format: CaseFormatUpper, want: true},
		{name: "Lower is valid", format: CaseFormatLower, want: true},
		{name: "TitleCase is valid", format: CaseFormatTitleCase, want: true},
		{name: "empty is invalid", format: CaseFormat(""), want: false},
		{name: "unknown is invalid", format: CaseFormat("Unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.format.IsValid())
		})
	}
}

func TestCaseFormatFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    CaseFormat
		wantErr bool
	}{
		{name: "AsEntered", input: "AsEntered", want: CaseFormatAsEntered, wantErr: false},
		{name: "Upper", input: "Upper", want: CaseFormatUpper, wantErr: false},
		{name: "Lower", input: "Lower", want: CaseFormatLower, wantErr: false},
		{name: "TitleCase", input: "TitleCase", want: CaseFormatTitleCase, wantErr: false},
		{name: "invalid returns error", input: "Invalid", want: CaseFormat(""), wantErr: true},
		{name: "empty returns error", input: "", want: CaseFormat(""), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := CaseFormatFromString(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidCaseFormat)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNewDefaultDataEntryControl(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	dec := NewDefaultDataEntryControl(orgID, buID)

	assert.Equal(t, orgID, dec.OrganizationID)
	assert.Equal(t, buID, dec.BusinessUnitID)
	assert.Equal(t, CaseFormatUpper, dec.CodeCase)
	assert.Equal(t, CaseFormatTitleCase, dec.NameCase)
	assert.Equal(t, CaseFormatLower, dec.EmailCase)
	assert.Equal(t, CaseFormatTitleCase, dec.CityCase)
}
