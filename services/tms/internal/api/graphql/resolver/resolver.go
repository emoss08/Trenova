package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/usstateservice"
	"github.com/emoss08/trenova/internal/core/services/workerptoservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                       *zap.Logger
	AnalyticsService             services.AnalyticsService
	OrganizationService          services.OrganizationService
	ShipmentService              services.ShipmentService
	ShipmentCommentService       services.ShipmentCommentService
	ShipmentEventService         services.ShipmentEventService
	ShipmentImportAssistant      services.ShipmentImportAssistantService `optional:"true"`
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	EquipmentTypeService         *equipmenttypeservice.Service
	TractorService               *tractorservice.Service
	TrailerService               *trailerservice.Service
	USStateService               *usstateservice.Service
	WorkerService                *workerservice.Service
	WorkerPTOService             *workerptoservice.Service
	PermissionEngine             services.PermissionEngine
}

type Resolver struct {
	l                            *zap.Logger
	analyticsService             services.AnalyticsService
	organizationService          services.OrganizationService
	shipmentService              services.ShipmentService
	shipmentCommentService       services.ShipmentCommentService
	shipmentEventService         services.ShipmentEventService
	shipmentImportAssistant      services.ShipmentImportAssistantService
	equipmentTypeService         *equipmenttypeservice.Service
	equipmentManufacturerService *equipmentmanufacturerservice.Service
	tractorService               *tractorservice.Service
	trailerService               *trailerservice.Service
	usStateService               *usstateservice.Service
	workerService                *workerservice.Service
	workerPTOService             *workerptoservice.Service
	permissionEngine             services.PermissionEngine
}

func New(p Params) *Resolver {
	return &Resolver{
		l:                            p.Logger.Named("api.graphql.resolver"),
		analyticsService:             p.AnalyticsService,
		organizationService:          p.OrganizationService,
		shipmentService:              p.ShipmentService,
		shipmentCommentService:       p.ShipmentCommentService,
		shipmentEventService:         p.ShipmentEventService,
		shipmentImportAssistant:      p.ShipmentImportAssistant,
		equipmentTypeService:         p.EquipmentTypeService,
		equipmentManufacturerService: p.EquipmentManufacturerService,
		tractorService:               p.TractorService,
		trailerService:               p.TrailerService,
		usStateService:               p.USStateService,
		workerService:                p.WorkerService,
		workerPTOService:             p.WorkerPTOService,
		permissionEngine:             p.PermissionEngine,
	}
}

func (r *Resolver) requirePermission(
	ctx context.Context,
	resource permission.Resource,
	operation permission.Operation,
) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	result, err := r.permissionEngine.Check(
		ctx,
		middleware.BuildPermissionCheckRequest(authCtx, resource.String(), operation),
	)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action",
		)
	}

	return authCtx, nil
}

func tenantInfo(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
