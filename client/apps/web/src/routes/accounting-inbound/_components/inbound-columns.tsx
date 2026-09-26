import type { AccountingInboundLabels } from "@/hooks/use-accounting-inbound-labels";
import {
  ACCOUNTING_INBOUND_REASONS,
  ACCOUNTING_INBOUND_STATUSES,
  accountingInboundPhase,
  formatAccountingMinor,
} from "@/lib/accounting-sync";
import type { AccountingInboundRow } from "@/lib/graphql/accounting-inbound-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { ColumnDef } from "@trenova/shared/types/data-table";

function paysLabel(t: TranslateFn, row: AccountingInboundRow): string {
  const numbers = row.lines
    .map((line) => line.objectNumber)
    .filter((number): number is string => number.length > 0);
  if (numbers.length === 0) {
    return t("{0, plural, one {# document} other {# documents}}", row.lines.length);
  }
  if (numbers.length <= 2) {
    return numbers.join(", ");
  }
  return t("{0} and {1} more", numbers.slice(0, 2).join(", "), numbers.length - 2);
}

export function getInboundColumns(
  t: TranslateFn,
  labels: AccountingInboundLabels,
): ColumnDef<AccountingInboundRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={phaseTone(accountingInboundPhase(row.original.status))}>
          {labels.status[row.original.status]}
        </Badge>
      ),
      size: 160,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ACCOUNTING_INBOUND_STATUSES.map((value) => ({
          value,
          label: labels.status[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "kind",
      header: t("Payment"),
      cell: ({ row }) => labels.kind[row.original.kind],
      size: 160,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Payment"),
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: (["CustomerPayment", "BillPayment"] as const).map((value) => ({
          value,
          label: labels.kind[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "externalNumber",
      header: t("Number in the books"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.externalNumber}</span>,
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Number in the books"),
        apiField: "externalNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "partyName",
      header: t("Customer or vendor"),
      cell: ({ row }) => row.original.partyName,
      size: 200,
      minSize: 150,
      maxSize: 280,
      meta: {
        label: t("Customer or vendor"),
        apiField: "partyName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "txnDate",
      header: t("Paid on"),
      cell: ({ row }) => formatUnixDate(row.original.txnDate),
      size: 120,
      minSize: 110,
      maxSize: 150,
      meta: {
        label: t("Paid on"),
        apiField: "txnDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "amountMinor",
      header: t("Amount"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatAccountingMinor(row.original.amountMinor, row.original.currencyCode)}
        </span>
      ),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("Amount"),
        apiField: "amountMinor",
        sortable: true,
      },
    },
    {
      id: "pays",
      header: t("Pays"),
      cell: ({ row }) => <span className="truncate">{paysLabel(t, row.original)}</span>,
      size: 200,
      minSize: 150,
      maxSize: 280,
      meta: { label: t("Pays") },
    },
    {
      accessorKey: "reason",
      header: t("Why it waits"),
      cell: ({ row }) => (row.original.reason ? labels.reason[row.original.reason] : ""),
      size: 220,
      minSize: 160,
      maxSize: 300,
      meta: {
        label: t("Why it waits"),
        apiField: "reason",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ACCOUNTING_INBOUND_REASONS.map((value) => ({
          value,
          label: labels.reason[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
  ];
}
