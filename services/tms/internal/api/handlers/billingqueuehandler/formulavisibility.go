package billingqueuehandler

import (
	"github.com/emoss08/trenova/internal/api/formulavisibility"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

func (h *Handler) respondItem(c *gin.Context, status int, item *billingqueue.BillingQueueItem) {
	formulavisibility.Respond(
		c,
		h.formulas,
		status,
		item,
		(*billingqueue.BillingQueueItem).WithoutFormulaContent,
	)
}

func (h *Handler) visibleItemPage(
	c *gin.Context,
	result *pagination.ListResult[*billingqueue.BillingQueueItem],
) *pagination.ListResult[*billingqueue.BillingQueueItem] {
	visible := *result
	visible.Items = formulavisibility.Apply(
		c,
		h.formulas,
		result.Items,
		billingqueue.WithoutFormulaContent,
	)

	return &visible
}
