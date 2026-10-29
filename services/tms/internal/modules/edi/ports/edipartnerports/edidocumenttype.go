package edipartnerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ListEDIDocumentTypesRequest struct {
	Standard       edidocument.EDIStandard       `json:"standard"`
	TransactionSet edidocument.TransactionSet    `json:"transactionSet"`
	Direction      edidocument.DocumentDirection `json:"direction"`
	Status         edidocument.DocumentStatus    `json:"status"`
}

type EDIDocumentTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Standard           edidocument.EDIStandard        `json:"standard"`
	TransactionSet     edidocument.TransactionSet     `json:"transactionSet"`
	Direction          edidocument.DocumentDirection  `json:"direction"`
	Status             edidocument.DocumentStatus     `json:"status"`
}

type EDIDocumentTypeRepository interface {
	ListDocumentTypes(
		ctx context.Context,
		req ListEDIDocumentTypesRequest,
	) ([]*edidocument.EDIDocumentType, error)
	SelectDocumentTypeOptions(
		ctx context.Context,
		req *EDIDocumentTypeSelectOptionsRequest,
	) (*pagination.ListResult[*edidocument.EDIDocumentType], error)
}
