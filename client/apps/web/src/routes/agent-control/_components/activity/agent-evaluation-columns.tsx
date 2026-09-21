import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentEvaluationRow } from "@/lib/graphql/agent-evaluations";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { toneVar } from "@/components/kpi/tone";
import {
  EvaluationStatusBadge,
  evaluationStatusChoices,
  TriggerBadge,
  triggerChoices,
} from "./agent-badges";
import { readComparison, summarizeComparison } from "./evaluation-comparison";

export function getEvaluationColumns(t: TranslateFn): ColumnDef<AgentEvaluationRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EvaluationStatusBadge value={row.original.status} t={t} />,
      size: 130,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: evaluationStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "outcome",
      accessorKey: "comparison",
      header: t("Outcome"),
      cell: ({ row }) => <OutcomeCell row={row.original} t={t} />,
      size: 320,
      meta: { label: t("Outcome"), apiField: "comparison", filterable: false, sortable: false },
    },
    {
      accessorKey: "trigger",
      header: t("Original started by"),
      cell: ({ row }) => <TriggerBadge value={row.original.trigger} t={t} />,
      size: 150,
      meta: {
        label: t("Original started by"),
        apiField: "trigger",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: triggerChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "reply",
      header: t("Replay said"),
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.reply || row.original.errorMessage}
          truncateLength={90}
        />
      ),
      size: 320,
      meta: {
        label: t("Replay said"),
        apiField: "reply",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "model",
      header: t("Model"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.model || "—"}</span>,
      size: 200,
      meta: {
        label: t("Model"),
        apiField: "model",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "sourceRunId",
      header: t("Original run"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">{row.original.sourceRunId}</span>
      ),
      size: 220,
      meta: {
        label: t("Original run"),
        apiField: "sourceRunId",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Replayed"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Replayed"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}

/**
 * The counts that matter, and the score when there is one. A replay that
 * regressed is coloured as such at a glance, because that is what the
 * column exists to catch.
 */
function OutcomeCell({ row, t }: { row: AgentEvaluationRow; t: TranslateFn }) {
  if (row.status === "Failed") {
    return (
      <span className="text-xs" style={{ color: toneVar("danger") }}>
        {row.errorMessage || t("The replay did not finish.")}
      </span>
    );
  }
  const comparison = readComparison(row.comparison);
  if (!comparison) {
    return <span className="text-muted-foreground text-xs">{t("Not finished yet")}</span>;
  }

  const bad = comparison.regressed + comparison.repeated > 0;

  return (
    <span className="flex items-center gap-2 text-xs">
      {comparison.score !== null && (
        <span
          className="tabular-nums font-medium"
          style={{ color: toneVar(bad ? "danger" : "success") }}
        >
          {Math.round(comparison.score * 100)}%
        </span>
      )}
      <span className="text-muted-foreground truncate">{summarizeComparison(comparison, t)}</span>
    </span>
  );
}
