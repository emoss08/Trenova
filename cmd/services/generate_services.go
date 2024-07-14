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
	"regexp"
	"strings"
	"sync"
)

func main() {
	// Define command-line flags
	projectRoot := flag.String("root", "", "Project root directory")
	modelsDir := flag.String("models", "", "Models directory")
	servicesDir := flag.String("services", "", "Services directory")
	flag.Parse()

	// Validate input
	if *projectRoot == "" || *modelsDir == "" || *servicesDir == "" {
		log.Println("Error: All parameters (root, models, services) must be provided")
		flag.Usage()
		os.Exit(1)
	}

	// Ensure the services directory exists
	err := os.MkdirAll(*servicesDir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating services directory: %v\n", err)
		return
	}

	// Process model files concurrently
	err = processModelFiles(*modelsDir, *servicesDir)
	if err != nil {
		log.Printf("Error processing model files: %v\n", err)
	}
}

// processModelFiles walks through the models directory and processes each Go file
func processModelFiles(modelsDir, servicesDir string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	err := filepath.Walk(modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				if err := generateServiceForModelFile(path, servicesDir); err != nil {
					errChan <- fmt.Errorf("error processing %s: %w", path, err)
				}
			}(path)
		}
		return nil
	})

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		log.Printf("%v\n", err)
	}

	return err
}

func generateServiceForModelFile(filename, servicesDir string) error {
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
			if err := generateServiceCode(modelName, structType, servicesDir, queryField); err != nil {
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

// generateServiceCode creates a service file for a given model
func generateServiceCode(modelName string, structType *ast.StructType, servicesDir string, queryField string) error {
	alias := getAliasTag(structType)
	if alias == "" {
		alias = strings.ToLower(modelName[:1])
	}

	queryFieldSnakeCase := toSnakeCase(queryField)

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

// %sService handles business logic for %s
type %sService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// New%sService creates a new instance of %sService
func New%sService(s *server.Server) *%sService {
	return &%sService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying %s
type %sQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s %sService) filterQuery(q *bun.SelectQuery, f *%sQueryFilter) *bun.SelectQuery {
	q = q.Where("%s.organization_id = ?", f.OrganizationID).
		Where("%s.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("%s.%s = ? OR %s.%s ILIKE ?", f.Query, "%%"+strings.ToLower(f.Query)+"%%")
	}

	q = q.OrderExpr("CASE WHEN %s.%s = ? THEN 0 ELSE 1 END", f.Query).
		Order("%s.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all %s based on the provided filter
func (s %sService) GetAll(ctx context.Context, filter *%sQueryFilter) ([]*models.%s, int, error) {
	var entities []*models.%s
	
	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch %s")
		return nil, 0, fmt.Errorf("failed to fetch %s: %%w", err)
	}

	return entities, count, nil
}

// Get retrieves a single %s by ID
func (s *%sService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.%s, error) {
	entity := new(models.%s)
	err := s.db.NewSelect().
		Model(entity).
		Where("%s.organization_id = ?", orgID).
		Where("%s.business_unit_id = ?", buID).
		Where("%s.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch %s")
		return nil, fmt.Errorf("failed to fetch %s: %%w", err)
	}

	return entity, nil
}

// Create creates a new %s
func (s %sService) Create(ctx context.Context, entity *models.%s) (*models.%s, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create %s")
		return nil, fmt.Errorf("failed to create %s: %%w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing %s
func (s %sService) UpdateOne(ctx context.Context, entity *models.%s) (*models.%s, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update %s")
		return nil, fmt.Errorf("failed to update %s: %%w", err)
	}

	return entity, nil
}
`

	code := fmt.Sprintf(serviceTemplate,
		modelName, modelName, modelName, modelName, modelName, modelName, modelName, modelName,
		modelName, modelName, modelName, modelName, alias, alias, alias, queryFieldSnakeCase, alias, queryFieldSnakeCase,
		alias, queryFieldSnakeCase, alias,
		modelName, modelName, modelName, modelName, modelName,
		modelName, modelName,
		modelName, modelName, modelName, modelName, alias, alias, alias, modelName, modelName,
		modelName, modelName, modelName, modelName, modelName, modelName,
		modelName, modelName, modelName, modelName, modelName, modelName)

	formattedCode, err := format.Source([]byte(code))
	if err != nil {
		return fmt.Errorf("error formatting code for %s: %w", modelName, err)
	}

	outputPath := filepath.Join(servicesDir, fmt.Sprintf("%s_service.go", strings.ToLower(modelName)))
	if err := os.WriteFile(outputPath, formattedCode, 0o644); err != nil {
		return fmt.Errorf("error writing service file for %s: %w", modelName, err)
	}

	log.Printf("Service code generated successfully for %s\n", modelName)
	return nil
}

// getAliasTag extracts the alias tag from the struct
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

// getRelations builds the relation string for the query
func getRelations(structType *ast.StructType) string {
	relations := make(map[string]struct{})
	for _, field := range structType.Fields.List {
		if field.Tag != nil {
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			bunTag := tag.Get("bun")
			if strings.Contains(bunTag, "rel:") {
				relationName := field.Names[0].Name
				relations[relationName] = struct{}{}
			}
		}
	}

	var relationsStr strings.Builder
	for relation := range relations {
		relationsStr.WriteString(fmt.Sprintf("Relation(\"%s\").\n\t\t", relation))
	}
	return relationsStr.String()
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(str string) string {
	snake := regexp.MustCompile("([a-z0-9])([A-Z])").ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}
