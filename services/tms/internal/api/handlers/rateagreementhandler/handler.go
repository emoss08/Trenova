package rateagreementhandler

import (
	"errors"
	"io"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *rateagreementservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *rateagreementservice.Service
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

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceRateAgreement.String()

	api := rg.Group("/rate-agreements")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateAgreementID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:rateAgreementID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)

	api.GET(
		"/:rateAgreementID/rules/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listRules,
	)
	api.POST(
		"/:rateAgreementID/rules/amend/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.amendRules,
	)
	api.GET(
		"/:rateAgreementID/versions/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listVersions,
	)

	// Registered before the parameterized routes so "rate-increase" is never
	// read as an agreement ID.
	api.POST(
		"/rate-increase/preview/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.previewRateIncrease,
	)
	api.POST(
		"/rate-increase/apply/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.applyRateIncrease,
	)

	api.POST(
		"/:rateAgreementID/duplicate/",
		h.pm.RequirePermission(resource, permission.OpDuplicate),
		h.duplicate,
	)

	api.POST(
		"/:rateAgreementID/submit/",
		h.pm.RequirePermission(resource, permission.OpSubmit),
		h.review(h.service.Submit),
	)
	api.POST(
		"/:rateAgreementID/approve/",
		h.pm.RequirePermission(resource, permission.OpApprove),
		h.review(h.service.Approve),
	)
	api.POST(
		"/:rateAgreementID/reject/",
		h.pm.RequirePermission(resource, permission.OpReject),
		h.review(h.service.Reject),
	)
	api.POST(
		"/:rateAgreementID/suspend/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.review(h.service.Suspend),
	)
	api.POST(
		"/:rateAgreementID/resume/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.review(h.service.Resume),
	)
	api.POST(
		"/:rateAgreementID/archive/",
		h.pm.RequirePermission(resource, permission.OpArchive),
		h.review(h.service.Archive),
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
}

// @Summary List rate agreements
// @ID listRateAgreements
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param partyType query string false "Filter by party type" Enums(Customer, Carrier)
// @Param status query string false "Filter by status"
// @Success 200 {object} pagination.Response[[]rateagreement.RateAgreement]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*rateagreement.RateAgreement], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRateAgreementRequest{
					Filter:    req,
					PartyType: rateagreement.PartyType(helpers.QueryString(c, "partyType")),
					Status:    rateagreement.Status(helpers.QueryString(c, "status")),
				},
			)
		},
	)
}

// @Summary Get a rate agreement
// @ID getRateAgreement
// @Tags Rate Agreements
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param includeChildren query bool false "Include rules, accessorials and the fuel binding"
// @Success 200 {object} rateagreement.RateAgreement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetRateAgreementByIDRequest{
			RateAgreementID: agreementID,
			TenantInfo:      tenantOf(authCtx),
			IncludeChildren: helpers.QueryBool(c, "includeChildren", true),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a rate agreement
// @ID createRateAgreement
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param request body rateagreement.RateAgreement true "Rate agreement payload"
// @Success 201 {object} rateagreement.RateAgreement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(rateagreement.RateAgreement)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a rate agreement
// @ID updateRateAgreement
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param request body rateagreement.RateAgreement true "Rate agreement payload"
// @Success 200 {object} rateagreement.RateAgreement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	entity := new(rateagreement.RateAgreement)
	entity.ID = agreementID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = agreementID

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary List a rate agreement's rules
// @ID listRateAgreementRules
// @Tags Rate Agreements
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param asOf query int false "Only rules effective at this epoch second"
// @Param includeInactive query bool false "Include rules that are not active"
// @Param laneKey query string false "Only rules on this lane"
// @Param includeSuperseded query bool false "Keep closed-out rules — the lane's full history, newest first"
// @Success 200 {array} rateagreement.RateAgreementRule
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID}/rules [get]
func (h *Handler) listRules(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	rules, err := h.service.ListRules(
		c.Request.Context(),
		&repositories.ListRateAgreementRulesRequest{
			TenantInfo:        tenantOf(authCtx),
			RateAgreementID:   agreementID,
			AsOf:              helpers.QueryInt64(c, "asOf", 0),
			IncludeInactive:   helpers.QueryBool(c, "includeInactive", false),
			LaneKey:           helpers.QueryString(c, "laneKey"),
			IncludeSuperseded: helpers.QueryBool(c, "includeSuperseded", false),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rules)
}

type amendRulesRequest struct {
	EffectiveFrom int64                              `json:"effectiveFrom"`
	SupersededIDs []pulid.ID                         `json:"supersededIds"`
	Rules         []*rateagreement.RateAgreementRule `json:"rules"`
}

// @Summary Amend a rate agreement's rules
// @Description Closes out the rules a change replaces and inserts their successors, in one transaction. Nothing is edited in place, so the superseded rates keep their history.
// @ID amendRateAgreementRules
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param request body amendRulesRequest true "Amendment payload"
// @Success 200 {object} rateagreement.RateAgreement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID}/rules/amend [post]
func (h *Handler) amendRules(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	body := new(amendRulesRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	amended, err := h.service.AmendRules(
		c.Request.Context(),
		&repositories.AmendRateAgreementRulesRequest{
			TenantInfo:      tenantOf(authCtx),
			RateAgreementID: agreementID,
			EffectiveFrom:   body.EffectiveFrom,
			SupersededIDs:   body.SupersededIDs,
			Rules:           body.Rules,
		},
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, amended)
}

// @Summary List a rate agreement's versions
// @ID listRateAgreementVersions
// @Tags Rate Agreements
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]rateagreement.RateAgreementVersion]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID}/versions [get]
func (h *Handler) listVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*rateagreement.RateAgreementVersion], error) {
			return h.service.ListVersions(
				c.Request.Context(),
				&repositories.ListRateAgreementVersionsRequest{
					TenantInfo:      tenantOf(authCtx),
					RateAgreementID: agreementID,
					Limit:           req.Pagination.Limit,
					Offset:          req.Pagination.Offset,
				},
			)
		},
	)
}

// @Summary List rate agreement options
// @ID listRateAgreementOptions
// @Tags Rate Agreements
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]rateagreement.RateAgreement]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*rateagreement.RateAgreement], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

type reviewRequest struct {
	Comment string `json:"comment"`
}

// review turns one of the service's review actions into a route.
//
// Every one of them takes the same request and returns the same thing, so a
// separate handler per action would be six copies of the same twelve lines
// differing only in which method they call.
// @Summary Preview a general rate increase
// @Description Answers what a bulk rate change would do — every lane's before and after — without doing any of it.
// @ID previewRateIncrease
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param request body rateagreementservice.RateIncreaseRequest true "Scope, adjustment, and effective date"
// @Success 200 {object} rateagreementservice.RateIncreasePlan
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/rate-increase/preview/ [post]
func (h *Handler) previewRateIncrease(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(rateagreementservice.RateIncreaseRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(authCtx)

	plan, err := h.service.PlanRateIncrease(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan)
}

// @Summary Apply a general rate increase
// @Description Closes out every affected rule and inserts its successor at the new rate, effective the announced date. History is never mutated — the old rates stay readable forever.
// @ID applyRateIncrease
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param request body rateagreementservice.RateIncreaseRequest true "Scope, adjustment, and effective date"
// @Success 200 {object} rateagreementservice.RateIncreasePlan
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/rate-increase/apply/ [post]
func (h *Handler) applyRateIncrease(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(rateagreementservice.RateIncreaseRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(authCtx)

	plan, err := h.service.ApplyRateIncrease(c.Request.Context(), req, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, plan)
}

// @Summary Duplicate a rate agreement
// @Description Copies the whole contract — lanes, breaks, accessorials, fuel terms — as a fresh draft with none of the original's history or approvals. The renewal workflow starts here.
// @ID duplicateRateAgreement
// @Tags Rate Agreements
// @Accept json
// @Produce json
// @Param rateAgreementID path string true "Rate agreement ID"
// @Param request body rateagreementservice.DuplicateRateAgreementRequest false "Optional code and name for the copy"
// @Success 201 {object} rateagreement.RateAgreement
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-agreements/{rateAgreementID}/duplicate/ [post]
func (h *Handler) duplicate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	agreementID, err := h.agreementID(c)
	if err != nil {
		return
	}

	req := new(rateagreementservice.DuplicateRateAgreementRequest)
	if err = c.ShouldBindJSON(req); err != nil && !errors.Is(err, io.EOF) {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(authCtx)
	req.RateAgreementID = agreementID

	created, err := h.service.Duplicate(c.Request.Context(), req, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) review(action rateagreementservice.ReviewAction) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)

		agreementID, err := h.agreementID(c)
		if err != nil {
			return
		}

		body := new(reviewRequest)
		if err = c.ShouldBindJSON(body); err != nil {
			h.eh.HandleError(c, err)
			return
		}

		updated, err := action(c.Request.Context(), &rateagreementservice.ApprovalActionRequest{
			TenantInfo: tenantOf(authCtx),
			EntityID:   agreementID,
			Comment:    body.Comment,
		})
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		c.JSON(http.StatusOK, updated)
	}
}

// agreementID reads the path parameter, reporting the error to the client and
// telling the caller to stop.
func (h *Handler) agreementID(c *gin.Context) (pulid.ID, error) {
	agreementID, err := pulid.MustParse(c.Param("rateAgreementID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return agreementID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
