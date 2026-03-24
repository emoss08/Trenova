package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateVersionRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	ChangeMessage string
}

type ListVersionsRequest struct {
	Filter     *pagination.QueryOptions
	TemplateID pulid.ID
}

type GetVersionRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	VersionNumber int64
}

type GetVersionRangeRequest struct {
	TenantInfo  pagination.TenantInfo
	TemplateID  pulid.ID
	FromVersion int64
	ToVersion   int64
}

type ForkTemplateRequest struct {
	TenantInfo       pagination.TenantInfo
	SourceTemplateID pulid.ID
	SourceVersion    *int64
	NewName          string
	ChangeMessage    string
}

type RollbackRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	TargetVersion int64
	ChangeMessage string
}

type CompareVersionsRequest struct {
	TenantInfo  pagination.TenantInfo
	TemplateID  pulid.ID
	FromVersion int64
	ToVersion   int64
}

type GetLineageRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

type GetForkedTemplatesRequest struct {
	TenantInfo       pagination.TenantInfo
	SourceTemplateID pulid.ID
}

type UpdateVersionTagsRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	VersionNumber int64
	Tags          []string
}

type FormulaTemplateVersionRepository interface {
	Create(
		ctx context.Context,
		version *formulatemplate.FormulaTemplateVersion,
	) (*formulatemplate.FormulaTemplateVersion, error)

	GetByTemplateAndVersion(
		ctx context.Context,
		req *GetVersionRequest,
	) (*formulatemplate.FormulaTemplateVersion, error)

	List(
		ctx context.Context,
		req *ListVersionsRequest,
	) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error)

	GetVersionRange(
		ctx context.Context,
		req *GetVersionRangeRequest,
	) ([]*formulatemplate.FormulaTemplateVersion, error)

	GetLatestVersion(
		ctx context.Context,
		templateID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*formulatemplate.FormulaTemplateVersion, error)

	GetForkedTemplates(
		ctx context.Context,
		req *GetForkedTemplatesRequest,
	) ([]*formulatemplate.FormulaTemplate, error)

	UpdateTags(
		ctx context.Context,
		req *UpdateVersionTagsRequest,
	) (*formulatemplate.FormulaTemplateVersion, error)
}
