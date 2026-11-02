package documenthandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentimportassistantports"
	"go.uber.org/zap"
)

func NewTestHandlerWithConfig(
	service *documentservice.Service,
	eh *helpers.ErrorHandler,
	pm *middleware.PermissionMiddleware,
	cfg *storageconfig.Config,
	importAssistant ...shipmentimportassistantports.ShipmentImportAssistantService,
) *Handler {
	var assistant shipmentimportassistantports.ShipmentImportAssistantService
	if len(importAssistant) > 0 {
		assistant = importAssistant[0]
	}

	storageConfig := storageconfig.Config{}
	if cfg != nil {
		storageConfig = *cfg
	}

	return &Handler{
		service:         service,
		importAssistant: assistant,
		eh:              eh,
		pm:              pm,
		storageConfig:   storageConfig,
		logger:          zap.NewNop(),
	}
}
