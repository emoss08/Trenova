package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrCaseChecklistTemplateStale is a save of a template someone else saved
// first, or a second template for the same customer and kind.
var ErrCaseChecklistTemplateStale = errors.New("the checklist was changed by someone else")

type ListCaseChecklistTemplatesRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       deskcase.ChecklistKind
}

type GetCaseChecklistTemplateForRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       deskcase.ChecklistKind
	// CustomerID, when set, reads that customer's template ahead of the
	// organization's.
	CustomerID pulid.ID
}

type GetCaseChecklistTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListCaseChecklistTicksRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType string
	SubjectID   pulid.ID
}

type UntickCaseChecklistItemRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType string
	SubjectID   pulid.ID
	ItemKey     deskcase.ItemKey
}

type CountCaseChecklistDocumentTypesRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

// CaseChecklistRepository keeps how organizations and their customers want
// case checklists laid out, and people's ticks on the steps they added.
type CaseChecklistRepository interface {
	// ListTemplates is the organization's template of a kind, when it saved
	// one, and every customer's, with the customer's name.
	ListTemplates(
		ctx context.Context,
		req *ListCaseChecklistTemplatesRequest,
	) ([]*deskcase.ChecklistTemplate, error)
	// TemplateFor is the template that applies: the customer's, else the
	// organization's, else nil for the default. One query.
	TemplateFor(
		ctx context.Context,
		req *GetCaseChecklistTemplateForRequest,
	) (*deskcase.ChecklistTemplate, error)
	GetTemplate(
		ctx context.Context,
		req *GetCaseChecklistTemplateRequest,
	) (*deskcase.ChecklistTemplate, error)
	// SaveTemplate inserts a template without an id and updates one with
	// an id at the version it was read at. Either is
	// ErrCaseChecklistTemplateStale when someone saved first.
	SaveTemplate(
		ctx context.Context,
		entity *deskcase.ChecklistTemplate,
	) (*deskcase.ChecklistTemplate, error)
	DeleteTemplate(ctx context.Context, req *GetCaseChecklistTemplateRequest) error
	// CountDocumentTypes counts how many of the ids are the organization's
	// document types.
	CountDocumentTypes(ctx context.Context, req *CountCaseChecklistDocumentTypesRequest) (int, error)
	ListTicks(
		ctx context.Context,
		req *ListCaseChecklistTicksRequest,
	) ([]*deskcase.ChecklistTick, error)
	// Tick records a tick; ticking a step already ticked keeps the first.
	Tick(ctx context.Context, entity *deskcase.ChecklistTick) error
	Untick(ctx context.Context, req *UntickCaseChecklistItemRequest) error
}
