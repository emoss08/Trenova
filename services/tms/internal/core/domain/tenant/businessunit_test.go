package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestBusinessUnit_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bu      *BusinessUnit
		wantErr bool
	}{
		{"valid entity passes", &BusinessUnit{Name: "Test Unit", Code: "TU01"}, false},
		{"missing name fails", &BusinessUnit{Name: "", Code: "TU01"}, true},
		{"name too long fails", &BusinessUnit{Name: string(make([]byte, 101)), Code: "TU01"}, true},
		{"name with special chars fails", &BusinessUnit{Name: "Test@Unit!", Code: "TU01"}, true},
		{"name with ampersand passes", &BusinessUnit{Name: "Test & Unit", Code: "TU01"}, false},
		{"name with period passes", &BusinessUnit{Name: "Test Inc.", Code: "TU01"}, false},
		{"missing code fails", &BusinessUnit{Name: "Test Unit", Code: ""}, true},
		{"code too short fails", &BusinessUnit{Name: "Test Unit", Code: "A"}, true},
		{"code too long fails", &BusinessUnit{Name: "Test Unit", Code: "ABCDEFGHIJK"}, true},
		{"code with lowercase fails", &BusinessUnit{Name: "Test Unit", Code: "abc"}, true},
		{"code at max length passes", &BusinessUnit{Name: "Test Unit", Code: "ABCDEFGHIJ"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tt.bu.Validate(multiErr)
			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestBusinessUnit_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()
		bu := &BusinessUnit{}
		err := bu.BeforeAppendModel(nil, (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, bu.ID.IsNil())
		assert.NotZero(t, bu.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("bu_")
		bu := &BusinessUnit{ID: existingID}
		err := bu.BeforeAppendModel(nil, (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, bu.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()
		bu := &BusinessUnit{}
		err := bu.BeforeAppendModel(nil, (*bun.UpdateQuery)(nil))
		require.NoError(t, err)
		assert.NotZero(t, bu.UpdatedAt)
	})
}
