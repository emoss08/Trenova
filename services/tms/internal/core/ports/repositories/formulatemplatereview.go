package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTemplateReviewsRequest struct {
	TemplateID pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
}

type ListStaleSubmissionsRequest struct {
	// SubmittedBefore is the unix time a submission must predate to count as
	// stale. Templates in review since then, across every tenant, are returned.
	SubmittedBefore int64
	Limit           int
}

// FormulaTemplateReviewRepository stores a template's review history and
// answers the review-cycle questions the sweep and the studio ask about it.
type FormulaTemplateReviewRepository interface {
	Create(ctx context.Context, entity *formulatemplate.Review) (*formulatemplate.Review, error)
	ListByTemplate(
		ctx context.Context,
		req *ListTemplateReviewsRequest,
	) ([]*formulatemplate.Review, error)
	// Latest returns the newest entry for the template, or nil when the
	// template has never been submitted.
	Latest(
		ctx context.Context,
		templateID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*formulatemplate.Review, error)
	// ListStaleSubmissions lists templates still in review whose submission
	// predates the cutoff, across tenants, oldest first.
	ListStaleSubmissions(
		ctx context.Context,
		req *ListStaleSubmissionsRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
}
