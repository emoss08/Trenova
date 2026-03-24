package customerhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *customerservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *customerservice.Service
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
	api := rg.Group("/customers")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:customerID",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpRead),
		h.get,
	)

	// This is un-protected because it is used only by the client, this api is not documented and is not part of the public api.
	api.GET(
		"/:customerID/billing-profile/",
		h.getBillingProfile,
	)

	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:customerID/",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:customerID/",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceCustomer.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:customerID/", h.getOption)
}

// @Summary List customers
// @ID listCustomers
// @Tags Customers
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param fieldFilters query string false "JSON array of field filters"
// @Param filterGroups query string false "JSON array of grouped filters"
// @Param sort query string false "JSON array of sort fields"
// @Param geoFilters query string false "JSON array of geographic filters"
// @Param aggregateFilters query string false "JSON array of aggregate filters"
// @Param includeState query bool false "Include state relationship"
// @Param includeBillingProfile query bool false "Include customer billing profile and document types"
// @Param includeEmailProfile query bool false "Include customer email profile"
// @Success 200 {object} pagination.Response[[]customer.Customer]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*customer.Customer], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListCustomerRequest{
					Filter: req,
					CustomerFilterOptions: repositories.CustomerFilterOptions{
						IncludeState:          helpers.QueryBool(c, "includeState"),
						IncludeBillingProfile: helpers.QueryBool(c, "includeBillingProfile"),
						IncludeEmailProfile:   helpers.QueryBool(c, "includeEmailProfile"),
					},
				},
			)
		},
	)
}

// @Summary Bulk update customer statuses
// @ID bulkUpdateCustomerStatus
// @Tags Customers
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateCustomerStatusRequest true "Bulk status update request"
// @Success 200 {array} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update customer status"),
		)
		return
	}

	req := new(repositories.BulkUpdateCustomerStatusRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	results, err := h.service.BulkUpdateStatus(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// @Summary Get a customer option
// @ID getCustomerOption
// @Tags Customers
// @Produce json
// @Param customerID path string true "Customer ID"
// @Success 200 {object} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/select-options/{customerID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetCustomerByIDRequest{
		ID: customerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List customer options
// @ID listCustomerOptions
// @Tags Customers
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]customer.Customer]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*customer.Customer], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.CustomerSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a customer
// @ID getCustomer
// @Tags Customers
// @Produce json
// @Param customerID path string true "Customer ID"
// @Success 200 {object} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/{customerID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCustomerByIDRequest{
			ID: customerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) getBillingProfile(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetBillingProfile(c.Request.Context(), customerID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a customer
// @ID createCustomer
// @Tags Customers
// @Accept json
// @Produce json
// @Param request body customer.Customer true "Customer payload"
// @Success 201 {object} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(customer.Customer)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Patch a customer
// @ID patchCustomer
// @Tags Customers
// @Accept json
// @Produce json
// @Param customerID path string true "Customer ID"
// @Param request body customer.Customer true "Partial customer payload"
// @Success 200 {object} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/{customerID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetCustomerByIDRequest{
			ID: customerID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updatedEntity, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Update a customer
// @ID updateCustomer
// @Tags Customers
// @Accept json
// @Produce json
// @Param customerID path string true "Customer ID"
// @Param request body customer.Customer true "Customer payload"
// @Success 200 {object} customer.Customer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /customers/{customerID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(customer.Customer)
	entity.ID = customerID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
