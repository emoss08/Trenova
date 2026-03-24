package dothazmatreferencehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dothazmatreferenceservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *dothazmatreferenceservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *dothazmatreferenceservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/dot-hazmat-references")

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:dotHazmatReferenceID", h.getOption)
}

// @Summary List DOT hazmat reference options
// @ID listDotHazmatReferenceOptions
// @Tags DOT Hazmat References
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]dothazmatreference.DotHazmatReference]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /dot-hazmat-references/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get a DOT hazmat reference option
// @ID getDotHazmatReferenceOption
// @Tags DOT Hazmat References
// @Produce json
// @Param dotHazmatReferenceID path string true "DOT hazmat reference ID"
// @Success 200 {object} dothazmatreference.DotHazmatReference
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /dot-hazmat-references/select-options/{dotHazmatReferenceID} [get]
func (h *Handler) getOption(c *gin.Context) {
	dotHazmatReferenceID, err := pulid.MustParse(c.Param("dotHazmatReferenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetDotHazmatReferenceByIDRequest{
		DotHazmatReferenceID: dotHazmatReferenceID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
