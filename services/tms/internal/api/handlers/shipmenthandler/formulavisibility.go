package shipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentformula"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) respondShipment(c *gin.Context, status int, entity *shipment.Shipment) {
	formulavisibility.Respond(
		c,
		h.formulas,
		status,
		entity,
		shipmentformula.Withhold,
	)
}

func (h *Handler) respondShipments(c *gin.Context, entities []*shipment.Shipment) {
	formulavisibility.Respond(
		c,
		h.formulas,
		http.StatusOK,
		entities,
		shipmentformula.WithholdAll,
	)
}

func (h *Handler) visibleShipmentPage(
	c *gin.Context,
	result *pagination.CursorListResult[*shipment.Shipment],
) *pagination.CursorListResult[*shipment.Shipment] {
	visible := *result
	visible.Items = formulavisibility.Apply(
		c,
		h.formulas,
		result.Items,
		shipmentformula.WithholdAll,
	)

	return &visible
}
