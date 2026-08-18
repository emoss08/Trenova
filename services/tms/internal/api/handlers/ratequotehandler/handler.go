package ratequotehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// shipmentQuoteHistoryLimit bounds a shipment's rating history. A shipment is
// re-rated on every stop edit and every fuel price refresh, so the list is
// unbounded in principle and nobody reads past the recent ones.
const shipmentQuoteHistoryLimit = 50

type Params struct {
	fx.In

	Service              *ratequoteservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratequoteservice.Service
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
	resource := permission.ResourceRateQuote.String()

	api := rg.Group("/rate-quotes")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateQuoteID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)

	api.GET(
		"/shipment/:shipmentID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listForShipment,
	)
	api.GET(
		"/shipment/:shipmentID/applied/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.appliedForShipment,
	)
	api.POST(
		"/shipment/:shipmentID/explain/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.explain,
	)

	api.POST("/quote/", h.pm.RequirePermission(resource, permission.OpRead), h.quote)
}

// @Summary List rate quotes
// @ID listRateQuotes
// @Tags Rate Quotes
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratequote.RateQuote]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratequote.RateQuote], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRateQuotesRequest{Filter: req},
			)
		},
	)
}

// @Summary Get a rate quote
// @ID getRateQuote
// @Tags Rate Quotes
// @Produce json
// @Param rateQuoteID path string true "Rate quote ID"
// @Success 200 {object} ratequote.RateQuote
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/{rateQuoteID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	quoteID, err := pulid.MustParse(c.Param("rateQuoteID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), &repositories.GetRateQuoteByIDRequest{
		RateQuoteID: quoteID,
		TenantInfo:  tenantOf(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List a shipment's rating history
// @ID listShipmentRateQuotes
// @Tags Rate Quotes
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {array} ratequote.RateQuote
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/shipment/{shipmentID} [get]
func (h *Handler) listForShipment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := h.shipmentID(c)
	if err != nil {
		return
	}

	quotes, err := h.service.ListForShipment(
		c.Request.Context(),
		&repositories.ListShipmentRateQuotesRequest{
			ShipmentID: shipmentID,
			TenantInfo: tenantOf(authCtx),
			Limit:      shipmentQuoteHistoryLimit,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, quotes)
}

// @Summary Get the quote governing a shipment
// @Description Returns the quote the shipment is currently billed from, including the full trace of every rate considered and why each one lost.
// @ID getAppliedShipmentRateQuote
// @Tags Rate Quotes
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param partyType query string false "Which side to read" Enums(Customer, Carrier)
// @Success 200 {object} ratequote.RateQuote
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/shipment/{shipmentID}/applied [get]
func (h *Handler) appliedForShipment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := h.shipmentID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetAppliedForShipment(
		c.Request.Context(),
		&repositories.GetShipmentRateQuoteRequest{
			ShipmentID: shipmentID,
			PartyType:  partyTypeOf(c),
			TenantInfo: tenantOf(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// ratedShipment names the engine's result in this package so the generated API
// documentation can describe what these endpoints return. The generator only
// resolves types the annotated file imports, which is the whole reason it is
// here rather than referenced directly.
//
//nolint:unused // referenced from the swagger annotations, not from Go code
type ratedShipment = services.RatedShipment

type explainRequest struct {
	PartyType rateagreement.PartyType `json:"partyType"`
	PartyID   pulid.ID                `json:"partyId"`
	// AsOf re-resolves against the terms effective on that date. Omit it to
	// reproduce the shipment's current rate.
	AsOf int64 `json:"asOf"`
}

// @Summary Explain or re-price a shipment
// @Description Re-resolves a saved shipment against the contracts effective on a date and returns the full trace, without changing what the shipment is billed at.
// @ID explainShipmentRate
// @Tags Rate Quotes
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body explainRequest true "Explanation options"
// @Success 200 {object} ratequotehandler.ratedShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/shipment/{shipmentID}/explain [post]
func (h *Handler) explain(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	shipmentID, err := h.shipmentID(c)
	if err != nil {
		return
	}

	body := new(explainRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rated, err := h.service.Explain(c.Request.Context(), &ratequoteservice.ExplainRequest{
		TenantInfo: tenantOf(authCtx),
		ShipmentID: shipmentID,
		PartyType:  body.PartyType,
		PartyID:    body.PartyID,
		AsOf:       body.AsOf,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rated)
}

type quoteRequest struct {
	Shipment  *shipment.Shipment      `json:"shipment"  binding:"required"`
	PartyType rateagreement.PartyType `json:"partyType"`
	PartyID   pulid.ID                `json:"partyId"`
	AsOf      int64                   `json:"asOf"`
	// Persist keeps the quote so it can be cited when the load is booked. A
	// screen re-pricing on every keystroke leaves it off.
	Persist bool `json:"persist"`
}

// @Summary Price a shipment that has not been created
// @Description Rates a hypothetical shipment against the contracts covering its lane, which is what a pre-booking quote is.
// @ID quoteShipment
// @Tags Rate Quotes
// @Accept json
// @Produce json
// @Param request body quoteRequest true "Quote payload"
// @Success 200 {object} ratequotehandler.ratedShipment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-quotes/quote/ [post]
func (h *Handler) quote(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	body := new(quoteRequest)
	if err := c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	rated, err := h.service.Quote(c.Request.Context(), &ratequoteservice.QuoteRequest{
		TenantInfo: tenantOf(authCtx),
		Shipment:   body.Shipment,
		PartyType:  body.PartyType,
		PartyID:    body.PartyID,
		AsOf:       body.AsOf,
		Persist:    body.Persist,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rated)
}

func (h *Handler) shipmentID(c *gin.Context) (pulid.ID, error) {
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return shipmentID, nil
}

// partyTypeOf reads which side of the shipment is being asked about, defaulting
// to the customer because that is the rate almost every screen wants.
func partyTypeOf(c *gin.Context) rateagreement.PartyType {
	if raw := helpers.QueryString(c, "partyType"); raw != "" {
		return rateagreement.PartyType(raw)
	}

	return rateagreement.PartyTypeCustomer
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
