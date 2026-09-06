package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListReviewTemplatesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
	Status string                   `json:"status"`
}

type GetReviewTemplateByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ReviewTemplateCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type CountReviewsByTemplateRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TemplateID pulid.ID              `json:"templateId"`
	OpenOnly   bool                  `json:"openOnly"`
}

type CountReviewsByTemplateIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
	OpenOnly    bool                  `json:"openOnly"`
}

type ListPerformanceReviewsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	// Statuses limits the list; empty means every status.
	Statuses        []worker.ReviewStatus `json:"statuses"`
	IncludeTemplate bool                  `json:"includeTemplate"`
	IncludeReviewer bool                  `json:"includeReviewer"`
}

// CountPerformanceReviewsRequest counts reviews across the organisation rather
// than for one worker, which is what the home screen needs and the per-worker
// list cannot answer.
type CountPerformanceReviewsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Statuses   []worker.ReviewStatus `json:"statuses"`
}

type GetPerformanceReviewByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeTemplate bool                  `json:"includeTemplate"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeReviewer bool                  `json:"includeReviewer"`
}

type PerformanceReviewRepository interface {
	ListTemplates(
		ctx context.Context,
		req *ListReviewTemplatesRequest,
	) (*pagination.CursorListResult[*worker.PerformanceReviewTemplate], error)
	ListActiveTemplates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.PerformanceReviewTemplate, error)
	GetTemplateByID(
		ctx context.Context,
		req *GetReviewTemplateByIDRequest,
	) (*worker.PerformanceReviewTemplate, error)
	TemplateCodeExists(ctx context.Context, req *ReviewTemplateCodeExistsRequest) (bool, error)
	CreateTemplate(
		ctx context.Context,
		entity *worker.PerformanceReviewTemplate,
	) (*worker.PerformanceReviewTemplate, error)
	UpdateTemplate(
		ctx context.Context,
		entity *worker.PerformanceReviewTemplate,
	) (*worker.PerformanceReviewTemplate, error)
	ClearDefaultTemplate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		exceptID pulid.ID,
	) error
	CountReviewsByTemplate(ctx context.Context, req *CountReviewsByTemplateRequest) (int, error)
	CountReviewsByTemplateIDs(
		ctx context.Context,
		req *CountReviewsByTemplateIDsRequest,
	) (map[pulid.ID]int, error)

	ListReviews(
		ctx context.Context,
		req *ListPerformanceReviewsRequest,
	) ([]*worker.PerformanceReview, error)
	CountReviews(
		ctx context.Context,
		req *CountPerformanceReviewsRequest,
	) (int, error)
	GetReviewByID(
		ctx context.Context,
		req *GetPerformanceReviewByIDRequest,
	) (*worker.PerformanceReview, error)
	CreateReview(
		ctx context.Context,
		entity *worker.PerformanceReview,
	) (*worker.PerformanceReview, error)
	UpdateReview(
		ctx context.Context,
		entity *worker.PerformanceReview,
	) (*worker.PerformanceReview, error)
	DeleteReview(ctx context.Context, req *GetPerformanceReviewByIDRequest) error
}
