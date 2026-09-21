import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AgingBadge } from "@/components/accounting/aging-buckets";
import {
  InvoiceEdiSendStatusBadge,
  PlainInvoiceDisputeBadge,
  PlainInvoiceScopeBadge,
  PlainInvoiceSplitBadge,
  PlainInvoiceStatusBadge,
  PlainSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import {
  billTypeChoices,
  invoiceEdiSendStatusChoices,
  invoiceScopeChoices,
  invoiceStatusChoices,
} from "@/lib/choices";
import type { InvoiceRegisterRow } from "@/lib/graphql/invoice-table";
import { invoicePanelPath } from "@/lib/invoice-links";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

const settlementStatusChoices = [
  { value: "Unpaid", label: "Unpaid" },
  { value: "PartiallyPaid", label: "Partially paid" },
  { value: "Paid", label: "Paid" },
];

const disputeStatusChoices = [
  { value: "None", label: "None" },
  { value: "Disputed", label: "Disputed" },
];

function formatDate(unix: number | null | undefined): string {
  return unix ? formatUnixDateMedium(unix, { fallback: "—" }) : "—";
}

/** The shipper is only worth a column when the invoice bills someone else's freight. */
function shipperName(row: InvoiceRegisterRow): string | null {
  if (!row.shipperCustomerId || row.shipperCustomerId === row.customerId) return null;
  return row.shipperCustomer?.name ?? null;
}

export function getColumns(t: TranslateFn): ColumnDef<InvoiceRegisterRow>[] {
  return [
    {
      id: "number",
      accessorKey: "number",
      header: t("Invoice #"),
      cell: ({ row }) => (
        <Link
          to={invoicePanelPath(row.original.id)}
          className="font-mono font-medium hover:underline"
          onClick={(event) => event.stopPropagation()}
        >
          {row.original.number}
        </Link>
      ),
      meta: {
        apiField: "number",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "billType",
      accessorKey: "billType",
      header: t("Type"),
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">{t(row.original.billType)}</span>
      ),
      meta: {
        apiField: "billType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: billTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <PlainInvoiceStatusBadge status={row.original.status} />,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: invoiceStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "billTo",
      accessorKey: "billToName",
      header: t("Bill To"),
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="truncate font-medium">{row.original.billToName}</span>
          {row.original.billToCode ? (
            <span className="text-muted-foreground text-2xs">{row.original.billToCode}</span>
          ) : null}
        </div>
      ),
      meta: {
        apiField: "billToName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "shipper",
      header: t("Shipper"),
      cell: ({ row }) => {
        const name = shipperName(row.original);
        return name ? (
          <span>{name}</span>
        ) : (
          <span className="text-muted-foreground text-xs">—</span>
        );
      },
      meta: {
        apiField: "shipperCustomerId",
        filterable: false,
        sortable: false,
        filterType: "text",
      },
    },
    {
      id: "invoiceDate",
      accessorKey: "invoiceDate",
      header: t("Invoice date"),
      cell: ({ row }) => <span>{formatDate(row.original.invoiceDate)}</span>,
      meta: {
        apiField: "invoiceDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      id: "dueDate",
      accessorKey: "dueDate",
      header: t("Due date"),
      cell: ({ row }) => <span>{formatDate(row.original.dueDate)}</span>,
      meta: {
        apiField: "dueDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      id: "totalAmount",
      accessorKey: "totalAmount",
      header: t("Total"),
      cell: ({ row }) => (
        <span className="block text-right font-medium tabular-nums">
          {formatCurrency(Number(row.original.totalAmount ?? 0), row.original.currencyCode)}
        </span>
      ),
      meta: {
        apiField: "totalAmount",
        filterable: true,
        sortable: true,
        filterType: "number",
        align: "right",
      },
    },
    {
      id: "openBalance",
      accessorKey: "openBalance",
      header: t("Open"),
      cell: ({ row }) => {
        const open = Number(row.original.openBalance ?? 0);
        return (
          <span
            className={
              open > 0
                ? "block text-right text-xs font-semibold tabular-nums"
                : "text-muted-foreground block text-right text-xs tabular-nums"
            }
          >
            {formatCurrency(open, row.original.currencyCode)}
          </span>
        );
      },
      meta: {
        apiField: "balanceDueMinor",
        filterable: true,
        sortable: true,
        filterType: "number",
        align: "right",
      },
    },
    {
      id: "daysPastDue",
      accessorKey: "daysPastDue",
      header: t("Past due"),
      cell: ({ row }) =>
        row.original.daysPastDue !== null && row.original.daysPastDue !== undefined ? (
          <AgingBadge daysPastDue={row.original.daysPastDue} />
        ) : (
          <span className="text-muted-foreground text-xs">—</span>
        ),
      // Days past due is computed on read; the register sorts and filters it
      // through the due date it derives from.
      meta: {
        apiField: "dueDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      id: "settlementStatus",
      accessorKey: "settlementStatus",
      header: t("Settlement"),
      cell: ({ row }) => <PlainSettlementStatusBadge status={row.original.settlementStatus} />,
      meta: {
        apiField: "settlementStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: settlementStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "disputeStatus",
      accessorKey: "disputeStatus",
      header: t("Dispute"),
      cell: ({ row }) =>
        row.original.disputeStatus === "Disputed" ? (
          <PlainInvoiceDisputeBadge disputeStatus="Disputed" />
        ) : (
          <span className="text-muted-foreground text-xs">—</span>
        ),
      meta: {
        apiField: "disputeStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: disputeStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "ediSendStatus",
      accessorKey: "ediSendStatus",
      header: t("EDI"),
      cell: ({ row }) =>
        row.original.ediSendStatus === "NotSent" ? (
          <span className="text-muted-foreground text-xs">—</span>
        ) : (
          <InvoiceEdiSendStatusBadge status={row.original.ediSendStatus} />
        ),
      meta: {
        apiField: "ediSendStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: invoiceEdiSendStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "scope",
      accessorKey: "scope",
      header: t("Scope"),
      cell: ({ row }) => <PlainInvoiceScopeBadge scope={row.original.scope} />,
      meta: {
        apiField: "scope",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: invoiceScopeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "split",
      accessorKey: "isSplitBill",
      header: t("Split"),
      cell: ({ row }) =>
        row.original.isSplitBill ? (
          <PlainInvoiceSplitBadge isSplitBill />
        ) : (
          <span className="text-muted-foreground text-xs">—</span>
        ),
      meta: {
        apiField: "isSplitBill",
        filterable: true,
        sortable: false,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
  ];
}
