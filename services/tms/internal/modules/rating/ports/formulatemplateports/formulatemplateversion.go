package formulatemplateports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
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

type ListVersionAuthorsRequest struct {
	TenantInfo   pagination.TenantInfo
	TemplateID   pulid.ID
	AfterVersion int64
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

type GetEffectiveVersionRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	AsOf       int64
}

type UpdateEffectiveDateRequest struct {
	TenantInfo    pagination.TenantInfo
	TemplateID    pulid.ID
	VersionNumber int64
	EffectiveFrom *int64
}

type ListScheduledVersionsRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
}

type GetLatestVersionByStatusRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Status     formulatemplate.Status
}

type ListLibraryUsagesRequest struct {
	TenantInfo    pagination.TenantInfo
	LibraryID     pulid.ID
	VersionNumber *int64
	Limit         int
}

type LibraryUsage struct {
	TemplateID           pulid.ID               `json:"templateId"`
	TemplateName         string                 `json:"templateName"`
	TemplateStatus       formulatemplate.Status `json:"templateStatus"`
	TemplateVersion      int64                  `json:"templateVersion"`
	VersionStatus        formulatemplate.Status `json:"versionStatus"`
	LibraryVersionNumber int64                  `json:"libraryVersionNumber"`
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

	GetEffectiveVersion(
		ctx context.Context,
		req *GetEffectiveVersionRequest,
	) (*formulatemplate.FormulaTemplateVersion, error)

	UpdateEffectiveDate(
		ctx context.Context,
		req *UpdateEffectiveDateRequest,
	) (*formulatemplate.FormulaTemplateVersion, error)

	ListScheduled(
		ctx context.Context,
		req *ListScheduledVersionsRequest,
	) ([]*formulatemplate.FormulaTemplateVersion, error)

	ListApproved(
		ctx context.Context,
		req *ListScheduledVersionsRequest,
	) ([]*formulatemplate.FormulaTemplateVersion, error)

	ListAuthorIDsSince(
		ctx context.Context,
		req *ListVersionAuthorsRequest,
	) ([]pulid.ID, error)

	GetLatestByStatus(
		ctx context.Context,
		req *GetLatestVersionByStatusRequest,
	) (*formulatemplate.FormulaTemplateVersion, error)

	ClearScheduled(
		ctx context.Context,
		req *ListScheduledVersionsRequest,
	) (int64, error)

	ListLibraryUsages(
		ctx context.Context,
		req *ListLibraryUsagesRequest,
	) ([]*LibraryUsage, error)
}
