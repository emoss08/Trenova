package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type EDITransactionSetSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	IDs                []pulid.ID                     `json:"ids"`
	Standard           edi.EDIStandard                `json:"standard"`
	Status             edi.DocumentStatus             `json:"status"`
}

type EDITransactionSetRepository interface {
	SelectTransactionSetOptions(
		ctx context.Context,
		req *EDITransactionSetSelectOptionsRequest,
	) (*pagination.ListResult[*edi.EDITransactionSet], error)
}
