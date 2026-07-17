package structparse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type ParseResult struct {
	Models   []Model
	Enums    []EnumDef
	Warnings []string
}

func (r *ParseResult) Enum(typeName string) (*EnumDef, bool) {
	for i := range r.Enums {
		if r.Enums[i].TypeName == typeName {
			return &r.Enums[i], true
		}
	}
	return nil, false
}

func ParsePackage(dir string) (*ParseResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{}
	fset := token.NewFileSet()
	enumTypes := make(map[string]bool)
	enumValues := make(map[string][]string)
	var enumOrder []string

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

		file, parseErr := parser.ParseFile(
			fset,
			filepath.Join(dir, name),
			nil,
			parser.ParseComments,
		)
		if parseErr != nil {
			return nil, parseErr
		}

		pkgName := file.Name.Name

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			switch genDecl.Tok {
			case token.TYPE:
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if ident, isIdent := typeSpec.Type.(*ast.Ident); isIdent &&
						ident.Name == "string" {
						if !enumTypes[typeSpec.Name.Name] {
							enumTypes[typeSpec.Name.Name] = true
							enumOrder = append(enumOrder, typeSpec.Name.Name)
						}
						continue
					}

					structType, isStruct := typeSpec.Type.(*ast.StructType)
					if !isStruct {
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
			case token.CONST:
				collectEnumValues(genDecl, enumValues)
			default:
			}
		}
	}

	for _, typeName := range enumOrder {
		values := enumValues[typeName]
		if len(values) == 0 {
			continue
		}
		result.Enums = append(result.Enums, EnumDef{TypeName: typeName, Values: values})
	}

	return result, nil
}

func collectEnumValues(decl *ast.GenDecl, enumValues map[string][]string) {
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			if !ast.IsExported(name.Name) || i >= len(valueSpec.Values) {
				continue
			}

			typeName, value, found := enumConstant(valueSpec, valueSpec.Values[i])
			if !found {
				continue
			}
			enumValues[typeName] = append(enumValues[typeName], value)
		}
	}
}

func enumConstant(spec *ast.ValueSpec, expr ast.Expr) (typeName, value string, found bool) {
	if typeIdent, ok := spec.Type.(*ast.Ident); ok {
		if lit, isLit := expr.(*ast.BasicLit); isLit && lit.Kind == token.STRING {
			unquoted, err := strconv.Unquote(lit.Value)
			if err != nil {
				return "", "", false
			}
			return typeIdent.Name, unquoted, true
		}
		return "", "", false
	}

	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return "", "", false
	}
	funIdent, ok := call.Fun.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", "", false
	}
	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", "", false
	}

	return funIdent.Name, unquoted, true
}

func parseStruct(pkgName, structName string, st *ast.StructType) (*Model, string) {
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

	model := &Model{
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

		if isBunEmbed(bunTag) {
			continue
		}

		jsonName := parseJSONTag(jsonTag)
		goType := types.ExprString(field.Type)
		_, isPointer := field.Type.(*ast.StarExpr)

		parsed := parseBunFieldTag(bunTag)
		switch {
		case parsed.RelationKind != "":
			model.Relations = append(model.Relations, Relation{
				GoName:    fieldName,
				JSONName:  jsonName,
				GoType:    goType,
				Kind:      parsed.RelationKind,
				JoinPairs: parsed.JoinPairs,
			})
			continue
		case parsed.M2MTable != "":
			model.M2MRelations = append(model.M2MRelations, M2MRelation{
				GoName:       fieldName,
				JSONName:     jsonName,
				GoType:       goType,
				ThroughTable: parsed.M2MTable,
				JoinSpec:     parsed.M2MJoinSpec,
			})
			continue
		}

		if parsed.ColumnName == "" {
			continue
		}

		model.Fields = append(model.Fields, Field{
			GoName:     fieldName,
			ColumnName: parsed.ColumnName,
			JSONName:   jsonName,
			GoType:     goType,
			SQLType:    parsed.SQLType,
			IsPK:       parsed.IsPK,
			IsScanOnly: parsed.IsScanOnly,
			IsNotNull:  parsed.IsNotNull,
			IsNullZero: parsed.IsNullZero,
			IsArray:    parsed.IsArray,
			IsPointer:  isPointer,
		})
	}

	if len(model.Fields) == 0 {
		return nil, structName + " has bun.BaseModel but no parseable columns — skipping"
	}

	return model, ""
}

func isBunEmbed(bunTag string) bool {
	for part := range strings.SplitSeq(bunTag, ",") {
		if strings.TrimSpace(part) == "embed" {
			return true
		}
	}

	return false
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

type parsedFieldTag struct {
	ColumnName   string
	SQLType      string
	RelationKind RelationKind
	JoinPairs    []JoinPair
	M2MTable     string
	M2MJoinSpec  string
	IsPK         bool
	IsScanOnly   bool
	IsNotNull    bool
	IsNullZero   bool
	IsArray      bool
}

func splitTagParts(tag string) []string {
	var parts []string
	var depth int
	var inQuote bool
	start := 0

	for i := range len(tag) {
		switch tag[i] {
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		case '\'':
			inQuote = !inQuote
		case ',':
			if depth == 0 && !inQuote {
				parts = append(parts, tag[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, tag[start:])

	return parts
}

func parseBunFieldTag(bunTag string) parsedFieldTag {
	var parsed parsedFieldTag

	if bunTag == "" || bunTag == "-" {
		return parsed
	}

	parts := splitTagParts(bunTag)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "rel:"); ok {
			parsed.RelationKind = RelationKind(v)
		}
		if v, ok := strings.CutPrefix(part, "m2m:"); ok {
			parsed.M2MTable = v
		}
	}

	if parsed.RelationKind != "" || parsed.M2MTable != "" {
		for _, part := range parts {
			part = strings.TrimSpace(part)
			v, ok := strings.CutPrefix(part, "join:")
			if !ok {
				continue
			}
			if parsed.M2MTable != "" {
				parsed.M2MJoinSpec = v
				continue
			}
			local, remote, found := strings.Cut(v, "=")
			if !found {
				continue
			}
			parsed.JoinPairs = append(parsed.JoinPairs, JoinPair{Local: local, Remote: remote})
		}
		return parsed
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if v, ok := strings.CutPrefix(part, "type:"); ok {
			parsed.SQLType = v
			continue
		}
		if strings.Contains(part, ":") {
			continue
		}
		if bunKeywords[part] {
			continue
		}
		if parsed.ColumnName == "" {
			parsed.ColumnName = part
		}
	}

	for _, part := range parts {
		switch strings.TrimSpace(part) {
		case "pk":
			parsed.IsPK = true
		case "scanonly":
			parsed.IsScanOnly = true
		case "notnull":
			parsed.IsNotNull = true
		case "nullzero":
			parsed.IsNullZero = true
		case "array":
			parsed.IsArray = true
		}
	}

	return parsed
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
