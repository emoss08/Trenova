package factoringsql

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
)

func ChargedBackInvoice() string {
	return invoiceWithItem(factoring.SubmissionItemStatusChargedBack)
}

func LiveInvoice() string {
	return invoiceWithItem(factoring.LiveSubmissionItemStatuses()...)
}

func invoiceWithItem(statuses ...factoring.SubmissionItemStatus) string {
	quoted := make([]string, 0, len(statuses))
	for _, status := range statuses {
		quoted = append(quoted, "'"+string(status)+"'")
	}
	item := factoringcol.SubmissionItemColumns
	inv := invoicecol.InvoiceColumns
	return buncolgen.Expr(
		"EXISTS (SELECT 1 FROM "+factoringcol.SubmissionItemTable.Name+
			" AS "+factoringcol.SubmissionItemTable.Alias+
			" WHERE {0} = {1} AND {2} = {3} AND {4} = {5} AND {6} IN ("+
			strings.Join(quoted, ", ")+"))",
		item.OrganizationID,
		inv.OrganizationID,
		item.BusinessUnitID,
		inv.BusinessUnitID,
		item.InvoiceID,
		inv.ID,
		item.Status,
	)
}
