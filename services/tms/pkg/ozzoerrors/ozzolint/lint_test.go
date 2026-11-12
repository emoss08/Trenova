package ozzolint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const moduleRoot = "../../.."

/*
ozzo's ValidateStruct finds each field by its address among the struct's own
fields. The address of a nested struct's field (`&r.TenantInfo.OrgID`) is not
one of them, so the whole run fails with "field #0 cannot be found in the
struct" and the request answers 500. Validate the nested struct as a field
instead: `validation.Field(&r.TenantInfo)` runs its own Validate, and a
`validation.By` rule validates one that has none.
*/
func TestNoValidatedFieldIsOfANestedStruct(t *testing.T) {
	t.Parallel()

	for _, dir := range []string{"internal", "pkg", "cmd"} {
		findings, err := NestedFields(filepath.Join(moduleRoot, dir))
		require.NoError(t, err)
		for _, finding := range findings {
			t.Error(finding.String())
		}
	}
}

func TestNestedFieldsFindsTheAddressOfANestedField(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := `package sample

import validation "github.com/go-ozzo/ozzo-validation/v4"

type inner struct{ ID string }
type outer struct {
	Name  string
	Inner inner
}

func (o *outer) Validate() error {
	return validation.ValidateStruct(
		o,
		validation.Field(&o.Name, validation.Required),
		validation.Field(&o.Inner),
		validation.Field(&o.Inner.ID, validation.Required),
	)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sample.go"), []byte(source), 0o600))

	findings, err := NestedFields(dir)
	require.NoError(t, err)
	require.Len(t, findings, 1)
	require.Equal(t, "o.Inner.ID", findings[0].Expr)
	require.Equal(t, 16, findings[0].Line)
}
