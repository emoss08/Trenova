package passwordutils

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPolicy(t *testing.T) {
	t.Parallel()

	address, err := emailutils.Parse("Dana.Whitfield@Example.com")
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		address  *emailutils.Address
		want     error
	}{
		{name: "accepts a long unusual password", password: "Tr4ck-the-l0ads-north!", address: &address},
		{name: "accepts without an address", password: "Tr4ck-the-l0ads-north!"},
		{name: "rejects eleven characters", password: "Tr4ck-l0ads", want: ErrTooShort},
		{name: "counts runes not bytes", password: "ééééééééééé", want: ErrTooShort},
		{name: "rejects more than 72 bytes", password: strings.Repeat("ab1-", 19), want: ErrTooLong},
		{name: "rejects only spaces", password: strings.Repeat(" ", MinLength), want: ErrBlank},
		{
			name:     "rejects the email address",
			password: "dana.whitfield@example.com",
			address:  &address,
			want:     ErrContainsEmail,
		},
		{
			name:     "rejects the local part",
			password: "MyDana.Whitfield2026",
			address:  &address,
			want:     ErrContainsEmail,
		},
		{name: "rejects a common password", password: "password1234", want: ErrCommon},
		{name: "rejects a sequence", password: "abcdefghijklmnop", want: ErrCommon},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckPolicy(tt.password, tt.address)
			if tt.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}
