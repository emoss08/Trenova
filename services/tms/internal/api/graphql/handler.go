package graphql

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	rejectionReasonAPIKey             = "api_key"
	rejectionReasonPersistedOperation = "persisted_operation"
	rejectionReasonBodyTooLarge       = "body_too_large"

	requestTooLargeErrorCode = "REQUEST_TOO_LARGE"

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
	Metrics       *metrics.Registry
}

type Handler struct {
	cfg           *config.Config
	l             *zap.Logger
	eh            *helpers.ErrorHandler
	loaderFactory *loaders.Factory
	persistedOps  *PersistedOperationManifest
	server        *gqlhandler.Server
	metrics       *metrics.GraphQL
}

func New(p Params) *Handler {
	return &Handler{
		cfg:           p.Config,
		l:             p.Logger.Named("api.graphql"),
		eh:            p.ErrorHandler,
		loaderFactory: p.LoaderFactory,
		persistedOps:  p.PersistedOps,
		server:        p.Server,
		metrics:       p.Metrics.GraphQL,
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
		h.metrics.RecordRejection(rejectionReasonAPIKey)
		h.eh.HandleError(c, errortypes.NewAuthorizationError("API keys cannot access GraphQL"))
		return
	}

	maxBodyBytes := h.cfg.Security.GraphQL.GetMaxRequestBodyBytes()
	if c.Request.ContentLength > maxBodyBytes {
		h.rejectRequestTooLarge(c, maxBodyBytes)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	if err := rewritePersistedOperationRequest(
		c.Request,
		h.persistedOps,
		enforcePersistedOperations(h.cfg),
	); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.rejectRequestTooLarge(c, maxBodyBytes)
			return
		}
		h.metrics.RecordRejection(rejectionReasonPersistedOperation)
		h.eh.HandleError(c, err)
		return
	}

	reqCtx := gqlctx.WithAuthContext(c.Request.Context(), authCtx)
	reqCtx = gqlctx.WithRequestID(reqCtx, requestid.Get(c))
	reqCtx = gqlctx.WithClientInfo(reqCtx, gqlctx.ClientInfo{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	reqCtx = gqlctx.WithPermissionMemo(reqCtx, gqlctx.NewPermissionMemo())
	responseStatus := gqlctx.NewResponseStatus()
	reqCtx = gqlctx.WithResponseStatus(reqCtx, responseStatus)
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
	h.server.ServeHTTP(
		statusOverrideWriter{ResponseWriter: c.Writer, status: responseStatus},
		c.Request,
	)
}

func (h *Handler) rejectRequestTooLarge(c *gin.Context, maxBodyBytes int64) {
	h.metrics.RecordRejection(rejectionReasonBodyTooLarge)

	body, err := sonic.Marshal(map[string]any{
		"errors": []map[string]any{{
			"message": "GraphQL request body exceeds the " +
				strconv.FormatInt(maxBodyBytes, 10) + " byte limit",
			"extensions": map[string]any{
				"code": requestTooLargeErrorCode,
				"type": h.cfg.App.GetProblemTypeBaseURI() +
					string(helpers.ProblemTypeValidation),
				"traceId": requestid.Get(c),
			},
		}},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusRequestEntityTooLarge, "application/json; charset=utf-8", body)
	c.Abort()
}

type statusOverrideWriter struct {
	http.ResponseWriter
	status *gqlctx.ResponseStatus
}

func (w statusOverrideWriter) WriteHeader(code int) {
	if override, ok := w.status.Code(); ok && code == http.StatusUnprocessableEntity {
		if retryAfter := w.status.RetryAfter(); retryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		}
		code = override
	}

	w.ResponseWriter.WriteHeader(code)
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
	return devToolingEnabled(h.cfg)
}
