package jurisdictionrulehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              services.JurisdictionRuleService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service services.JurisdictionRuleService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
	logger  *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
		logger:  p.Logger.Named("api.jurisdiction-rule-handler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/jurisdiction-rules")

	api.GET("/",
		h.pm.RequirePermission(permission.ResourceJurisdictionRule.String(), permission.OpRead),
		h.list,
	)
	api.GET("/:ruleID/",
		h.pm.RequirePermission(permission.ResourceJurisdictionRule.String(), permission.OpRead),
		h.get,
	)
	api.POST("/",
		h.pm.RequirePermission(permission.ResourceJurisdictionRule.String(), permission.OpCreate),
		h.create,
	)
	api.PUT("/:ruleID/",
		h.pm.RequirePermission(permission.ResourceJurisdictionRule.String(), permission.OpUpdate),
		h.update,
	)
	api.POST("/:ruleID/verify/",
		h.pm.RequirePermission(permission.ResourceJurisdictionRule.String(), permission.OpApprove),
		h.verify,
	)

	h.registerOverrideRoutes(rg)
}

// @Summary List jurisdiction rules
// @Description Shared oversize limits per state. This data is global — every organization reads the same rows.
// @ID listJurisdictionRules
// @Tags Jurisdiction Rules
// @Produce json
// @Param verificationState query string false "Filter by verification state"
// @Param status query string false "Filter by status"
// @Success 200 {object} pagination.ListResult[jurisdictionrule.JurisdictionRule]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /jurisdiction-rules/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*jurisdictionrule.JurisdictionRule], error) {
			return h.service.List(c.Request.Context(), &services.ListJurisdictionRulesRequest{
				Filter: req,
				VerificationState: jurisdictionrule.VerificationState(
					helpers.QueryStringTrimmed(c, "verificationState"),
				),
				Status: jurisdictionrule.Status(helpers.QueryStringTrimmed(c, "status")),
			})
		},
	)
}

// @Summary Get a jurisdiction rule
// @ID getJurisdictionRule
// @Tags Jurisdiction Rules
// @Produce json
// @Param ruleID path string true "Rule ID"
// @Success 200 {object} jurisdictionrule.JurisdictionRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /jurisdiction-rules/{ruleID}/ [get]
func (h *Handler) get(c *gin.Context) {
	ruleID, err := pulid.MustParse(c.Param("ruleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), ruleID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a jurisdiction rule
// @Description Creates a globally visible rule. It always lands unverified; use the verify endpoint to confirm it.
// @ID createJurisdictionRule
// @Tags Jurisdiction Rules
// @Accept json
// @Produce json
// @Param request body jurisdictionrule.JurisdictionRule true "Rule payload"
// @Success 201 {object} jurisdictionrule.JurisdictionRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /jurisdiction-rules/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(jurisdictionrule.JurisdictionRule)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(
		c.Request.Context(), entity, actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a jurisdiction rule
// @Description Changing any limit, permit term or curfew resets the rule to unverified.
// @ID updateJurisdictionRule
// @Tags Jurisdiction Rules
// @Accept json
// @Produce json
// @Param ruleID path string true "Rule ID"
// @Param request body jurisdictionrule.JurisdictionRule true "Rule payload"
// @Success 200 {object} jurisdictionrule.JurisdictionRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /jurisdiction-rules/{ruleID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	ruleID, err := pulid.MustParse(c.Param("ruleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(jurisdictionrule.JurisdictionRule)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	// The path owns identity. A payload naming a different row must not be able
	// to redirect the write.
	entity.ID = ruleID

	updated, err := h.service.Update(
		c.Request.Context(), entity, actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

type verifyBody struct {
	VerificationState string `json:"verificationState"`
	SourceNote        string `json:"sourceNote"`
	SourceURL         string `json:"sourceUrl"`
}

// @Summary Verify or dispute a jurisdiction rule
// @Description Records that this rule was checked against the issuing state. Writes only the verification columns, so it cannot alter the limits it is confirming.
// @ID verifyJurisdictionRule
// @Tags Jurisdiction Rules
// @Accept json
// @Produce json
// @Param ruleID path string true "Rule ID"
// @Param request body verifyBody true "Verification payload"
// @Success 200 {object} jurisdictionrule.JurisdictionRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /jurisdiction-rules/{ruleID}/verify/ [post]
func (h *Handler) verify(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	ruleID, err := pulid.MustParse(c.Param("ruleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(verifyBody)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	verified, err := h.service.Verify(
		c.Request.Context(),
		&services.VerifyJurisdictionRuleRequest{
			RuleID:     ruleID,
			State:      jurisdictionrule.VerificationState(body.VerificationState),
			SourceNote: body.SourceNote,
			SourceURL:  body.SourceURL,
			Actor:      actorutil.FromAuthContext(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, verified)
}

func (h *Handler) registerOverrideRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/jurisdiction-rule-overrides")
	resource := permission.ResourceJurisdictionRuleOverride.String()

	api.GET("/:overrideID/",
		h.pm.RequirePermission(resource, permission.OpRead), h.getOverride)
	api.POST("/",
		h.pm.RequirePermission(resource, permission.OpCreate), h.createOverride)
	api.PUT("/:overrideID/",
		h.pm.RequirePermission(resource, permission.OpUpdate), h.updateOverride)
	api.DELETE("/:overrideID/",
		h.pm.RequirePermission(resource, permission.OpDelete), h.deleteOverride)
}

// @Summary Get a carrier override
// @ID getJurisdictionRuleOverride
// @Tags Jurisdiction Rules
// @Produce json
// @Param overrideID path string true "Override ID"
// @Success 200 {object} jurisdictionrule.Override
// @Security BearerAuth
// @Router /jurisdiction-rule-overrides/{overrideID}/ [get]
func (h *Handler) getOverride(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	overrideID, err := pulid.MustParse(c.Param("overrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetOverrideByID(
		c.Request.Context(),
		overrideID,
		actorutil.TenantInfoFrom(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a carrier override
// @Description Holds this organization to a stricter limit than a state requires. An override that would loosen a statutory limit is rejected.
// @ID createJurisdictionRuleOverride
// @Tags Jurisdiction Rules
// @Accept json
// @Produce json
// @Param request body jurisdictionrule.Override true "Override payload"
// @Success 201 {object} jurisdictionrule.Override
// @Security BearerAuth
// @Router /jurisdiction-rule-overrides/ [post]
func (h *Handler) createOverride(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(jurisdictionrule.Override)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	// The session owns the tenant. Taking it from the body would let a caller
	// write an override into another organization.
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	created, err := h.service.CreateOverride(
		c.Request.Context(), entity, actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a carrier override
// @ID updateJurisdictionRuleOverride
// @Tags Jurisdiction Rules
// @Accept json
// @Produce json
// @Param overrideID path string true "Override ID"
// @Param request body jurisdictionrule.Override true "Override payload"
// @Success 200 {object} jurisdictionrule.Override
// @Security BearerAuth
// @Router /jurisdiction-rule-overrides/{overrideID}/ [put]
func (h *Handler) updateOverride(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	overrideID, err := pulid.MustParse(c.Param("overrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(jurisdictionrule.Override)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity.ID = overrideID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	updated, err := h.service.UpdateOverride(
		c.Request.Context(), entity, actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Remove a carrier override
// @Description Returns this jurisdiction to its statutory limits.
// @ID deleteJurisdictionRuleOverride
// @Tags Jurisdiction Rules
// @Produce json
// @Param overrideID path string true "Override ID"
// @Success 204
// @Security BearerAuth
// @Router /jurisdiction-rule-overrides/{overrideID}/ [delete]
func (h *Handler) deleteOverride(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	overrideID, err := pulid.MustParse(c.Param("overrideID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.DeleteOverride(
		c.Request.Context(),
		overrideID,
		actorutil.TenantInfoFrom(authCtx),
		actorutil.FromAuthContext(authCtx),
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
