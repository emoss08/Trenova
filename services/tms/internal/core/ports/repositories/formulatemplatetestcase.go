package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTestCasesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TemplateID pulid.ID              `json:"templateId"`
}

type GetTestCaseByIDRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TestCaseID pulid.ID              `json:"testCaseId"`
	TemplateID pulid.ID              `json:"templateId"`
}

type FormulaTemplateTestCaseRepository interface {
	Create(
		ctx context.Context,
		entity *formulatemplate.TestCase,
	) (*formulatemplate.TestCase, error)
	Update(
		ctx context.Context,
		entity *formulatemplate.TestCase,
	) (*formulatemplate.TestCase, error)
	Delete(ctx context.Context, req GetTestCaseByIDRequest) error
	GetByID(
		ctx context.Context,
		req GetTestCaseByIDRequest,
	) (*formulatemplate.TestCase, error)
	ListByTemplate(
		ctx context.Context,
		req ListTestCasesRequest,
	) ([]*formulatemplate.TestCase, error)
}
