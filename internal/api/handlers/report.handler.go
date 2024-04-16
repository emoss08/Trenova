package handlers

import (
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct {
	Server  *api.Server
	Service *services.ReportService
}

func NewReportHandler(s *api.Server) *ReportHandler {
	return &ReportHandler{
		Server:  s,
		Service: services.NewReportService(s),
	}
}

func (h *ReportHandler) GetColumnNames() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Query("tableName")
		if tableName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "query parameter 'tableName' is required",
						Attr:   "tableName",
					},
				},
			})
		}

		columns, count, err := h.Service.GetColumnsByTableName(c.UserContext(), tableName)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results: columns,
			Count:   count,
		})
	}
}
