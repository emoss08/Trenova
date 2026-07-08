package graphql

import (
	"net/http"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	playgroundContentSecurityPolicy = "default-src 'none'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; font-src 'self' data: https://cdn.jsdelivr.net; img-src 'self' data:; connect-src 'self' https://cdn.jsdelivr.net; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
)

type Params struct {
	fx.In

	Config        *config.Config
	Logger        *zap.Logger
	ErrorHandler  *helpers.ErrorHandler
	LoaderFactory *loaders.Factory
	PersistedOps  *PersistedOperationManifest
	Server        *gqlhandler.Server
}

type Handler struct {
	cfg           *config.Config
	l             *zap.Logger
	eh            *helpers.ErrorHandler
	loaderFactory *loaders.Factory
	persistedOps  *PersistedOperationManifest
	server        *gqlhandler.Server
}

func New(p Params) *Handler {
	return &Handler{
		cfg:           p.Config,
		l:             p.Logger.Named("api.graphql"),
		eh:            p.ErrorHandler,
		loaderFactory: p.LoaderFactory,
		persistedOps:  p.PersistedOps,
		server:        p.Server,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/graphql", h.handle)
}

func (h *Handler) RegisterPlaygroundRoutes(rg *gin.RouterGroup) {
	rg.GET("/graphql", h.handlePlayground)
}

func (h *Handler) handle(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot access GraphQL"))
		return
	}

	if err := rewritePersistedOperationRequest(
		c.Request,
		h.persistedOps,
		enforcePersistedOperations(h.cfg),
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reqCtx := gqlctx.WithAuthContext(c.Request.Context(), authCtx)
	reqCtx = gqlctx.WithRequestID(reqCtx, requestid.Get(c))
	reqCtx = loaders.WithLoaders(
		reqCtx,
		h.loaderFactory.NewForTenant(pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}),
	)

	c.Request = c.Request.WithContext(reqCtx)
	h.l.Debug("handling GraphQL request", zap.String("request_id", requestid.Get(c)))
	h.server.ServeHTTP(c.Writer, c.Request)
}

func (h *Handler) handlePlayground(c *gin.Context) {
	if !h.playgroundEnabled() {
		c.Status(404)
		return
	}

	c.Header("Content-Security-Policy", playgroundContentSecurityPolicy)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(playgroundHTML))
}

func (h *Handler) playgroundEnabled() bool {
	return h.cfg.App.Debug || h.cfg.App.IsDevelopment() || h.cfg.App.IsTest()
}
