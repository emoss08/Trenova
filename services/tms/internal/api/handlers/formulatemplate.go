package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	formulatemplateservice "github.com/emoss08/trenova/internal/core/services/formulatemplate"
	"github.com/emoss08/trenova/pkg/formula/expression"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type FormulaTemplateHandlerParams struct {
	fx.In

	Service          *formulatemplateservice.Service
	PM               *middleware.PermissionMiddleware
	ErrorHandler     *helpers.ErrorHandler
	VariableRegistry *variables.Registry
}

type FormulaTemplateHandler struct {
	service          *formulatemplateservice.Service
	errorHandler     *helpers.ErrorHandler
	pm               *middleware.PermissionMiddleware
	variableRegistry *variables.Registry
}

func NewFormulaTemplateHandler(p FormulaTemplateHandlerParams) *FormulaTemplateHandler {
	return &FormulaTemplateHandler{
		service:          p.Service,
		errorHandler:     p.ErrorHandler,
		pm:               p.PM,
		variableRegistry: p.VariableRegistry,
	}
}

func (h *FormulaTemplateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/formula-templates/")

	// CRUD operations
	api.GET("", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "update"), h.update)

	// LSP and editor support endpoints
	api.GET("variables", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.listVariables)
	api.GET("functions", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.listFunctions)
	api.POST("validate", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.validateExpression)
	api.POST("test", h.pm.RequirePermission(permission.ResourceFormulaTemplate, "read"), h.testFormula)
}

func (h *FormulaTemplateHandler) list(c *gin.Context) {
	pagination.Handle[*formulatemplate.FormulaTemplate](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			return h.service.List(c.Request.Context(), &repositories.ListFormulaTemplateRequest{
				Filter: opts,
			})
		})
}

func (h *FormulaTemplateHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetFormulaTemplateByIDRequest{
			ID:    id,
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FormulaTemplateHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(formulatemplate.FormulaTemplate)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *FormulaTemplateHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(formulatemplate.FormulaTemplate)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// VariableInfo represents variable metadata for LSP/autocomplete
type VariableInfo struct {
	Name        string                `json:"name"`
	Type        formulatypes.ValueType `json:"type"`
	Description string                `json:"description"`
	Category    string                `json:"category"`
	Example     string                `json:"example,omitempty"`
}

// listVariables returns all available variables for LSP/autocomplete
func (h *FormulaTemplateHandler) listVariables(c *gin.Context) {
	category := c.Query("category")

	var vars []variables.Variable
	if category != "" {
		vars = h.variableRegistry.GetByCategory(category)
	} else {
		vars = h.variableRegistry.List()
	}

	result := make([]VariableInfo, 0, len(vars))
	for _, v := range vars {
		result = append(result, VariableInfo{
			Name:        v.Name(),
			Type:        v.Type(),
			Description: v.Description(),
			Category:    v.Category(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"variables":  result,
		"categories": h.variableRegistry.Categories(),
		"count":      len(result),
	})
}

// FunctionInfo represents function metadata for LSP/autocomplete
type FunctionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MinArgs     int    `json:"minArgs"`
	MaxArgs     int    `json:"maxArgs"` // -1 means unlimited
	Signature   string `json:"signature"`
	Example     string `json:"example"`
	Category    string `json:"category"`
}

// listFunctions returns all available functions for LSP/autocomplete
func (h *FormulaTemplateHandler) listFunctions(c *gin.Context) {
	functions := []FunctionInfo{
		// Math functions
		{Name: "abs", Description: "Returns the absolute value of a number", MinArgs: 1, MaxArgs: 1, Signature: "abs(number)", Example: "abs(-5) → 5", Category: "math"},
		{Name: "min", Description: "Returns the smallest of the given numbers", MinArgs: 1, MaxArgs: -1, Signature: "min(number, ...)", Example: "min(5, 3, 8) → 3", Category: "math"},
		{Name: "max", Description: "Returns the largest of the given numbers", MinArgs: 1, MaxArgs: -1, Signature: "max(number, ...)", Example: "max(5, 3, 8) → 8", Category: "math"},
		{Name: "round", Description: "Rounds a number to the nearest integer or specified decimals", MinArgs: 1, MaxArgs: 2, Signature: "round(number, decimals?)", Example: "round(3.7) → 4, round(3.14159, 2) → 3.14", Category: "math"},
		{Name: "floor", Description: "Rounds a number down to the nearest integer", MinArgs: 1, MaxArgs: 1, Signature: "floor(number)", Example: "floor(3.7) → 3", Category: "math"},
		{Name: "ceil", Description: "Rounds a number up to the nearest integer", MinArgs: 1, MaxArgs: 1, Signature: "ceil(number)", Example: "ceil(3.2) → 4", Category: "math"},
		{Name: "sqrt", Description: "Returns the square root of a number", MinArgs: 1, MaxArgs: 1, Signature: "sqrt(number)", Example: "sqrt(16) → 4", Category: "math"},
		{Name: "pow", Description: "Returns base raised to the power of exponent", MinArgs: 2, MaxArgs: 2, Signature: "pow(base, exponent)", Example: "pow(2, 3) → 8", Category: "math"},
		{Name: "log", Description: "Returns the natural logarithm of a number", MinArgs: 1, MaxArgs: 1, Signature: "log(number)", Example: "log(2.718) → 1", Category: "math"},
		{Name: "exp", Description: "Returns e raised to the power of a number", MinArgs: 1, MaxArgs: 1, Signature: "exp(number)", Example: "exp(1) → 2.718", Category: "math"},
		{Name: "sin", Description: "Returns the sine of a number (radians)", MinArgs: 1, MaxArgs: 1, Signature: "sin(radians)", Example: "sin(0) → 0", Category: "math"},
		{Name: "cos", Description: "Returns the cosine of a number (radians)", MinArgs: 1, MaxArgs: 1, Signature: "cos(radians)", Example: "cos(0) → 1", Category: "math"},
		{Name: "tan", Description: "Returns the tangent of a number (radians)", MinArgs: 1, MaxArgs: 1, Signature: "tan(radians)", Example: "tan(0) → 0", Category: "math"},

		// Type conversion
		{Name: "number", Description: "Converts a value to a number", MinArgs: 1, MaxArgs: 1, Signature: "number(value)", Example: "number(\"42\") → 42", Category: "conversion"},
		{Name: "string", Description: "Converts a value to a string", MinArgs: 1, MaxArgs: 1, Signature: "string(value)", Example: "string(42) → \"42\"", Category: "conversion"},
		{Name: "bool", Description: "Converts a value to a boolean", MinArgs: 1, MaxArgs: 1, Signature: "bool(value)", Example: "bool(1) → true", Category: "conversion"},

		// Array functions
		{Name: "len", Description: "Returns the length of an array or string", MinArgs: 1, MaxArgs: 1, Signature: "len(array|string)", Example: "len([1, 2, 3]) → 3", Category: "array"},
		{Name: "sum", Description: "Returns the sum of all numbers in an array", MinArgs: 1, MaxArgs: 1, Signature: "sum(array)", Example: "sum([1, 2, 3]) → 6", Category: "array"},
		{Name: "avg", Description: "Returns the average of all numbers in an array", MinArgs: 1, MaxArgs: 1, Signature: "avg(array)", Example: "avg([2, 4, 6]) → 4", Category: "array"},
		{Name: "slice", Description: "Extracts a portion of an array", MinArgs: 2, MaxArgs: 3, Signature: "slice(array, start, end?)", Example: "slice([1, 2, 3, 4], 1, 3) → [2, 3]", Category: "array"},
		{Name: "concat", Description: "Concatenates multiple arrays", MinArgs: 2, MaxArgs: -1, Signature: "concat(array, ...)", Example: "concat([1, 2], [3, 4]) → [1, 2, 3, 4]", Category: "array"},
		{Name: "contains", Description: "Checks if an array contains a value", MinArgs: 2, MaxArgs: 2, Signature: "contains(array, value)", Example: "contains([1, 2, 3], 2) → true", Category: "array"},
		{Name: "indexOf", Description: "Returns the index of a value in an array (-1 if not found)", MinArgs: 2, MaxArgs: 2, Signature: "indexOf(array, value)", Example: "indexOf([1, 2, 3], 2) → 1", Category: "array"},

		// Conditional
		{Name: "if", Description: "Returns one of two values based on a condition", MinArgs: 3, MaxArgs: 3, Signature: "if(condition, trueValue, falseValue)", Example: "if(weight > 1000, 100, 50)", Category: "conditional"},
		{Name: "coalesce", Description: "Returns the first non-null value", MinArgs: 1, MaxArgs: -1, Signature: "coalesce(value, ...)", Example: "coalesce(null, null, 5) → 5", Category: "conditional"},
	}

	category := c.Query("category")
	if category != "" {
		filtered := make([]FunctionInfo, 0)
		for _, f := range functions {
			if f.Category == category {
				filtered = append(filtered, f)
			}
		}
		functions = filtered
	}

	categories := []string{"math", "conversion", "array", "conditional"}

	c.JSON(http.StatusOK, gin.H{
		"functions":  functions,
		"categories": categories,
		"count":      len(functions),
	})
}

// ValidateExpressionRequest represents a request to validate an expression
type ValidateExpressionRequest struct {
	Expression string `json:"expression" binding:"required"`
}

// ValidateExpressionResponse represents the validation result
type ValidateExpressionResponse struct {
	Valid   bool     `json:"valid"`
	Error   string   `json:"error,omitempty"`
	Line    int      `json:"line,omitempty"`
	Column  int      `json:"column,omitempty"`
	Message string   `json:"message,omitempty"`
	Tokens  []string `json:"tokens,omitempty"`
}

// validateExpression provides real-time syntax validation for formulas
func (h *FormulaTemplateHandler) validateExpression(c *gin.Context) {
	var req ValidateExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	// Try to parse the expression
	_, err := expression.Parse(req.Expression)
	if err != nil {
		c.JSON(http.StatusOK, ValidateExpressionResponse{
			Valid:   false,
			Error:   err.Error(),
			Message: "Expression parsing failed",
		})
		return
	}

	c.JSON(http.StatusOK, ValidateExpressionResponse{
		Valid:   true,
		Message: "Expression is valid",
	})
}

// TestFormulaRequest represents a request to test a formula with sample data
type TestFormulaRequest struct {
	Expression string         `json:"expression" binding:"required"`
	Variables  map[string]any `json:"variables"`
}

// TestFormulaResponse represents the test result
type TestFormulaResponse struct {
	Success       bool           `json:"success"`
	Result        any            `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
	UsedVariables []string       `json:"usedVariables,omitempty"`
	Steps         []string       `json:"steps,omitempty"`
	ResultType    string         `json:"resultType,omitempty"`
}

// testFormula allows testing a formula with sample data
func (h *FormulaTemplateHandler) testFormula(c *gin.Context) {
	var req TestFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	// Parse the expression
	parsedExpr, err := expression.Parse(req.Expression)
	if err != nil {
		c.JSON(http.StatusOK, TestFormulaResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Create a simple context with the provided variables
	ctx := expression.NewEvaluationContext(req.Variables, expression.DefaultFunctionRegistry())

	// Evaluate the parsed expression
	result, err := parsedExpr.Evaluate(ctx)
	if err != nil {
		c.JSON(http.StatusOK, TestFormulaResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Determine result type
	resultType := "unknown"
	if result != nil {
		switch result.(type) {
		case float64, int, int64, int32:
			resultType = "number"
		case string:
			resultType = "string"
		case bool:
			resultType = "boolean"
		case []any:
			resultType = "array"
		case map[string]any:
			resultType = "object"
		}
	}

	// Collect used variables
	usedVars := make([]string, 0)
	for varName := range req.Variables {
		usedVars = append(usedVars, varName)
	}

	c.JSON(http.StatusOK, TestFormulaResponse{
		Success:       true,
		Result:        result,
		UsedVariables: usedVars,
		ResultType:    resultType,
	})
}
