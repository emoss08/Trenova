import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { manualJournalStatusChoices } from "@/lib/choices";
import type { ManualJournalRow } from "@/lib/graphql/manual-journal-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatUnixDate } from "@trenova/shared/lib/date";

export function getManualJournalColumns(t: TranslateFn): ColumnDef<ManualJournalRow>[] {
  return [
    {
      accessorKey: "requestNumber",
      header: t("Request #"),
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.requestNumber}</span>
      ),
      meta: {
        apiField: "requestNumber",
        label: t("Request #"),
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 160,
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <AccountingStatusBadge status={row.original.status} />,
      meta: {
        apiField: "status",
        label: t("Status"),
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: manualJournalStatusChoices,
      },
      size: 160,
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <span className="line-clamp-1 text-xs">{t(row.original.description)}</span>
      ),
      meta: {
        apiField: "description",
        label: t("Description"),
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 300,
    },
    {
      accessorKey: "accountingDate",
      header: t("Accounting Date"),
      cell: ({ row }) => (
        <span className="text-xs">{formatUnixDate(row.original.accountingDate)}</span>
      ),
      meta: {
        apiField: "accountingDate",
        label: t("Accounting Date"),
        sortable: true,
      },
      size: 140,
    },
    {
      accessorKey: "totalDebit",
      header: t("Total Debit"),
      cell: ({ row }) => <AmountDisplay value={row.original.totalDebit} className="text-xs" />,
      meta: {
        apiField: "totalDebit",
        label: t("Total Debit"),
        sortable: true,
      },
      size: 130,
    },
    {
      accessorKey: "totalCredit",
      header: t("Total Credit"),
      cell: ({ row }) => <AmountDisplay value={row.original.totalCredit} className="text-xs" />,
      meta: {
        apiField: "totalCredit",
        label: t("Total Credit"),
        sortable: true,
      },
      size: 130,
    },
  ];
}
