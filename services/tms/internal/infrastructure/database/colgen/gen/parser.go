package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type ParseResult struct {
	Models   []ModelInfo
	Warnings []string
}

func ParsePackage(dir string) (*ParseResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{}
	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_gen.go") {
			continue
		}

		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if parseErr != nil {
			return nil, parseErr
		}

		pkgName := file.Name.Name

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				model, warning := parseStruct(pkgName, typeSpec.Name.Name, structType)
				if warning != "" {
					result.Warnings = append(result.Warnings, name+": "+warning)
				}
				if model != nil {
					result.Models = append(result.Models, *model)
				}
			}
		}
	}

	return result, nil
}

func parseStruct(pkgName, structName string, st *ast.StructType) (*ModelInfo, string) {
	var tableName, alias string
	hasBunBase := false

	for _, field := range st.Fields.List {
		if !isBunBaseModel(field) {
			continue
		}
		hasBunBase = true
		if field.Tag != nil {
			tag := unquoteTag(field.Tag.Value)
			bunTag := reflect.StructTag(tag).Get("bun")
			tableName, alias = parseBaseModelTag(bunTag)
		}
		break
	}

	if !hasBunBase {
		return nil, ""
	}

	if tableName == "" || alias == "" {
		missing := "table"
		if tableName != "" {
			missing = "alias"
		}
		return nil, structName + " has bun.BaseModel but missing " + missing + " in tag — skipping"
	}

	model := &ModelInfo{
		PackageName: pkgName,
		StructName:  structName,
		TableName:   tableName,
		Alias:       alias,
	}

	for _, field := range st.Fields.List {
		if isBunBaseModel(field) {
			continue
		}
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue
		}

		if field.Tag == nil {
			continue
		}

		tag := unquoteTag(field.Tag.Value)
		bunTag := reflect.StructTag(tag).Get("bun")
		jsonTag := reflect.StructTag(tag).Get("json")

		if bunTag == "-" {
			continue
		}

		columnName, isRel, isScanOnly, isPK := parseBunFieldTag(bunTag)
		if isRel {
			model.Relations = append(model.Relations, RelationInfo{
				GoName: fieldName,
			})
			continue
		}

		if columnName == "" {
			continue
		}

		jsonName := parseJSONTag(jsonTag)

		model.Fields = append(model.Fields, FieldInfo{
			GoName:     fieldName,
			ColumnName: columnName,
			JSONName:   jsonName,
			IsPK:       isPK,
			IsScanOnly: isScanOnly,
		})
	}

	if len(model.Fields) == 0 {
		return nil, structName + " has bun.BaseModel but no parseable columns — skipping"
	}

	return model, ""
}

func isBunBaseModel(field *ast.Field) bool {
	if len(field.Names) != 0 {
		return false
	}

	switch t := field.Type.(type) {
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "bun" && t.Sel.Name == "BaseModel"
		}
	}
	return false
}

func parseBaseModelTag(bunTag string) (tableName, alias string) {
	for part := range strings.SplitSeq(bunTag, ",") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "table:"); ok {
			tableName = v
		}
		if v, ok := strings.CutPrefix(part, "alias:"); ok {
			alias = v
		}
	}
	return tableName, alias
}

var bunKeywords = map[string]bool{
	"pk":          true,
	"notnull":     true,
	"unique":      true,
	"scanonly":    true,
	"nullzero":    true,
	"soft_delete": true,
	"extend":      true,
	"array":       true,
}

func parseBunFieldTag(bunTag string) (columnName string, isRelation, isScanOnly, isPK bool) {
	if bunTag == "" {
		return "", false, false, false
	}

	if bunTag == "-" {
		return "", false, false, false
	}

	parts := strings.Split(bunTag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "rel:") {
			return "", true, false, false
		}
	}

	// First non-keyword, non-key:value part is the column name
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, ":") {
			continue
		}
		if bunKeywords[part] {
			continue
		}
		if columnName == "" {
			columnName = part
		}
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "pk" {
			isPK = true
		}
		if part == "scanonly" {
			isScanOnly = true
		}
	}

	return columnName, false, isScanOnly, isPK
}

func parseJSONTag(jsonTag string) string {
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}
	name := strings.Split(jsonTag, ",")[0]
	if name == "-" || name == "" {
		return ""
	}
	return name
}

func unquoteTag(raw string) string {
	if len(raw) < 2 {
		return raw
	}
	return raw[1 : len(raw)-1]
}
