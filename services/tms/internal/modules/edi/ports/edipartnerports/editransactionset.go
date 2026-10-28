package edipartnerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type EDITransactionSetSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	IDs                []pulid.ID                     `json:"ids"`
	Standard           edidocument.EDIStandard        `json:"standard"`
	Status             edidocument.DocumentStatus     `json:"status"`
}

type EDITransactionSetRepository interface {
	SelectTransactionSetOptions(
		ctx context.Context,
		req *EDITransactionSetSelectOptionsRequest,
	) (*pagination.ListResult[*edidocument.EDITransactionSet], error)
}
