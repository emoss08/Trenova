import { CopyIconButton } from "@/components/copy-icon-button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { isAbsoluteUrl } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ExternalLinkIcon } from "lucide-react";

type Traced = { traceId: string | null; traceUrl?: string | null };

/**
 * The Trace column every AI Control table of agent work shares: the id to
 * copy, the link when a tracing backend is configured, and a filter on the
 * exact id, so a trace pasted from the backend finds its rows.
 */
export function traceColumn<TData extends Traced>(t: TranslateFn): ColumnDef<TData> {
  return {
    accessorKey: "traceId",
    header: t("Trace"),
    cell: ({ row }) => (
      <TraceLink traceId={row.original.traceId} traceUrl={row.original.traceUrl} />
    ),
    size: 150,
    meta: {
      label: t("Trace"),
      apiField: "traceId",
      filterable: true,
      sortable: false,
      filterType: "text",
      defaultFilterOperator: "eq",
    },
  } as ColumnDef<TData>;
}

const SHORT_TRACE_LENGTH = 8;

type TraceLinkProps = {
  traceId: string | null | undefined;
  /** Where the trace opens in the tracing backend; absent when none is configured. */
  traceUrl?: string | null;
};

/**
 * The trace a piece of agent work was recorded in: its id to copy, and a
 * link into the tracing backend when one is configured. A record from before
 * traces were kept has neither.
 */
export function TraceLink({ traceId, traceUrl }: TraceLinkProps) {
  const t = useT();

  if (!traceId) {
    return <span className="text-foreground-subtle">—</span>;
  }

  const short = traceId.slice(0, SHORT_TRACE_LENGTH);
  return (
    <span className="inline-flex min-w-0 items-center gap-1">
      {traceUrl && isAbsoluteUrl(traceUrl) ? (
        <a
          href={traceUrl}
          target="_blank"
          rel="noopener noreferrer"
          title={t("Open trace {0}", traceId)}
          className="text-brand ui-focus-ring inline-flex items-center gap-1 rounded-sm font-mono text-xs underline-offset-4 hover:underline"
        >
          {short}
          <ExternalLinkIcon aria-hidden className="size-3" />
        </a>
      ) : (
        <span title={traceId} className="text-foreground-muted font-mono text-xs">
          {short}
        </span>
      )}
      <CopyIconButton value={traceId} label={t("Copy trace id")} size="icon-xxs" />
    </span>
  );
}
