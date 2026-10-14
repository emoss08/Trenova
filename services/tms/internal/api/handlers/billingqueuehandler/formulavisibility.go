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
		h.pm,
		h.eh,
		status,
		item,
		(*billingqueue.BillingQueueItem).WithoutFormulaContent,
	)
}

func (h *Handler) visibleItemPage(
	c *gin.Context,
	result *pagination.ListResult[*billingqueue.BillingQueueItem],
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	items, err := formulavisibility.Apply(c, h.pm, result.Items, billingqueue.WithoutFormulaContent)
	if err != nil {
		return nil, err
	}

	visible := *result
	visible.Items = items

	return &visible, nil
}
