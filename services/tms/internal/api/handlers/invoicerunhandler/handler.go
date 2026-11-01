package invoicerunhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicerunservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *invoicerunservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *invoicerunservice.Service
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
	// Issuing invoices to customers is a separable authority from building the
	// preview, so committing takes Approve rather than Create.

	// Statements are the live half of the same thing: the open period every
	// statement customer is accumulating into, derived on read from the billing
	// queue rather than staged anywhere. They read as invoice runs because that is
	// what they become.
	statements := rg.Group("/billing/statements")
	statements.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpRead),
		h.listStatements,
	)
	statements.GET(
		"/:customerID/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpRead),
		h.getStatement,
	)
	// Billing an open period early issues invoices, so it takes the same authority
	// as committing a run rather than the lighter read.
	statements.POST(
		"/:customerID/bill/",
		h.pm.RequirePermission(permission.ResourceInvoiceRun.String(), permission.OpApprove),
		h.billStatement,
	)
}

type statementExclusionRequest struct {
	BillingQueueItemID pulid.ID `json:"billingQueueItemId"`
	Reason             string   `json:"reason"`
}

type billStatementRequest struct {
	Reason  string                      `json:"reason"`
	Exclude []statementExclusionRequest `json:"exclude"`
}

// @Summary List open statements
// @ID listOpenStatements
// @Tags Invoice Run
// @Produce json
// @Success 200 {object} []services.OpenStatement
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/statements/ [get]
func (h *Handler) listStatements(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	statements, err := h.service.ListOpenStatements(
		c.Request.Context(),
		&invoicerunports.ListOpenStatementsRequest{TenantInfo: tenantInfoFrom(authCtx)},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, gin.H{"results": statements, "count": len(statements)})
}

// @Summary Get one customer's open statement
// @ID getOpenStatement
// @Tags Invoice Run
// @Produce json
// @Param customerID path string true "Customer ID"
// @Success 200 {object} services.OpenStatement
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/statements/{customerID}/ [get]
func (h *Handler) getStatement(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	statement, err := h.service.GetOpenStatement(
		c.Request.Context(),
		&invoicerunports.ListOpenStatementsRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			CustomerID: customerID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, statement)
}

// @Summary Bill an open statement before its cycle closes
// @ID billStatementNow
// @Tags Invoice Run
// @Accept json
// @Produce json
// @Param customerID path string true "Customer ID"
// @Param request body billStatementRequest true "Reason for billing early"
// @Success 200 {object} services.CommitInvoiceRunResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing/statements/{customerID}/bill/ [post]
func (h *Handler) billStatement(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req billStatementRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"request",
			errortypes.ErrInvalid,
			"Invalid request body",
		))
		return
	}

	exclusions := make([]invoicerunports.StatementExclusion, 0, len(req.Exclude))
	for _, exclusion := range req.Exclude {
		exclusions = append(exclusions, invoicerunports.StatementExclusion{
			BillingQueueItemID: exclusion.BillingQueueItemID,
			Reason:             exclusion.Reason,
		})
	}

	result, err := h.service.BillStatementNow(
		c.Request.Context(),
		&invoicerunports.BillStatementNowRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			CustomerID: customerID,
			Reason:     req.Reason,
			Exclude:    exclusions,
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func tenantInfoFrom(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}
