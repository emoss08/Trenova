import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { journalReversalStatusChoices } from "@/lib/choices";
import type { JournalReversalRow } from "@/lib/graphql/journal-reversal-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

export function getColumns(t: TranslateFn): ColumnDef<JournalReversalRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
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
      header: t("Original Journal Entry"),
      cell: ({ row }) => (
        <Link
          to={`/accounting/journal-entries/${row.original.originalJournalEntryId}`}
          className="text-muted-foreground hover:text-foreground font-mono text-xs hover:underline"
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
      header: t("Reason Code"),
      cell: ({ row }) => <span className="font-medium">{row.original.reasonCode}</span>,
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
      header: t("Reason"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.reasonText} truncateLength={80} />
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
      header: t("Created At"),
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
