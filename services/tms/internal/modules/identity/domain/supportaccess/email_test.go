package supportaccess_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeStaffEmail_LowerCasesASCIIAndKeepsTheStaffDomain(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"jane.doe@trenova.app", " Jane.Doe@TRENOVA.app ", "JANE.DOE@trenova.app"} {
		got, err := supportaccess.NormalizeStaffEmail(raw, "Trenova.App")
		require.NoError(t, err, raw)
		assert.Equal(t, "jane.doe@trenova.app", got)
	}
}

func TestNormalizeStaffEmail_RefusesAddressesThatCouldPassForStaff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want error
	}{
		{"another domain", "jane@example.com", supportaccess.ErrEmailNotStaff},
		{"a subdomain of the staff domain", "jane@mail.trenova.app", supportaccess.ErrEmailNotStaff},
		{"a domain ending in the staff domain", "jane@nottrenova.app", supportaccess.ErrEmailNotStaff},
		{"a Cyrillic a in the local part", "jаne@trenova.app", supportaccess.ErrEmailInvalid},
		{"a Cyrillic a in the domain", "jane@trenоva.app", supportaccess.ErrEmailInvalid},
		{"a fullwidth at sign", "jane＠trenova.app", supportaccess.ErrEmailInvalid},
		{"a Kelvin sign that lower-cases to k", "Kate@trenova.app", supportaccess.ErrEmailInvalid},
		{"a zero-width space", "jane​@trenova.app", supportaccess.ErrEmailInvalid},
		{"two at signs", "jane@evil.com@trenova.app", supportaccess.ErrEmailInvalid},
		{"a quoted local part", "\"jane\"@trenova.app", supportaccess.ErrEmailInvalid},
		{"a leading dot", ".jane@trenova.app", supportaccess.ErrEmailInvalid},
		{"two dots", "jane..doe@trenova.app", supportaccess.ErrEmailInvalid},
		{"a space inside", "jane doe@trenova.app", supportaccess.ErrEmailInvalid},
		{"no local part", "@trenova.app", supportaccess.ErrEmailInvalid},
		{"nothing", "", supportaccess.ErrEmailInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := supportaccess.NormalizeStaffEmail(tt.raw, "trenova.app")
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestNormalizeStaffEmail_RefusesEveryAddressWithoutAStaffDomain(t *testing.T) {
	t.Parallel()

	for _, domain := range []string{"", "localhost", "trenova..app", "trénova.app"} {
		_, err := supportaccess.NormalizeStaffEmail("jane@trenova.app", domain)
		require.ErrorIs(t, err, supportaccess.ErrStaffDomainUnset, domain)
	}
}
