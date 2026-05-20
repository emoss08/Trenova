package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ListEDIDocumentTypesRequest struct {
	Standard       edi.EDIStandard       `json:"standard"`
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
	Status         edi.DocumentStatus    `json:"status"`
}

type EDIDocumentTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Standard           edi.EDIStandard                `json:"standard"`
	TransactionSet     edi.TransactionSet             `json:"transactionSet"`
	Direction          edi.DocumentDirection          `json:"direction"`
	Status             edi.DocumentStatus             `json:"status"`
}

type EDIDocumentTypeRepository interface {
	ListDocumentTypes(
		ctx context.Context,
		req ListEDIDocumentTypesRequest,
	) ([]*edi.EDIDocumentType, error)
	SelectDocumentTypeOptions(
		ctx context.Context,
		req *EDIDocumentTypeSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDIDocumentType], error)
}
