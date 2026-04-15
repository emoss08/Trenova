import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@/components/accounting/amount-display";
import { manualJournalStatusChoices } from "@/lib/choices";
import type { ManualJournal } from "@/types/manual-journal";
import type { ColumnDef } from "@tanstack/react-table";

export function getManualJournalColumns(): ColumnDef<ManualJournal>[] {
  return [
    {
      accessorKey: "requestNumber",
      header: "Request #",
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.requestNumber}</span>
      ),
      meta: {
        apiField: "requestNumber",
        label: "Request #",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 160,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <AccountingStatusBadge status={row.original.status} />,
      meta: {
        apiField: "status",
        label: "Status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: manualJournalStatusChoices,
      },
      size: 160,
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="line-clamp-1 text-xs">{row.original.description}</span>
      ),
      meta: {
        apiField: "description",
        label: "Description",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
      size: 300,
    },
    {
      accessorKey: "accountingDate",
      header: "Accounting Date",
      cell: ({ row }) => (
        <span className="text-xs">
          {new Date(row.original.accountingDate * 1000).toLocaleDateString()}
        </span>
      ),
      meta: {
        apiField: "accountingDate",
        label: "Accounting Date",
        sortable: true,
      },
      size: 140,
    },
    {
      accessorKey: "totalDebit",
      header: "Total Debit",
      cell: ({ row }) => (
        <AmountDisplay value={row.original.totalDebit} className="text-xs" />
      ),
      meta: {
        apiField: "totalDebit",
        label: "Total Debit",
        sortable: true,
      },
      size: 130,
    },
    {
      accessorKey: "totalCredit",
      header: "Total Credit",
      cell: ({ row }) => (
        <AmountDisplay value={row.original.totalCredit} className="text-xs" />
      ),
      meta: {
        apiField: "totalCredit",
        label: "Total Credit",
        sortable: true,
      },
      size: 130,
    },
  ];
}
