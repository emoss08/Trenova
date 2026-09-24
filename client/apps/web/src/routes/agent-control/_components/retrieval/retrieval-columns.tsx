import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AIRetrievalFailedEntryRow } from "@/lib/graphql/ai-retrieval";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  SOURCE_LABEL,
  failedEntryStatus,
  failedEntryStatusChoices,
  sourceTypeChoices,
} from "./retrieval-model";

function Muted() {
  return <span className="text-muted-foreground">—</span>;
}

/**
 * One line per item that could not be indexed: which item, whether another
 * attempt is coming, and the provider's or the indexer's own words for why.
 * The full error opens in the row's panel.
 */
export function getFailedEntryColumns(t: TranslateFn): ColumnDef<AIRetrievalFailedEntryRow>[] {
  return [
    {
      accessorKey: "sourceType",
      header: t("Item"),
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate">{t(SOURCE_LABEL[row.original.sourceType].label)}</span>
          <span className="text-muted-foreground truncate font-mono text-xs">
            {row.original.sourceId}
          </span>
        </div>
      ),
      size: 220,
      meta: {
        label: t("Source"),
        apiField: "sourceType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: sourceTypeChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const status = failedEntryStatus(row.original);
        return <Badge variant={phaseTone(status.phase)}>{t(status.text)}</Badge>;
      },
      size: 120,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: failedEntryStatusChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "error",
      header: t("Error"),
      cell: ({ row }) => (
        <span className="block truncate" title={row.original.error}>
          {row.original.error}
        </span>
      ),
      size: 420,
      meta: {
        label: t("Error"),
        apiField: "error",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "attempts",
      header: t("Attempts"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.attempts}</span>,
      size: 100,
      meta: {
        label: t("Attempts"),
        apiField: "attempts",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "lastAttemptAt",
      header: t("Last attempt"),
      cell: ({ row }) =>
        row.original.lastAttemptAt ? (
          <HoverCardTimestamp timestamp={row.original.lastAttemptAt} />
        ) : (
          <Muted />
        ),
      size: 170,
      meta: {
        label: t("Last attempt"),
        apiField: "lastAttemptAt",
        sortable: true,
      },
    },
    {
      accessorKey: "nextAttemptAt",
      header: t("Next attempt"),
      cell: ({ row }) =>
        row.original.nextAttemptAt ? (
          <HoverCardTimestamp timestamp={row.original.nextAttemptAt} />
        ) : (
          <Muted />
        ),
      size: 170,
      meta: {
        label: t("Next attempt"),
        apiField: "nextAttemptAt",
        sortable: true,
      },
    },
    {
      accessorKey: "modelKey",
      header: t("Model"),
      cell: ({ row }) => (
        <span className="text-muted-foreground truncate font-mono text-xs">
          {row.original.modelKey}
        </span>
      ),
      size: 260,
      meta: {
        label: t("Model"),
        apiField: "modelKey",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
  ];
}
