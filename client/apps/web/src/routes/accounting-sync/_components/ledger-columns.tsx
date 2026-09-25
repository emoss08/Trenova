import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import { accountingSyncRecordPhase } from "@/lib/accounting-sync";
import type { AccountingSyncLedgerRow } from "@/lib/graphql/accounting-sync-ledger-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ACCOUNTING_SYNC_OBJECT_TYPES } from "./ledger-schemas";

const RECORD_STATUSES = [
  "Queued",
  "AwaitingApproval",
  "InFlight",
  "Retrying",
  "Synced",
  "Blocked",
  "DeadLettered",
  "Skipped",
  "Superseded",
] as const;

export function getLedgerColumns(
  t: TranslateFn,
  labels: AccountingSyncLabels,
): ColumnDef<AccountingSyncLedgerRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={phaseTone(accountingSyncRecordPhase(row.original.status))}>
          {labels.status[row.original.status]}
        </Badge>
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: RECORD_STATUSES.map((value) => ({ value, label: labels.status[value] })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "objectType",
      header: t("Document"),
      cell: ({ row }) => labels.objectType[row.original.objectType],
      size: 160,
      minSize: 140,
      maxSize: 200,
      meta: {
        label: t("Document"),
        apiField: "objectType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ACCOUNTING_SYNC_OBJECT_TYPES.map((value) => ({
          value,
          label: labels.objectType[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "objectNumber",
      header: t("Number"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.objectNumber}</span>,
      size: 180,
      minSize: 140,
      maxSize: 260,
      meta: {
        label: t("Number"),
        apiField: "objectNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "operation",
      header: t("Action"),
      cell: ({ row }) => labels.operation[row.original.operation],
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: t("Action"),
        apiField: "operation",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: (["Create", "Update", "Void"] as const).map((value) => ({
          value,
          label: labels.operation[value],
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "documentDate",
      header: t("Document date"),
      cell: ({ row }) =>
        row.original.documentDate ? formatUnixDate(row.original.documentDate) : "",
      size: 130,
      minSize: 120,
      maxSize: 160,
      meta: {
        label: t("Document date"),
        apiField: "documentDate",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "resolution",
      header: t("What to do"),
      cell: ({ row }) => (
        <span className="text-foreground-muted block truncate">
          {row.original.resolution || row.original.errorMessage}
        </span>
      ),
      size: 320,
      minSize: 200,
      maxSize: 520,
      meta: {
        label: t("What to do"),
        apiField: "resolution",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "attemptCount",
      header: t("Tries"),
      cell: ({ row }) => row.original.attemptCount,
      size: 80,
      minSize: 70,
      maxSize: 100,
      meta: {
        label: t("Tries"),
        apiField: "attemptCount",
        filterable: false,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "externalDocNumber",
      header: t("Number in the books"),
      cell: ({ row }) => row.original.externalDocNumber,
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Number in the books"),
        apiField: "externalDocNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "queuedAt",
      header: t("Queued"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.queuedAt} />,
      size: 170,
      minSize: 150,
      maxSize: 210,
      meta: {
        label: t("Queued"),
        apiField: "queuedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "syncedAt",
      header: t("Synced"),
      cell: ({ row }) =>
        row.original.syncedAt ? <HoverCardTimestamp timestamp={row.original.syncedAt} /> : null,
      size: 170,
      minSize: 150,
      maxSize: 210,
      meta: {
        label: t("Synced"),
        apiField: "syncedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
