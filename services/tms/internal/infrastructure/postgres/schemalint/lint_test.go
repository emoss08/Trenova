package schemalint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleDomain = `package sample

import "github.com/uptrace/bun"

type Control struct {
	bun.BaseModel ` + "`bun:\"table:controls\"`" + `

	Named        bool ` + "`bun:\"named_flag,type:BOOLEAN,notnull,default:true\"`" + `
	LeadingComma bool ` + "`bun:\",default:true\"`" + `
	NoColumn     bool ` + "`bun:\"default:true\"`" + `
	AutoDelay    bool ` + "`bun:\"default:true\"`" + `
	NoDefault    bool ` + "`bun:\"no_default,type:BOOLEAN,notnull\"`" + `
	NotABool     string ` + "`bun:\"text_col,default:'x'\"`" + `
}

type Unstored struct {
	Ignored bool ` + "`bun:\"ignored,default:true\"`" + `
}
`

func writeSample(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sample.go"), []byte(sampleDomain), 0o600))

	return dir
}

// TestBoolFieldsWithDefaultTag covers the tag forms bun accepts, including the
// two that omit the column name and so have to be matched against the name bun
// derives from the field.
func TestBoolFieldsWithDefaultTag(t *testing.T) {
	t.Parallel()

	fields, err := BoolFieldsWithDefaultTag(writeSample(t))
	require.NoError(t, err)

	byKey := make(map[string]BoolField, len(fields))
	for i := range fields {
		byKey[fields[i].Key()] = fields[i]
	}

	for _, key := range []string{
		"controls.named_flag",
		"controls.leading_comma",
		"controls.no_column",
		"controls.auto_delay",
	} {
		assert.Contains(t, byKey, key, "%s declares a bun default and must be reported", key)
	}

	assert.NotContains(t, byKey, "controls.no_default", "a field with no default must be skipped")
	assert.NotContains(t, byKey, "controls.text_col", "a non-bool field must be skipped")
	assert.NotContains(t, byKey, "ignored", "a struct that names no table must be skipped")
	assert.Len(t, fields, 4)
}

func TestUnderscoreMatchesBun(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"LeadingComma": "leading_comma",
		"AutoDelay":    "auto_delay",
		"ID":           "id",
		"DOTNumber":    "dot_number",
		"Loaded":       "loaded",
		"AutoBill":     "auto_bill",
	} {
		assert.Equal(t, want, underscore(input), "underscore(%q)", input)
	}
}
