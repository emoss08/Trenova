package documentintelligenceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type noopDocumentSearchProjectionService struct{}

func (noopDocumentSearchProjectionService) Upsert(
	context.Context,
	*document.Document,
	string,
) error {
	return nil
}

func (noopDocumentSearchProjectionService) Delete(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) error {
	return nil
}

var _ serviceports.DocumentSearchProjectionService = noopDocumentSearchProjectionService{}
