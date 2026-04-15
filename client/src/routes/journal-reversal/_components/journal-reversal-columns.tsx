import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { journalReversalStatusChoices } from "@/lib/choices";
import type { JournalReversal } from "@/types/journal-reversal";
import type { ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router";

export function getColumns(): ColumnDef<JournalReversal>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <AccountingStatusBadge status={row.original.status} />,
      size: 140,
      minSize: 100,
      maxSize: 160,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: journalReversalStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "originalJournalEntryId",
      header: "Original Journal Entry",
      cell: ({ row }) => (
        <Link
          to={`/accounting/journal-entries/${row.original.originalJournalEntryId}`}
          className="font-mono text-xs text-muted-foreground hover:text-foreground hover:underline"
        >
          {row.original.originalJournalEntryId}
        </Link>
      ),
      size: 220,
      minSize: 180,
      maxSize: 300,
      meta: {
        apiField: "originalJournalEntryId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "reasonCode",
      header: "Reason Code",
      cell: ({ row }) => (
        <span className="font-medium">{row.original.reasonCode}</span>
      ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        apiField: "reasonCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "reasonText",
      header: "Reason",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.reasonText}
          truncateLength={80}
        />
      ),
      size: 250,
      minSize: 200,
      maxSize: 400,
      meta: {
        apiField: "reasonText",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
