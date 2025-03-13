package document

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/ctx"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	DocumentService services.DocumentService
	ErrorHandler    *validator.ErrorHandler
}

type Handler struct {
	ds services.DocumentService
	eh *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ds: p.DocumentService, eh: p.ErrorHandler}
}

func (h Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/documents")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	// api.Get("/select-options/", rl.WithRateLimit(
	// 	[]fiber.Handler{h.selectOptions},
	// 	middleware.PerMinute(120), // 120 reads per minute
	// )...)

	// api.Post("/", rl.WithRateLimit(
	// 	[]fiber.Handler{h.create},
	// 	middleware.PerMinute(60), // 60 writes per minute
	// )...)

	// api.Get("/:fleetCodeID/", rl.WithRateLimit(
	// 	[]fiber.Handler{h.get},
	// 	middleware.PerMinute(60), // 60 reads per minute
	// )...)

	// api.Put("/:fleetCodeID/", rl.WithRateLimit(
	// 	[]fiber.Handler{h.update},
	// 	middleware.PerMinute(60), // 60 writes per minute
	// )...)
}

func (h Handler) list(c *fiber.Ctx) error {
	reqCtx, err := ctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*document.Document], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		statuses := []document.DocumentStatus{}
		if status := fc.Query("status"); status != "" {
			statuses = append(statuses, document.DocumentStatus(status))
		}

		tags := []string{}
		if tag := fc.Query("tag"); tag != "" {
			tags = append(tags, tag)
		}

		return h.ds.List(fc.UserContext(), &repositories.ListDocumentsRequest{
			Filter:              filter,
			ResourceType:        permission.Resource(fc.Query("resourceType")),
			DocumentType:        document.DocumentType(fc.Params("documentType")),
			ResourceID:          pulid.Must(fc.Query("resourceID")),
			Statuses:            statuses,
			Tags:                tags,
			SortBy:              fc.Query("sortBy"),
			SortDir:             fc.Query("sortDir"),
			ExpirationDateStart: intutils.SafeInt64PtrOrNil(fc.QueryInt("expirationDateStart")),
			ExpirationDateEnd:   intutils.SafeInt64PtrOrNil(fc.QueryInt("expirationDateEnd")),
			CreatedAtStart:      intutils.SafeInt64PtrOrNil(fc.QueryInt("createdAtStart")),
			CreatedAtEnd:        intutils.SafeInt64PtrOrNil(fc.QueryInt("createdAtEnd")),
			DocumentRequest: repositories.DocumentRequest{
				ExpandDocumentDetails: fc.QueryBool("expandDocumentDetails"),
			},
		})
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}
