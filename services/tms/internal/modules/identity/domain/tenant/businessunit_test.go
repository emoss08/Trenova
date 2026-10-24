package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
