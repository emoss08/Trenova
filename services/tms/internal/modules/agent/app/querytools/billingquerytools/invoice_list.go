package billingquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

type invoiceRow struct {
	ID               string                 `json:"id"`
	Number           string                 `json:"number"`
	Status           string                 `json:"status"`
	SettlementStatus string                 `json:"settlementStatus"`
	DisputeStatus    string                 `json:"disputeStatus,omitempty"`
	BillType         string                 `json:"billType"`
	BillTo           string                 `json:"billTo,omitempty"`
	ProNumber        string                 `json:"proNumber,omitempty"`
	TotalAmount      string                 `json:"totalAmount,omitempty"`
	Currency         string                 `json:"currency,omitempty"`
	InvoiceDate      querytool.OptionalDate `json:"invoiceDate"`
	// A receivable with no due date is not a receivable that is current; it is
	// one nobody gave terms to. Omitting the key made it look like the former.
	DueDate querytool.OptionalDate `json:"dueDate"`
}

func NewListInvoicesTool(
	repo invoiceports.InvoiceRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return listtool.New(&listtool.Spec{
		Name:         "invoice_list",
		Row:          invoiceRow{},
		EntityPlural: "invoices",
		Summary: "List invoices narrowed by customer, status, whether they are paid or " +
			"disputed, the amount, or the invoice and due dates. It answers what is unpaid, " +
			"overdue or in dispute.",
		SearchTerms: []string{"last invoice", "latest invoice"},
		Resource:    permission.ResourceInvoice,
		Config:      listquery.Describe((*invoice.Invoice)(nil), invoicedb.InvoicesSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   "status",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Draft", "Posted", "Voided"},
				Note:   "where the invoice is in its own lifecycle, not whether it is paid",
			},
			{
				Name:   "settlementStatus",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Unpaid", "PartiallyPaid", "Paid"},
				Note:   "this is the one to use for \"unpaid\" or \"outstanding\"",
			},
			{
				Name:   "disputeStatus",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"None", "Disputed"},
			},
			{
				Name:   "billType",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"Invoice", "CreditMemo", "DebitMemo"},
				Note:   "CreditMemo and DebitMemo are memos; apply a posted CreditMemo with receivable_apply_credit_memo",
			},
			{Name: "number", Kind: filtercatalog.KindText, Sortable: true},
			{
				Name: "billToName",
				Kind: filtercatalog.KindText,
				Note: "who the invoice is billed to",
			},
			{Name: "shipmentProNumber", Kind: filtercatalog.KindText},
			{
				Name:     "dueDate",
				Kind:     filtercatalog.KindDate,
				Sortable: true,
				Note:     "overdue is a due date before today with settlementStatus not Paid",
			},
			{Name: "invoiceDate", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: "totalAmount", Kind: filtercatalog.KindNumber, Sortable: true},
			{Name: "createdAt", Kind: filtercatalog.KindDate, Sortable: true},
		},
		Access: fieldgate.NewAccess(permissions),
		FetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			_ pagination.CursorInfo,
			gate *fieldgate.Gate,
		) (listtool.Page, error) {
			result, err := repo.List(ctx, &invoiceports.ListInvoicesRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}

			showTotal := gate.Show("totalAmount", "totalAmount")

			return listtool.OffsetPage(result.Items, func(item *invoice.Invoice) any {
				row := invoiceRow{
					ID:               item.ID.String(),
					Number:           item.Number,
					Status:           string(item.Status),
					SettlementStatus: string(item.SettlementStatus),
					DisputeStatus:    string(item.DisputeStatus),
					BillType:         string(item.BillType),
					BillTo:           item.BillToName,
					ProNumber:        item.ShipmentProNumber,
					Currency:         item.CurrencyCode,
					InvoiceDate:      querytool.RecordedDate(item.InvoiceDate),
					DueDate:          querytool.PointerDate(item.DueDate),
				}
				if showTotal {
					row.TotalAmount = item.TotalAmount.StringFixed(2)
				}

				return row
			}), nil
		},
	})
}
