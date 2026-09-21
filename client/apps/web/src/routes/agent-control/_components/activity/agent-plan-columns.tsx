import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { toneVar } from "@/components/kpi/tone";
import type { AgentPlanRow } from "@/lib/graphql/agent-activity-tables";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { PlanStatusBadge, planStatusChoices } from "./agent-badges";

export function getPlanColumns(t: TranslateFn): ColumnDef<AgentPlanRow>[] {
  return [
    {
      accessorKey: "title",
      header: t("Plan"),
      cell: ({ row }) => <span className="text-sm">{row.original.title}</span>,
      size: 260,
      meta: {
        label: t("Plan"),
        apiField: "title",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <PlanStatusBadge value={row.original.status} t={t} />,
      size: 170,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: planStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "stepCount",
      header: t("Steps"),
      cell: ({ row }) => <StepsCell row={row.original} t={t} />,
      size: 150,
      meta: { label: t("Steps"), apiField: "stepCount", filterable: false, sortable: true },
    },
    {
      accessorKey: "summary",
      header: t("Why"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.summary} truncateLength={90} />
      ),
      size: 360,
      meta: {
        label: t("Why"),
        apiField: "summary",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "runId",
      header: t("Run"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">{row.original.runId}</span>
      ),
      size: 220,
      meta: {
        label: t("Run"),
        apiField: "runId",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Proposed"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Proposed"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "decidedAt",
      header: t("Decided"),
      cell: ({ row }) =>
        row.original.decidedAt ? (
          <HoverCardTimestamp timestamp={row.original.decidedAt} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 170,
      meta: {
        label: t("Decided"),
        apiField: "decidedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}

/**
 * How far the plan got, read against how far it was meant to go. A plan that
 * stopped names the step, because that is the one somebody has to finish by
 * hand.
 */
function StepsCell({ row, t }: { row: AgentPlanRow; t: TranslateFn }) {
  if (row.status === "Failed" && row.failedStep) {
    return (
      <span className="text-xs" style={{ color: toneVar("danger") }}>
        {t("Stopped at {0} of {1}", row.failedStep, row.stepCount)}
      </span>
    );
  }

  if (row.status === "Pending" || row.status === "Rejected" || row.status === "Expired") {
    return <span className="text-xs tabular-nums">{row.stepCount}</span>;
  }

  return (
    <span className="text-xs tabular-nums">
      {t("{0} of {1} done", row.completedSteps, row.stepCount)}
    </span>
  );
}
