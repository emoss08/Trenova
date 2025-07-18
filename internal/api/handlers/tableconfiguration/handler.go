package tableconfiguration

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	tableconfigurationdomain "github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tableconfiguration"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/paginationutils/limitoffsetpagination"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Handler struct {
	ts *tableconfiguration.Service
	eh *validator.ErrorHandler
}

type HandlerParams struct {
	fx.In

	TableConfigurationService *tableconfiguration.Service
	ErrorHandler              *validator.ErrorHandler
}

func NewHandler(p HandlerParams) *Handler {
	return &Handler{ts: p.TableConfigurationService, eh: p.ErrorHandler}
}

func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/table-configurations")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.list},
		middleware.PerMinute(120), // 120 reads per minute
	)...)

	api.Get("/me/:resource", rl.WithRateLimit(
		[]fiber.Handler{h.listUserConfigurations},
		middleware.PerSecond(5), // 5 reads per second
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.create},
		middleware.PerSecond(3), // 3 writes per second
	)...)

	api.Post("/share", rl.WithRateLimit(
		[]fiber.Handler{h.share},
		middleware.PerSecond(3), // 3 writes per second
	)...)

	api.Get(":resource", rl.WithRateLimit(
		[]fiber.Handler{h.getDefaultOrLatestConfiguration},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Get("/public/:resource", rl.WithRateLimit(
		[]fiber.Handler{h.listPublicConfigurations},
		middleware.PerSecond(10), // 10 reads per second
	)...)

	api.Put(":configID", rl.WithRateLimit(
		[]fiber.Handler{h.update},
		middleware.PerSecond(3), // 3 writes per second
	)...)

	api.Delete(":configID", rl.WithRateLimit(
		[]fiber.Handler{h.delete},
		middleware.PerSecond(5), // 5 writes per second
	)...)
}

func (h *Handler) list(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	opts := &ports.FilterQueryOptions{
		OrgID:  reqCtx.OrgID,
		BuID:   reqCtx.BuID,
		UserID: reqCtx.UserID,
		Query:  c.Query("query"),
	}

	result, err := h.ts.List(c.UserContext(),
		&repositories.TableConfigurationFilters{
			Base:           opts,
			Search:         c.Query("search"),
			IncludeShares:  c.QueryBool("include_shares", false),
			IncludeCreator: c.QueryBool("include_creator", false),
		})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) listUserConfigurations(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*tableconfigurationdomain.Configuration], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		resource := fc.Params("resource")

		return h.ts.ListUserConfigurations(
			fc.UserContext(),
			&repositories.ListUserConfigurationRequest{
				Resource: resource,
				Filter:   filter,
			},
		)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) listPublicConfigurations(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	handler := func(fc *fiber.Ctx, filter *ports.LimitOffsetQueryOptions) (*ports.ListResult[*tableconfigurationdomain.Configuration], error) {
		if err = fc.QueryParser(filter); err != nil {
			return nil, h.eh.HandleError(fc, err)
		}

		resource := fc.Params("resource")

		return h.ts.ListPublicConfigurations(
			fc.UserContext(),
			&repositories.TableConfigurationFilters{
				Base: &ports.FilterQueryOptions{
					OrgID:  reqCtx.OrgID,
					BuID:   reqCtx.BuID,
					UserID: reqCtx.UserID,
				},
				Resource: resource,
				Search:   c.Query("search"),
			},
		)
	}

	return limitoffsetpagination.HandlePaginatedRequest(c, h.eh, reqCtx, handler)
}

func (h *Handler) create(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	config := new(tableconfigurationdomain.Configuration)
	config.OrganizationID = reqCtx.OrgID
	config.BusinessUnitID = reqCtx.BuID
	config.UserID = reqCtx.UserID

	if err = c.BodyParser(config); err != nil {
		return h.eh.HandleError(c, err)
	}

	created, err := h.ts.Create(c.UserContext(), config)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.JSON(created)
}

func (h *Handler) getDefaultOrLatestConfiguration(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	resource := c.Params("resource")

	config, err := h.ts.GetDefaultOrLatestConfiguration(c.UserContext(), resource, reqCtx)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	if config == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "configuration not found"})
	}
	return c.Status(fiber.StatusOK).JSON(config)
}

func (h *Handler) update(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	configID, err := pulid.MustParse(c.Params("configID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	tr := new(tableconfigurationdomain.Configuration)
	tr.ID = configID
	tr.OrganizationID = reqCtx.OrgID
	tr.BusinessUnitID = reqCtx.BuID
	tr.UserID = reqCtx.UserID

	if err = c.BodyParser(tr); err != nil {
		return h.eh.HandleError(c, err)
	}

	updatedConfig, err := h.ts.Update(c.UserContext(), tr)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedConfig)
}

func (h *Handler) delete(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	configID, err := pulid.MustParse(c.Params("configID"))
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.ts.Delete(c.UserContext(), repositories.DeleteUserConfigurationRequest{
		ConfigID: configID,
		UserID:   reqCtx.UserID,
		OrgID:    reqCtx.OrgID,
		BuID:     reqCtx.BuID,
	})
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) share(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	share := new(tableconfigurationdomain.ConfigurationShare)
	share.OrganizationID = reqCtx.OrgID
	share.BusinessUnitID = reqCtx.BuID

	if err = c.BodyParser(share); err != nil {
		return h.eh.HandleError(c, err)
	}

	err = h.ts.ShareConfiguration(c.UserContext(), share, reqCtx.UserID)
	if err != nil {
		return h.eh.HandleError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
