package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ValidateQueryRequest struct {
	Query string `json:"query" binding:"required"`
}

type ValidateResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type TestVariableRequest struct {
	Query      string         `json:"query"      binding:"required"`
	TestParams map[string]any `json:"testParams"`
}

type TestVariableResponse struct {
	Result string `json:"result"`
}

type ValidateFormatSQLRequest struct {
	FormatSQL string `json:"formatSQL" binding:"required"`
}

type TestFormatRequest struct {
	FormatSQL string `json:"formatSQL" binding:"required"`
	TestValue string `json:"testValue" binding:"required"`
}

type TestFormatResponse struct {
	Result string `json:"result"`
}

type VariableService interface {
	List(
		ctx context.Context,
		req *repositories.ListVariableRequest,
	) (*pagination.ListResult[*variable.Variable], error)
	Get(ctx context.Context, req repositories.GetVariableByIDRequest) (*variable.Variable, error)
	Create(
		ctx context.Context,
		entity *variable.Variable,
		userID pulid.ID,
	) (*variable.Variable, error)
	Update(
		ctx context.Context,
		entity *variable.Variable,
		userID pulid.ID,
	) (*variable.Variable, error)
	Delete(ctx context.Context, req repositories.GetVariableByIDRequest) error
	ValidateQuery(req *ValidateQueryRequest) (*ValidateResponse, error)
	TestVariable(ctx context.Context, req *TestVariableRequest) (*TestVariableResponse, error)
	GetVariablesByContext(
		ctx context.Context,
		req repositories.GetVariablesByContextRequest,
	) ([]*variable.Variable, error)
	ListFormats(
		ctx context.Context,
		req *repositories.ListVariableFormatRequest,
	) (*pagination.ListResult[*variable.VariableFormat], error)
	GetFormat(
		ctx context.Context,
		req repositories.GetVariableFormatByIDRequest,
	) (*variable.VariableFormat, error)
	CreateFormat(
		ctx context.Context,
		entity *variable.VariableFormat,
		userID pulid.ID,
	) (*variable.VariableFormat, error)
	UpdateFormat(
		ctx context.Context,
		entity *variable.VariableFormat,
		userID pulid.ID,
	) (*variable.VariableFormat, error)
	DeleteFormat(ctx context.Context, req repositories.GetVariableFormatByIDRequest) error
	ValidateFormatSQL(req *ValidateFormatSQLRequest) (*ValidateResponse, error)
	TestFormat(ctx context.Context, req *TestFormatRequest) (*TestFormatResponse, error)
}

type VariableResolverService interface {
	ProcessTemplate(
		ctx context.Context,
		template string,
		context variable.Context,
		contextID pulid.ID,
		orgID pulid.ID,
		buID pulid.ID,
	) (string, error)
	ResolveVariable(
		ctx context.Context,
		v *variable.Variable,
		params map[string]any,
	) (string, error)
	ResolveVariables(
		ctx context.Context,
		variables []*variable.Variable,
		params map[string]any,
	) (map[string]string, error)
}
