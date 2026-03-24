package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validOrganization() *Organization {
	return &Organization{
		ID:             pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("st_"),
		Name:           "Test Organization",
		ScacCode:       "ABCD",
		DOTNumber:      "1234567",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		Timezone:       "America/New_York",
		BucketName:     "test-bucket",
	}
}

func TestOrganization_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(o *Organization)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *Organization) {},
			wantErr: false,
		},
		{
			name: "missing name fails",
			modify: func(o *Organization) {
				o.Name = ""
			},
			wantErr: true,
		},
		{
			name: "missing SCAC code fails",
			modify: func(o *Organization) {
				o.ScacCode = ""
			},
			wantErr: true,
		},
		{
			name: "SCAC code too short fails",
			modify: func(o *Organization) {
				o.ScacCode = "ABC"
			},
			wantErr: true,
		},
		{
			name: "SCAC code too long fails",
			modify: func(o *Organization) {
				o.ScacCode = "ABCDE"
			},
			wantErr: true,
		},
		{
			name: "SCAC code exactly 4 characters passes",
			modify: func(o *Organization) {
				o.ScacCode = "WXYZ"
			},
			wantErr: false,
		},
		{
			name: "missing DOT number fails",
			modify: func(o *Organization) {
				o.DOTNumber = ""
			},
			wantErr: true,
		},
		{
			name: "non-numeric DOT number fails",
			modify: func(o *Organization) {
				o.DOTNumber = "ABCDEFG"
			},
			wantErr: true,
		},
		{
			name: "DOT number too long fails",
			modify: func(o *Organization) {
				o.DOTNumber = "123456789"
			},
			wantErr: true,
		},
		{
			name: "valid numeric DOT number passes",
			modify: func(o *Organization) {
				o.DOTNumber = "12345678"
			},
			wantErr: false,
		},
		{
			name: "missing timezone fails",
			modify: func(o *Organization) {
				o.Timezone = ""
			},
			wantErr: true,
		},
		{
			name: "invalid timezone fails",
			modify: func(o *Organization) {
				o.Timezone = "Not/A/Timezone"
			},
			wantErr: true,
		},
		{
			name: "valid timezone passes",
			modify: func(o *Organization) {
				o.Timezone = "America/Chicago"
			},
			wantErr: false,
		},
		{
			name: "missing city fails",
			modify: func(o *Organization) {
				o.City = ""
			},
			wantErr: true,
		},
		{
			name: "address line 1 too long fails",
			modify: func(o *Organization) {
				o.AddressLine1 = string(make([]byte, 151))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			o := validOrganization()
			tt.modify(o)

			multiErr := errortypes.NewMultiError()
			o.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestOrganization_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		o := &Organization{}
		require.True(t, o.ID.IsNil())

		err := o.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, o.ID.IsNil())
		assert.NotZero(t, o.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("org_")
		o := &Organization{ID: existingID}

		err := o.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, o.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		o := &Organization{}

		err := o.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, o.UpdatedAt)
	})
}
