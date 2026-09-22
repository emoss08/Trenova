package tablequeryhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *tablequeryservice.Service
	Catalog      *filtercatalog.Catalog
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *tablequeryservice.Service
	catalog *filtercatalog.Catalog
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, catalog: p.Catalog, eh: p.ErrorHandler}
}

// RegisterRoutes mounts the composer.
//
// There is no permission middleware on these: the resource being narrowed is
// in the path, so the check has to be against that resource rather than
// against a blanket one, and the service does it before it spends anything.
// A middleware here would either be the wrong check or a second one.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/tables")
	api.GET("/", h.catalogue)
	api.POST("/:resource/compose/", h.compose)
}

type composeRequest struct {
	Prompt  string                        `json:"prompt"`
	Current tablequeryservice.CurrentView `json:"current"`
}

func (h *Handler) compose(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body composeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActor(authCtx)
	result, err := h.service.Compose(c.Request.Context(), &tablequeryservice.ComposeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		Actor:    &actor,
		Resource: permission.Resource(c.Param("resource")),
		Prompt:   body.Prompt,
		Current:  body.Current,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type catalogueField struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Note      string   `json:"note,omitempty"`
	Values    []string `json:"values,omitempty"`
	Sortable  bool     `json:"sortable"`
	Operators []string `json:"operators"`
}

type catalogueResource struct {
	Resource string           `json:"resource"`
	Entity   string           `json:"entity"`
	Summary  string           `json:"summary,omitempty"`
	Fields   []catalogueField `json:"fields"`
}

// catalogue says which tables can be asked in words, and in which terms.
//
// The client needs it to know whether to show the Ask input at all: offering
// it on a table nothing can narrow is a promise the request would refuse.
func (h *Handler) catalogue(c *gin.Context) {
	resources := h.catalog.Resources()
	out := make([]catalogueResource, 0, len(resources))

	for _, resource := range resources {
		fields := make([]catalogueField, 0, len(resource.Fields))
		for _, field := range resource.Fields {
			operators := filtercatalog.Operators(field.Kind)
			names := make([]string, 0, len(operators))
			for _, operator := range operators {
				names = append(names, string(operator))
			}
			fields = append(fields, catalogueField{
				Name:      field.Name,
				Kind:      string(field.Kind),
				Note:      field.Note,
				Values:    field.Values,
				Sortable:  field.Sortable,
				Operators: names,
			})
		}
		out = append(out, catalogueResource{
			Resource: resource.Resource.String(),
			Entity:   resource.Entity,
			Summary:  resource.Summary,
			Fields:   fields,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": out, "count": len(out)})
}

func requestActor(authCtx *authctx.AuthContext) serviceports.RequestActor {
	return serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}
