package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
)

func main() {
	// Define command-line flags
	root := flag.String("root", "", "Project root directory")
	modelsDir := flag.String("models", "", "Models directory")
	handlersDir := flag.String("handlers", "", "Handlers directory")
	servicesPackage := flag.String("services", "services", "Services package name")
	flag.Parse()

	// Validate input
	if *root == "" || *modelsDir == "" || *handlersDir == "" {
		fmt.Println("Usage: go run generate_handlers.go -root <project_root> -models <models_dir> -handlers <handlers_dir> [-services <services_package>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Resolve full paths
	modelsPath := filepath.Join(*root, *modelsDir)
	handlersPath := filepath.Join(*root, *handlersDir)

	// Ensure the handlers directory exists
	err := os.MkdirAll(handlersPath, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating handlers directory: %v\n", err)
		return
	}

	// Process model files
	err = filepath.Walk(modelsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			err := generateHandlerForModel(path, handlersPath, *servicesPackage)
			if err != nil {
				fmt.Printf("Error generating handler for %s: %v\n", path, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error walking through model files: %v\n", err)
	}
}

func generateHandlerForModel(modelPath, handlersDir, servicesPackage string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, modelPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file %s: %w", modelPath, err)
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
			err := generateHandlerCode(modelName, handlersDir, servicesPackage)
			if err != nil {
				return fmt.Errorf("error generating handler code for %s: %w", modelName, err)
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

func generateHandlerCode(modelName, handlersDir, servicesPackage string) error {
	handlerTemplate := `
package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/{{.ServicesPackage}}"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type {{.ModelName}}Handler struct {
	logger            *zerolog.Logger
	service           *services.{{.ModelName}}Service
	permissionService *services.PermissionService
}

func New{{.ModelName}}Handler(s *server.Server) *{{.ModelName}}Handler {
	return &{{.ModelName}}Handler{
		logger:            s.Logger,
		service:           services.New{{.ModelName}}Service(s),
		permissionService: services.NewPermissionService(s),
	}
}

func (h {{.ModelName}}Handler) RegisterRoutes(r fiber.Router) {
	api := r.Group("/{{.RoutePrefix}}s")
	api.Get("/", h.Get())
	api.Get("/:{{.LowerModelName}}ID", h.GetByID())
	api.Post("/", h.Create())
	api.Put("/:{{.LowerModelName}}ID", h.Update())
}

func (h {{.ModelName}}Handler) Get() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("{{.ModelName}}Handler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		offset, limit, err := utils.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ProblemDetail{
				Type:     "invalid",
				Title:    "Invalid Request",
				Status:   fiber.StatusBadRequest,
				Detail:   err.Error(),
				Instance: fmt.Sprintf("%s/probs/validation-error", c.BaseURL()),
				InvalidParams: []types.InvalidParam{
					{
						Name:   "limit",
						Reason: "Limit must be a positive integer",
					},
					{
						Name:   "offset",
						Reason: "Offset must be a positive integer",
					},
				},
			})
		}

		if err = h.permissionService.CheckUserPermission(c, models.Permission{{.ModelName}}View.String()); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		filter := &services.{{.ModelName}}QueryFilter{
			Query:          c.Query("search", ""),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Limit:          limit,
			Offset:         offset,
		}


		entities, cnt, err := h.service.GetAll(c.UserContext(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get {{.PluralModelName}}",
			})
		}

		nextURL := utils.GetNextPageURL(c, limit, offset, cnt)
		prevURL := utils.GetPrevPageURL(c, limit, offset)

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse[[]*models.{{.ModelName}}]{
			Results: entities,
			Count:   cnt,
			Next:    nextURL,
			Prev:    prevURL,
		})
	}
}

func (h {{.ModelName}}Handler) Create() fiber.Handler {
	return func(c *fiber.Ctx) error {
		createdEntity := new(models.{{.ModelName}})

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, models.Permission{{.ModelName}}Create.String()); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		createdEntity.BusinessUnitID = buID
		createdEntity.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(c, createdEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		entity, err := h.service.Create(c.UserContext(), createdEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

func (h {{.ModelName}}Handler) GetByID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		{{.LowerModelName}}ID := c.Params("{{.LowerModelName}}ID")
		if {{.LowerModelName}}ID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "{{.ModelName}} ID is required",
			})
		}

		orgID, ok := c.Locals(utils.CTXOrganizationID).(uuid.UUID)
		buID, orgOK := c.Locals(utils.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !orgOK {
			h.logger.Error().Msg("{{.ModelName}}Handler: Organization & Business Unit ID not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Error{
				Code:    fiber.StatusUnauthorized,
				Message: "Organization & Business Unit ID not found in context",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, models.Permission{{.ModelName}}View.String()); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		entity, err := h.service.Get(c.UserContext(), uuid.MustParse({{.LowerModelName}}ID), orgID, buID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Error{
				Code:    fiber.StatusInternalServerError,
				Message: "Failed to get {{.ModelName}}",
			})
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}

func (h {{.ModelName}}Handler) Update() fiber.Handler {
	return func(c *fiber.Ctx) error {
		{{.LowerModelName}}ID := c.Params("{{.LowerModelName}}ID")
		if {{.LowerModelName}}ID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Error{
				Code:    fiber.StatusBadRequest,
				Message: "{{.ModelName}} ID is required",
			})
		}

		if err := h.permissionService.CheckUserPermission(c, models.Permission{{.ModelName}}Edit.String()); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Error{
				Code:    fiber.StatusForbidden,
				Message: "You do not have permission to perform this action.",
			})
		}

		updatedEntity := new(models.{{.ModelName}})

		if err := utils.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		updatedEntity.ID = uuid.MustParse({{.LowerModelName}}ID)

		entity, err := h.service.UpdateOne(c.UserContext(), updatedEntity)
		if err != nil {
			resp := utils.CreateServiceError(c, err)
			return c.Status(fiber.StatusInternalServerError).JSON(resp)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
`

	tmpl, err := template.New("handler").Parse(handlerTemplate)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	data := struct {
		ModelName       string
		LowerModelName  string
		PluralModelName string
		RoutePrefix     string
		ServicesPackage string
	}{
		ModelName:       modelName,
		LowerModelName:  strings.ToLower(modelName),
		PluralModelName: pluralize(modelName),
		RoutePrefix:     lo.KebabCase(modelName),
		ServicesPackage: servicesPackage,
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

	outputPath := filepath.Join(handlersDir, fmt.Sprintf("%s_handler.go", lo.SnakeCase(modelName)))
	err = os.WriteFile(outputPath, formattedCode, 0o644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("Generated handler for %s\n", modelName)
	return nil
}

func pluralize(str string) string {
	// This is a very simple pluralization.
	// For more complex rules, consider using a dedicated pluralization library.
	if strings.HasSuffix(str, "y") {
		return str[:len(str)-1] + "ies"
	}
	return str + "s"
}
