package shipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) respondShipment(c *gin.Context, status int, entity *shipment.Shipment) {
	formulavisibility.Respond(
		c,
		h.pm,
		h.eh,
		status,
		entity,
		(*shipment.Shipment).WithoutFormulaContent,
	)
}

func (h *Handler) respondShipments(c *gin.Context, entities []*shipment.Shipment) {
	formulavisibility.Respond(
		c,
		h.pm,
		h.eh,
		http.StatusOK,
		entities,
		shipment.WithoutFormulaContent,
	)
}

func (h *Handler) visibleShipmentPage(
	c *gin.Context,
	result *pagination.CursorListResult[*shipment.Shipment],
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	items, err := formulavisibility.Apply(c, h.pm, result.Items, shipment.WithoutFormulaContent)
	if err != nil {
		return nil, err
	}

	visible := *result
	visible.Items = items

	return &visible, nil
}
