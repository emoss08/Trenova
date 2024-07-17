// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/samber/lo"
)

type Config struct {
	ProjectRoot     string
	ModelsDir       string
	ServicesDir     string
	ServicesPackage string
}

type TemplateData struct {
	ModelName           string
	LowerModelName      string
	PluralModelName     string
	QueryField          string
	QueryFieldSnakeCase string
	Alias               string
	ServicesPackage     string
}

func main() {
	config := parseFlags()
	if err := run(config); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() Config {
	config := Config{}
	flag.StringVar(&config.ProjectRoot, "root", "", "Project root directory")
	flag.StringVar(&config.ModelsDir, "models", "", "Models directory")
	flag.StringVar(&config.ServicesDir, "services", "", "Services directory")
	flag.StringVar(&config.ServicesPackage, "servicesPackage", "services", "Services package name")
	flag.Parse()

	if config.ProjectRoot == "" || config.ModelsDir == "" || config.ServicesDir == "" {
		log.Println("Error: All parameters (root, models, services) must be provided")
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func run(config Config) error {
	if err := os.MkdirAll(config.ServicesDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating services directory: %w", err)
	}

	return processModelFiles(config)
}

func processModelFiles(config Config) error {
	modelsPath := filepath.Join(config.ProjectRoot, config.ModelsDir)
	return filepath.Walk(modelsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if err := generateServiceForModelFile(path, config); err != nil {
				log.Printf("Error processing %s: %v", path, err)
			}
		}
		return nil
	})
}

func generateServiceForModelFile(filename string, config Config) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file %s: %w", filename, err)
	}

	for _, decl := range node.Decls {
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

			if !containsBaseModel(structType) {
				continue
			}

			modelName := typeSpec.Name.Name
			queryField := getQueryField(structType)
			if queryField == "" {
				continue
			}
			if err := generateServiceCode(modelName, structType, queryField, config); err != nil {
				return fmt.Errorf("error generating service code for %s: %w", modelName, err)
			}
		}
	}

	return nil
}

func containsBaseModel(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		if selectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
			if selectorExpr.Sel.Name == "BaseModel" {
				return true
			}
		}
	}
	return false
}

func getQueryField(structType *ast.StructType) string {
	for _, field := range structType.Fields.List {
		if field.Tag != nil {
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			if tag.Get("queryField") == "true" {
				return field.Names[0].Name
			}
		}
	}
	return ""
}

func generateServiceCode(modelName string, structType *ast.StructType, queryField string, config Config) error {
	alias := getAliasTag(structType)
	if alias == "" {
		alias = strings.ToLower(modelName[:1])
	}

	data := TemplateData{
		ModelName:           modelName,
		LowerModelName:      strings.ToLower(modelName),
		PluralModelName:     pluralize(modelName),
		QueryField:          queryField,
		QueryFieldSnakeCase: lo.SnakeCase(queryField),
		Alias:               alias,
		ServicesPackage:     config.ServicesPackage,
	}

	serviceTemplate := `
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// {{.ModelName}}Service handles business logic for {{.ModelName}}
type {{.ModelName}}Service struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// New{{.ModelName}}Service creates a new instance of {{.ModelName}}Service
func New{{.ModelName}}Service(s *server.Server) *{{.ModelName}}Service {
	return &{{.ModelName}}Service{
		db:     s.DB,
		logger: s.Logger,
	}
}

// {{.ModelName}}QueryFilter defines the filter parameters for querying {{.ModelName}}
type {{.ModelName}}QueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s {{.ModelName}}Service) filterQuery(q *bun.SelectQuery, f *{{.ModelName}}QueryFilter) *bun.SelectQuery {
	q = q.Where("{{.Alias}}.organization_id = ?", f.OrganizationID).
		Where("{{.Alias}}.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("{{.Alias}}.{{.QueryField}} = ? OR {{.Alias}}.{{.QueryField}} ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN {{.Alias}}.{{.QueryField}} = ? THEN 0 ELSE 1 END", f.Query).
		Order("{{.Alias}}.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all {{.PluralModelName}} based on the provided filter
func (s {{.ModelName}}Service) GetAll(ctx context.Context, filter *{{.ModelName}}QueryFilter) ([]*models.{{.ModelName}}, int, error) {
	var entities []*models.{{.ModelName}}
	
	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch {{.PluralModelName}}")
		return nil, 0, fmt.Errorf("failed to fetch {{.PluralModelName}}: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single {{.ModelName}} by ID
func (s *{{.ModelName}}Service) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.{{.ModelName}}, error) {
	entity := new(models.{{.ModelName}})
	err := s.db.NewSelect().
		Model(entity).
		Where("{{.Alias}}.organization_id = ?", orgID).
		Where("{{.Alias}}.business_unit_id = ?", buID).
		Where("{{.Alias}}.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch {{.ModelName}}")
		return nil, fmt.Errorf("failed to fetch {{.ModelName}}: %w", err)
	}

	return entity, nil
}

// Create creates a new {{.ModelName}}
func (s {{.ModelName}}Service) Create(ctx context.Context, entity *models.{{.ModelName}}) (*models.{{.ModelName}}, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create {{.ModelName}}")
		return nil, fmt.Errorf("failed to create {{.ModelName}}: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing {{.ModelName}}
func (s {{.ModelName}}Service) UpdateOne(ctx context.Context, entity *models.{{.ModelName}}) (*models.{{.ModelName}}, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update {{.ModelName}}")
		return nil, fmt.Errorf("failed to update {{.ModelName}}: %w", err)
	}

	return entity, nil
}
`

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	formattedCode, err := format.Source([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("error formatting code: %w", err)
	}

	outputPath := filepath.Join(config.ServicesDir, fmt.Sprintf("%s_service.go", lo.SnakeCase(modelName)))
	if err := os.WriteFile(outputPath, formattedCode, 0o644); err != nil {
		return fmt.Errorf("error writing service file for %s: %w", modelName, err)
	}

	log.Printf("Service code generated successfully for %s", modelName)
	return nil
}

func getAliasTag(structType *ast.StructType) string {
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		bunTag := tag.Get("bun")
		tags := strings.Split(bunTag, ",")
		for _, tag := range tags {
			if strings.HasPrefix(tag, "alias:") {
				return strings.TrimPrefix(tag, "alias:")
			}
		}
	}
	return ""
}

func pluralize(str string) string {
	if strings.HasSuffix(str, "y") {
		return str[:len(str)-1] + "ies"
	}
	return str + "s"
}
