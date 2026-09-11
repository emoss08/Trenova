// Package schemalint checks Go struct tags against the schema they are stored
// in, for mismatches the compiler cannot see.
//
// Bun writes the DEFAULT keyword in place of any zero value whose field
// declares a default, and it tests only that a default exists — never what it
// is. On a boolean column that defaults to TRUE this means false never reaches
// the database on an insert: the value the caller asked for is replaced by the
// opposite one. An upsert that feeds the same insert into
// "SET col = EXCLUDED.col" carries the substitution into the update too, so the
// flag can never be turned off at all.
package schemalint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// BoolField is a boolean struct field that declares a bun default, together
// with the table and column it is stored in.
type BoolField struct {
	Table  string
	Column string
	Struct string
	Field  string
	File   string
	Line   int
}

// Key identifies the stored column, which is what the schema is queried by.
func (f *BoolField) Key() string { return f.Table + "." + f.Column }

// Location reports where the field is declared, for a failure message that
// points at the line to edit.
func (f *BoolField) Location() string {
	return fmt.Sprintf("%s:%d (%s.%s)", f.File, f.Line, f.Struct, f.Field)
}

// BoolFieldsWithDefaultTag returns every bool field under domainDir whose bun
// tag declares a default. Fields on structs that do not name a table are
// skipped: they are not stored on their own.
func BoolFieldsWithDefaultTag(domainDir string) ([]BoolField, error) {
	found := make([]BoolField, 0, 128)
	fset := token.NewFileSet()

	err := filepath.WalkDir(domainDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}

		rel, relErr := filepath.Rel(domainDir, path)
		if relErr != nil {
			rel = path
		}

		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			strct, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			table := tableOf(strct)
			if table == "" {
				return true
			}
			for _, field := range strct.Fields.List {
				entry, isBoolDefault := boolDefaultField(fset, field, table, spec.Name.Name, rel)
				if isBoolDefault {
					found = append(found, entry)
				}
			}

			return true
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return found, nil
}

// tableOf reads the table name off the struct's bun.BaseModel tag.
func tableOf(strct *ast.StructType) string {
	for _, field := range strct.Fields.List {
		tag, ok := bunTag(field)
		if !ok || !strings.HasPrefix(tag, "table:") {
			continue
		}
		table := strings.TrimPrefix(strings.Split(tag, ",")[0], "table:")

		return table
	}

	return ""
}

func boolDefaultField(
	fset *token.FileSet,
	field *ast.Field,
	table, structName, file string,
) (BoolField, bool) {
	ident, ok := field.Type.(*ast.Ident)
	if !ok || ident.Name != "bool" || len(field.Names) == 0 {
		return BoolField{}, false
	}

	tag, ok := bunTag(field)
	if !ok {
		return BoolField{}, false
	}

	parts := strings.Split(tag, ",")
	hasDefault := false
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "default:") {
			hasDefault = true
			break
		}
	}
	if !hasDefault || parts[0] == "" {
		return BoolField{}, false
	}

	return BoolField{
		Table:  table,
		Column: parts[0],
		Struct: structName,
		Field:  field.Names[0].Name,
		File:   file,
		Line:   fset.Position(field.Pos()).Line,
	}, true
}

func bunTag(field *ast.Field) (string, bool) {
	if field.Tag == nil {
		return "", false
	}
	// The tag literal carries its backquotes; StructTag wants them removed.
	value := strings.Trim(field.Tag.Value, "`")
	tag, ok := lookupTag(value, "bun")

	return tag, ok
}

// lookupTag reads one key out of a struct tag. reflect.StructTag.Get would do
// this, but it silently returns "" for a malformed tag, and a tag this lint
// cannot read has to be a parse error rather than a skipped field.
func lookupTag(tag, key string) (string, bool) {
	idx := strings.Index(tag, key+`:"`)
	if idx < 0 {
		return "", false
	}
	rest := tag[idx+len(key)+2:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return "", false
	}

	return rest[:end], true
}
