package ratesimulationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ratesimulationservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ratesimulationservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratesimulationservice.Service
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
	resource := permission.ResourceRateSimulation.String()

	api := rg.Group("/rate-simulations")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateSimulationID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.GET(
		"/:rateSimulationID/results/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listResults,
	)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
}

// @Summary List rate simulations
// @ID listRateSimulations
// @Tags Rate Simulations
// @Produce json
// @Param rateAgreementId query string false "Narrow to one agreement's simulations"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratesimulation.RateSimulation]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-simulations/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	var agreementID *pulid.ID
	if raw := helpers.QueryString(c, "rateAgreementId"); raw != "" {
		parsed, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		agreementID = &parsed
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratesimulation.RateSimulation], error) {
			return h.service.List(c.Request.Context(), &repositories.ListRateSimulationsRequest{
				Filter:          req,
				RateAgreementID: agreementID,
			})
		},
	)
}

// @Summary Get a rate simulation
// @ID getRateSimulation
// @Tags Rate Simulations
// @Produce json
// @Param rateSimulationID path string true "Rate simulation ID"
// @Success 200 {object} ratesimulation.RateSimulation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-simulations/{rateSimulationID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	simulationID, err := h.simulationID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetRateSimulationByIDRequest{
			RateSimulationID: simulationID,
			TenantInfo:       tenantOf(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List a simulation's per-shipment results
// @Description Returns what each replayed shipment was billed and what the simulated contract would have charged, largest increases first.
// @ID listRateSimulationResults
// @Tags Rate Simulations
// @Produce json
// @Param rateSimulationID path string true "Rate simulation ID"
// @Param changedOnly query bool false "Hide the shipments the change did not move"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratesimulation.RateSimulationResult]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-simulations/{rateSimulationID}/results [get]
func (h *Handler) listResults(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	simulationID, err := h.simulationID(c)
	if err != nil {
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratesimulation.RateSimulationResult], error) {
			return h.service.ListResults(
				c.Request.Context(),
				&repositories.ListRateSimulationResultsRequest{
					RateSimulationID: simulationID,
					TenantInfo:       tenantOf(authCtx),
					Filter:           req,
					ChangedOnly:      helpers.QueryBool(c, "changedOnly", false),
				},
			)
		},
	)
}

type createRequest struct {
	RateAgreementID pulid.ID                `json:"rateAgreementId" binding:"required"`
	Name            string                  `json:"name"            binding:"required"`
	Description     string                  `json:"description"`
	PartyType       rateagreement.PartyType `json:"partyType"`
	SampleFrom      int64                   `json:"sampleFrom"      binding:"required"`
	SampleTo        int64                   `json:"sampleTo"        binding:"required"`
	SampleLimit     int                     `json:"sampleLimit"`
}

// @Summary Run a rate simulation
// @Description Records a simulation to be replayed against historical shipments. The run happens in the background; poll the simulation for its status.
// @ID createRateSimulation
// @Tags Rate Simulations
// @Accept json
// @Produce json
// @Param request body createRequest true "Simulation payload"
// @Success 201 {object} ratesimulation.RateSimulation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-simulations/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	body := new(createRequest)
	if err := c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := &ratesimulation.RateSimulation{
		OrganizationID:  authCtx.OrganizationID,
		BusinessUnitID:  authCtx.BusinessUnitID,
		RateAgreementID: body.RateAgreementID,
		Name:            body.Name,
		Description:     body.Description,
		PartyType:       body.PartyType,
		SampleFrom:      body.SampleFrom,
		SampleTo:        body.SampleTo,
		SampleLimit:     body.SampleLimit,
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) simulationID(c *gin.Context) (pulid.ID, error) {
	simulationID, err := pulid.MustParse(c.Param("rateSimulationID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return simulationID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
