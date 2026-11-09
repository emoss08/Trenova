package scimhandler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	tenantSlugParam = "tenantSlug"
	idParam         = "id"
	directoryKey    = "trenova.scim.directory"
	maxBodyBytes    = 2 << 20
	basePath        = "/api/v1/scim/"
	authChallenge   = `Bearer realm="SCIM"`
)

type Params struct {
	fx.In

	Provisioner scimports.Provisioner
	RateLimiter *middleware.RateLimiter
	AppConfig   *appconfig.Config
	Logger      *zap.Logger
}

type Handler struct {
	provisioner scimports.Provisioner
	rateLimiter *middleware.RateLimiter
	appConfig   *appconfig.Config
	l           *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		provisioner: p.Provisioner,
		rateLimiter: p.RateLimiter,
		appConfig:   p.AppConfig,
		l:           p.Logger.Named("handler.scim"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/scim/:" + tenantSlugParam + "/v2")
	api.Use(middleware.NewBodyLimit(maxBodyBytes), h.requireDirectory())

	api.GET("/ServiceProviderConfig", h.serviceProviderConfig)
	api.GET("/ResourceTypes", h.resourceTypes)
	api.GET("/ResourceTypes/:"+idParam, h.resourceType)
	api.GET("/Schemas", h.schemas)
	api.GET("/Schemas/:"+idParam, h.schema)

	api.GET("/Users", h.listUsers)
	api.POST("/Users", h.createUser)
	api.GET("/Users/:"+idParam, h.getUser)
	api.PUT("/Users/:"+idParam, h.replaceUser)
	api.PATCH("/Users/:"+idParam, h.patchUser)
	api.DELETE("/Users/:"+idParam, h.deleteUser)

	api.GET("/Groups", h.listGroups)
	api.POST("/Groups", h.createGroup)
	api.GET("/Groups/:"+idParam, h.getGroup)
	api.PUT("/Groups/:"+idParam, h.replaceGroup)
	api.PATCH("/Groups/:"+idParam, h.patchGroup)
	api.DELETE("/Groups/:"+idParam, h.deleteGroup)
}

func (h *Handler) requireDirectory() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			h.refuse(c)
			return
		}

		dir, err := h.provisioner.Authenticate(c.Request.Context(), scimports.AuthenticateRequest{
			Token:      token,
			TenantSlug: c.Param(tenantSlugParam),
		})
		if err != nil {
			if scimErr, isSCIM := scimproto.AsError(err); isSCIM &&
				scimErr.Status == http.StatusUnauthorized {
				h.refuse(c)
				return
			}
			h.fail(c, err)
			c.Abort()
			return
		}

		if !h.rateLimiter.Allow(
			c,
			securityconfig.RateLimitScopeSCIM,
			"{"+dir.OrganizationID.String()+"}:"+dir.ID.String(),
		) {
			c.Abort()
			return
		}

		c.Set(directoryKey, dir)
		c.Request = c.Request.WithContext(auditctx.WithChannel(
			dbscope.WithTenant(c.Request.Context(), dir.DBTenant()),
			auditctx.ChannelSCIM,
		))
		c.Next()
	}
}

func (h *Handler) refuse(c *gin.Context) {
	if !h.rateLimiter.AllowClientIP(c) {
		c.Abort()
		return
	}

	c.Header("WWW-Authenticate", authChallenge)
	h.fail(c, scimproto.Unauthorized("The bearer token is not valid for this directory"))
	c.Abort()
}

func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)

	return token, token != ""
}

func directoryOf(c *gin.Context) *scimports.Directory {
	value, _ := c.Get(directoryKey)
	dir, _ := value.(*scimports.Directory)

	return dir
}

func (h *Handler) baseURL(c *gin.Context) string {
	origin := h.appConfig.GetWebBaseURL()
	if origin == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		origin = scheme + "://" + c.Request.Host
	}

	return origin + basePath + c.Param(tenantSlugParam) + "/v2"
}

func (h *Handler) readBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, scimproto.PayloadTooLarge("The request body is too large")
		}
		return nil, scimproto.InvalidSyntax("The request body could not be read")
	}

	return body, nil
}

func (h *Handler) respond(c *gin.Context, status int, value any) {
	body, err := sonic.Marshal(value)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.Data(status, scimproto.MediaType, body)
}

func (h *Handler) fail(c *gin.Context, err error) {
	scimErr := toSCIMError(err)
	if scimErr.Status >= http.StatusInternalServerError {
		h.l.Error("SCIM request failed",
			zap.String("method", c.Request.Method),
			zap.String("route", c.FullPath()),
			zap.Error(err))
	}

	body, marshalErr := sonic.Marshal(scimErr.Body())
	if marshalErr != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(scimErr.Status, scimproto.MediaType, body)
}

func toSCIMError(err error) *scimproto.Error {
	if scimErr, ok := scimproto.AsError(err); ok {
		return scimErr
	}

	switch {
	case errortypes.IsNotFoundError(err):
		return scimproto.NotFound("The resource was not found in this directory")
	case errortypes.IsAuthenticationError(err):
		return scimproto.Unauthorized("The bearer token is not valid for this directory")
	case errortypes.IsAuthorizationError(err):
		return scimproto.Forbidden(errortypes.Summary(err))
	case errortypes.IsQuotaExceededError(err), errortypes.IsPlanRestrictionError(err):
		return scimproto.Forbidden(errortypes.Summary(err))
	case errortypes.IsConflictError(err), errortypes.IsVersionMismatchError(err):
		return scimproto.Uniqueness(errortypes.Summary(err))
	}

	if multiErr, ok := errortypes.RefusalOf(err); ok {
		for _, fieldErr := range multiErr.Errors {
			if fieldErr.Code == errortypes.ErrDuplicate ||
				fieldErr.Code == errortypes.ErrAlreadyExists {
				return scimproto.Uniqueness(fieldErr.Error())
			}
		}
		return scimproto.InvalidValue(errortypes.Summary(err))
	}
	if fieldErr, ok := errors.AsType[*errortypes.Error](err); ok {
		if fieldErr.Code == errortypes.ErrDuplicate ||
			fieldErr.Code == errortypes.ErrAlreadyExists {
			return scimproto.Uniqueness(fieldErr.Error())
		}
		return scimproto.InvalidValue(fieldErr.Error())
	}
	if errortypes.IsBusinessError(err) {
		return scimproto.InvalidValue(errortypes.Summary(err))
	}

	return &scimproto.Error{
		Status: http.StatusInternalServerError,
		Detail: "The request could not be completed",
	}
}

func pathID(c *gin.Context) pulid.ID {
	return pulid.ID(strings.TrimSpace(c.Param(idParam)))
}

func listRequest(c *gin.Context, dir *scimports.Directory) *scimports.ListRequest {
	startIndex, err := strconv.Atoi(c.Query("startIndex"))
	if err != nil {
		startIndex = 1
	}
	count := scimports.MaxPageSize
	if raw, present := c.GetQuery("count"); present {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			count = parsed
		}
	}

	return &scimports.ListRequest{
		Directory:  dir,
		Filter:     c.Query("filter"),
		StartIndex: startIndex,
		Count:      count,
	}
}

func excludesMembers(c *gin.Context) bool {
	for _, attribute := range strings.Split(c.Query("excludedAttributes"), ",") {
		if scimproto.NormalizeAttribute(attribute) == "members" {
			return true
		}
	}

	return false
}
