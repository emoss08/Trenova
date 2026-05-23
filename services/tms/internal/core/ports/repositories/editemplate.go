package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDITemplatesRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	TransactionSet edi.TransactionSet       `json:"transactionSet"`
	Direction      edi.DocumentDirection    `json:"direction"`
	Status         edi.TemplateStatus       `json:"status"`
}

type EDITemplateSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	TransactionSet     edi.TransactionSet             `json:"transactionSet"`
	Direction          edi.DocumentDirection          `json:"direction"`
	Status             edi.TemplateStatus             `json:"status"`
}

type GetEDITemplateByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITemplateRequest struct {
	Template        *edi.EDITemplate                `json:"template"`
	Version         *edi.EDITemplateVersion         `json:"version"`
	Segments        []*edi.EDITemplateSegment       `json:"segments"`
	ScriptLibraries []*edi.EDITemplateScriptLibrary `json:"scriptLibraries"`
}

type GetActiveEDITemplateVersionRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	VersionID  pulid.ID              `json:"versionId"`
}

type ListEDITemplateVersionsRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDITemplateVersionByIDRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	VersionID  pulid.ID              `json:"versionId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ReplaceEDITemplateVersionSegmentsRequest struct {
	Version  *edi.EDITemplateVersion   `json:"version"`
	Segments []*edi.EDITemplateSegment `json:"segments"`
}

type ListEDITemplateScriptLibrariesRequest struct {
	TemplateID pulid.ID              `json:"templateId"`
	VersionID  pulid.ID              `json:"versionId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CreateEDITemplateVersionRequest struct {
	Version         *edi.EDITemplateVersion         `json:"version"`
	Segments        []*edi.EDITemplateSegment       `json:"segments"`
	ScriptLibraries []*edi.EDITemplateScriptLibrary `json:"scriptLibraries"`
}

type ReplaceEDITemplateVersionScriptLibrariesRequest struct {
	Version         *edi.EDITemplateVersion         `json:"version"`
	ScriptLibraries []*edi.EDITemplateScriptLibrary `json:"scriptLibraries"`
}

type ActivateEDITemplateVersionRequest struct {
	VersionID  pulid.ID              `json:"versionId"`
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActorID    pulid.ID              `json:"actorId"`
	Notes      string                `json:"notes"`
	IsRollback bool                  `json:"isRollback"`
}

type ArchiveEDITemplateVersionRequest struct {
	VersionID  pulid.ID              `json:"versionId"`
	TemplateID pulid.ID              `json:"templateId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActorID    pulid.ID              `json:"actorId"`
	Notes      string                `json:"notes"`
}

type EDITemplateRepository interface {
	List(
		ctx context.Context,
		req *ListEDITemplatesRequest,
	) (*pagination.ListResult[*edi.EDITemplate], error)
	SelectTemplateOptions(
		ctx context.Context,
		req *EDITemplateSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDITemplate], error)
	GetTemplateByID(ctx context.Context, req GetEDITemplateByIDRequest) (*edi.EDITemplate, error)
	CreateTemplate(
		ctx context.Context,
		req *CreateEDITemplateRequest,
	) (*edi.EDITemplate, *edi.EDITemplateVersion, error)
	UpdateTemplate(ctx context.Context, entity *edi.EDITemplate) (*edi.EDITemplate, error)
	ListTemplateVersions(
		ctx context.Context,
		req ListEDITemplateVersionsRequest,
	) ([]*edi.EDITemplateVersion, error)
	GetTemplateVersionByID(
		ctx context.Context,
		req GetEDITemplateVersionByIDRequest,
	) (*edi.EDITemplateVersion, error)
	GetActiveTemplateVersion(
		ctx context.Context,
		req GetActiveEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	CreateTemplateVersion(
		ctx context.Context,
		req *CreateEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	UpdateTemplateVersionMetadata(
		ctx context.Context,
		version *edi.EDITemplateVersion,
	) (*edi.EDITemplateVersion, error)
	ReplaceTemplateVersionSegments(
		ctx context.Context,
		req ReplaceEDITemplateVersionSegmentsRequest,
	) (*edi.EDITemplateVersion, error)
	ListTemplateScriptLibraries(
		ctx context.Context,
		req ListEDITemplateScriptLibrariesRequest,
	) ([]*edi.EDITemplateScriptLibrary, error)
	ReplaceTemplateVersionScriptLibraries(
		ctx context.Context,
		req ReplaceEDITemplateVersionScriptLibrariesRequest,
	) (*edi.EDITemplateVersion, error)
	ActivateTemplateVersion(
		ctx context.Context,
		req ActivateEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	ArchiveTemplateVersion(
		ctx context.Context,
		req ArchiveEDITemplateVersionRequest,
	) (*edi.EDITemplateVersion, error)
	EnsureBase204Template(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*edi.EDITemplate, *edi.EDITemplateVersion, error)
}
