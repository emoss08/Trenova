package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
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

	Logger                  *zap.Logger
	AnalyticsService        services.AnalyticsService
	ShipmentService         services.ShipmentService
	ShipmentCommentService  services.ShipmentCommentService
	ShipmentEventService    services.ShipmentEventService
	ShipmentImportAssistant services.ShipmentImportAssistantService `optional:"true"`
	TractorService          *tractorservice.Service
	TrailerService          *trailerservice.Service
	WorkerService           *workerservice.Service
	WorkerPTOService        *workerptoservice.Service
	PermissionEngine        services.PermissionEngine
}

type Resolver struct {
	l                       *zap.Logger
	analyticsService        services.AnalyticsService
	shipmentService         services.ShipmentService
	shipmentCommentService  services.ShipmentCommentService
	shipmentEventService    services.ShipmentEventService
	shipmentImportAssistant services.ShipmentImportAssistantService
	tractorService          *tractorservice.Service
	trailerService          *trailerservice.Service
	workerService           *workerservice.Service
	workerPTOService        *workerptoservice.Service
	permissionEngine        services.PermissionEngine
}

func New(p Params) *Resolver {
	return &Resolver{
		l:                       p.Logger.Named("api.graphql.resolver"),
		analyticsService:        p.AnalyticsService,
		shipmentService:         p.ShipmentService,
		shipmentCommentService:  p.ShipmentCommentService,
		shipmentEventService:    p.ShipmentEventService,
		shipmentImportAssistant: p.ShipmentImportAssistant,
		tractorService:          p.TractorService,
		trailerService:          p.TrailerService,
		workerService:           p.WorkerService,
		workerPTOService:        p.WorkerPTOService,
		permissionEngine:        p.PermissionEngine,
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
