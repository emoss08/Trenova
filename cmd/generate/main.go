package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

func parseStructFields(fileName, structName string) ([]*ast.Field, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, f := range node.Decls {
		gen, ok := f.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			return structType.Fields.List, nil
		}
	}
	return nil, nil
}

type FieldData struct {
	Name            string
	IncludeInUpdate bool
}
type ServiceData struct {
	EntityName      string
	EntityNameLower string
	Fields          []FieldData
}

func main() {
	fields, err := parseStructFields("ent/hazardousmaterialsegregation.go", "HazardousMaterialSegregation")
	if err != nil {
		log.Fatalf("Error parsing struct: %v", err)
	}

	var fieldData []FieldData
	for _, field := range fields {
		if len(field.Names) > 0 {
			fieldName := field.Names[0].Name
			// Exclude common fields like ID, CreatedAt, etc., and specific fields in updates.
			includeInUpdate := fieldName != "OrganizationID" && fieldName != "BusinessUnitID"
			if fieldName != "ID" && fieldName != "CreatedAt" && fieldName != "UpdatedAt" && fieldName != "Edges" && fieldName != "selectValues" {
				fieldData = append(fieldData, FieldData{Name: fieldName, IncludeInUpdate: includeInUpdate})
			}
		}
	}

	entityName := "HazardousMaterialSegregation"
	data := ServiceData{
		EntityName:      entityName,
		EntityNameLower: strings.ToLower(entityName),
		Fields:          fieldData,
	}

	tempText := `
// Package services provides the service layer for the {{.EntityName}} entity.

package services

import (
	"context"

    "github.com/emoss08/trenova/ent/{{.EntityNameLower}}"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
    "github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type {{.EntityName}}Ops struct {
	ctx    context.Context
	client *ent.Client
}

func New{{.EntityName}}Ops(ctx context.Context) *{{.EntityName}}Ops {
	return &{{.EntityName}}Ops{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// Get{{.EntityName}}s retrieves entities for an organization and business unit.
func (ops *{{.EntityName}}Ops) Get{{.EntityName}}s(limit, offset int, orgID, buID uuid.UUID) ([]*ent.{{.EntityName}}, int, error) {
	count, countErr := ops.client.{{.EntityName}}.Query().
		Where({{.EntityNameLower}}.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ops.ctx)
	
	if countErr != nil {
		return nil, 0, countErr
	}

	items, err := ops.client.{{.EntityName}}.Query().
		Limit(limit).
		Offset(offset).
		Where(
			{{.EntityNameLower}}.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ops.ctx)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

// Create{{.EntityName}} creates a new entity.
func (ops *{{.EntityName}}Ops) Create{{.EntityName}}(item *ent.{{.EntityName}}) (*ent.{{.EntityName}}, error) {
	return ops.client.{{.EntityName}}.Create().
		{{- range .Fields}}
		Set{{.Name}}(item.{{.Name}}).
		{{- end}}
		Save(ops.ctx)
}

// Update{{.EntityName}} updates a {{.EntityName}} entity.
func (ops *{{.EntityName}}Ops) Update{{.EntityName}}(item *ent.{{.EntityName}}) (*ent.{{.EntityName}}, error) {
	// Start building the update operation
	updateOp := ops.client.{{.EntityName}}.UpdateOneID(item.ID).
	{{- range .Fields}}
	{{- if .IncludeInUpdate}}
	Set{{.Name}}(item.{{.Name}}).
	{{- end}}
	{{- end}}

	updatedEntity, err := updateOp.Save(ops.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
`

	tmpl, err := template.New("service").Parse(tempText)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	file, err := os.Create(strings.ToLower(entityName) + "_service.go")
	if err != nil {
		log.Fatalf("Error creating service file: %v", err)
	}
	defer file.Close()

	err = tmpl.Execute(file, data)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	log.Println("Service file generated successfully")
}
