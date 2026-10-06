package graphql

import (
	"context"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const readOnlyExtensionName = "PlanReadOnlyGuard"

type ReadOnlyParams struct {
	fx.In

	Plans  services.PlanService
	Logger *zap.Logger
}

type ReadOnlyExtension struct {
	plans services.PlanService
	l     *zap.Logger
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = (*ReadOnlyExtension)(nil)

func NewReadOnlyExtension(p ReadOnlyParams) *ReadOnlyExtension {
	return &ReadOnlyExtension{
		plans: p.Plans,
		l:     p.Logger.Named("api.graphql.readonly"),
	}
}

func (*ReadOnlyExtension) ExtensionName() string {
	return readOnlyExtensionName
}

func (*ReadOnlyExtension) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e *ReadOnlyExtension) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	if e.plans == nil || !e.plans.EnforcesPlans() {
		return nil
	}
	if opCtx == nil || opCtx.Operation == nil || opCtx.Operation.Operation != ast.Mutation {
		return nil
	}

	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil || authCtx.OrganizationID.IsNil() || authCtx.BusinessUnitID.IsNil() {
		return nil
	}

	err := planservice.Admit(ctx, e.plans, &planservice.AdmissionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Write:  true,
		APIKey: authCtx.IsAPIKey(),
	})
	if err == nil {
		return nil
	}

	e.l.Debug("GraphQL mutation refused by the organization's plan",
		zap.String("operation", opCtx.OperationName),
		zap.String("organizationId", authCtx.OrganizationID.String()),
		zap.Error(err),
	)
	if status, found := gqlctx.ResponseStatusFrom(ctx); found {
		status.Override(http.StatusForbidden, 0)
	}

	return gqlerror.WrapIfUnwrapped(err)
}
