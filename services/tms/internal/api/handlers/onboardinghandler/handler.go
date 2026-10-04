package onboardinghandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.OnboardingService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.OnboardingService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

type organizationBody struct {
	Name         string   `json:"name"`
	Timezone     string   `json:"timezone"`
	AddressLine1 string   `json:"addressLine1"`
	City         string   `json:"city"`
	StateID      pulid.ID `json:"stateId"`
	PostalCode   string   `json:"postalCode"`
	ScacCode     string   `json:"scacCode"`
	DOTNumber    string   `json:"dotNumber"`
}

type completeBody struct {
	Organization   organizationBody         `json:"organization"`
	OperationType  onboarding.OperationType `json:"operationType"`
	LoadSampleData bool                     `json:"loadSampleData"`
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/onboarding/")
	api.GET("", h.get)
	api.POST(
		"complete/",
		h.pm.RequirePermission(permission.ResourceOrganization.String(), permission.OpUpdate),
		h.complete,
	)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	tenantInfo := actorutil.TenantInfoFrom(authCtx)
	if authCtx != nil {
		tenantInfo.UserID = authCtx.UserID
	}

	state, err := h.service.Get(c.Request.Context(), tenantInfo)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, state)
}

func (h *Handler) complete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body completeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if authCtx == nil || authCtx.UserID.IsNil() {
		h.eh.HandleError(c, errortypes.NewAuthorizationError("Onboarding must be completed by a person"))
		return
	}

	tenantInfo := actorutil.TenantInfoFrom(authCtx)
	tenantInfo.UserID = authCtx.UserID

	state, err := h.service.Complete(c.Request.Context(), &services.CompleteOnboardingRequest{
		TenantInfo: tenantInfo,
		Actor:      actorutil.FromAuthContext(authCtx),
		Organization: services.OnboardingOrganization{
			Name:         body.Organization.Name,
			Timezone:     body.Organization.Timezone,
			AddressLine1: body.Organization.AddressLine1,
			City:         body.Organization.City,
			StateID:      body.Organization.StateID,
			PostalCode:   body.Organization.PostalCode,
			ScacCode:     body.Organization.ScacCode,
			DOTNumber:    body.Organization.DOTNumber,
		},
		OperationType:  body.OperationType,
		LoadSampleData: body.LoadSampleData,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, state)
}
