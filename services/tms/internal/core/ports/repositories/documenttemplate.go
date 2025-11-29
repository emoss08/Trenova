package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListDocumentTemplateRequest struct {
	Filter         *pagination.QueryOptions         `json:"filter"         form:"filter"`
	DocumentTypeID *pulid.ID                        `json:"documentTypeId" form:"documentTypeId"`
	Status         *documenttemplate.TemplateStatus `json:"status"         form:"status"`
	IsDefault      *bool                            `json:"isDefault"      form:"isDefault"`
	IncludeType    bool                             `json:"includeType"    form:"includeType"`
}

type GetDocumentTemplateByIDRequest struct {
	ID          pulid.ID `json:"id"          form:"id"`
	OrgID       pulid.ID `json:"orgId"       form:"orgId"`
	BuID        pulid.ID `json:"buId"        form:"buId"`
	UserID      pulid.ID `json:"userId"      form:"userId"`
	IncludeType bool     `json:"includeType" form:"includeType"`
}

type GetDefaultTemplateRequest struct {
	OrgID          pulid.ID `json:"orgId"          form:"orgId"`
	BuID           pulid.ID `json:"buId"           form:"buId"`
	DocumentTypeID pulid.ID `json:"documentTypeId" form:"documentTypeId"`
}

type DocumentTemplateRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentTemplateRequest,
	) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error)
	GetByID(
		ctx context.Context,
		req GetDocumentTemplateByIDRequest,
	) (*documenttemplate.DocumentTemplate, error)
	GetByCode(
		ctx context.Context,
		orgID, buID pulid.ID,
		code string,
	) (*documenttemplate.DocumentTemplate, error)
	GetDefault(
		ctx context.Context,
		req GetDefaultTemplateRequest,
	) (*documenttemplate.DocumentTemplate, error)
	Create(
		ctx context.Context,
		dt *documenttemplate.DocumentTemplate,
	) (*documenttemplate.DocumentTemplate, error)
	Update(
		ctx context.Context,
		dt *documenttemplate.DocumentTemplate,
	) (*documenttemplate.DocumentTemplate, error)
	Delete(ctx context.Context, dt *documenttemplate.DocumentTemplate) error
	ClearDefaultForType(ctx context.Context, orgID, buID, documentTypeID pulid.ID) error
}

type ListGeneratedDocumentRequest struct {
	Filter        *pagination.QueryOptions           `json:"filter"        form:"filter"`
	ReferenceType *string                            `json:"referenceType" form:"referenceType"`
	ReferenceID   *pulid.ID                          `json:"referenceId"   form:"referenceId"`
	Status        *documenttemplate.GenerationStatus `json:"status"        form:"status"`
	IncludeType   bool                               `json:"includeType"   form:"includeType"`
}

type GetGeneratedDocumentByIDRequest struct {
	ID          pulid.ID `json:"id"          form:"id"`
	OrgID       pulid.ID `json:"orgId"       form:"orgId"`
	BuID        pulid.ID `json:"buId"        form:"buId"`
	UserID      pulid.ID `json:"userId"      form:"userId"`
	IncludeType bool     `json:"includeType" form:"includeType"`
}

type GetByReferenceRequest struct {
	OrgID   pulid.ID `json:"orgId"   form:"orgId"`
	BuID    pulid.ID `json:"buId"    form:"buId"`
	RefType string   `json:"refType" form:"refType"`
	RefID   pulid.ID `json:"refId"   form:"refId"`
}

type GeneratedDocumentRepository interface {
	List(
		ctx context.Context,
		req *ListGeneratedDocumentRequest,
	) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error)
	GetByID(
		ctx context.Context,
		req GetGeneratedDocumentByIDRequest,
	) (*documenttemplate.GeneratedDocument, error)
	GetByReference(
		ctx context.Context,
		req *GetByReferenceRequest,
	) ([]*documenttemplate.GeneratedDocument, error)
	Create(
		ctx context.Context,
		gd *documenttemplate.GeneratedDocument,
	) (*documenttemplate.GeneratedDocument, error)
	Update(
		ctx context.Context,
		gd *documenttemplate.GeneratedDocument,
	) (*documenttemplate.GeneratedDocument, error)
	Delete(ctx context.Context, gd *documenttemplate.GeneratedDocument) error
}
