import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentProposalRow } from "@/lib/graphql/agent-activity-tables";
import { Progress } from "@trenova/shared/components/ui/progress";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ProposalStatusBadge, proposalStatusChoices, TierBadge } from "./agent-badges";

export function getProposalColumns(t: TranslateFn): ColumnDef<AgentProposalRow>[] {
  return [
    {
      accessorKey: "toolName",
      header: t("Proposed change"),
      cell: ({ row }) => (
        <span className="font-mono text-xs font-medium">{row.original.toolName}</span>
      ),
      size: 200,
      meta: {
        label: t("Proposed change"),
        apiField: "toolName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <ProposalStatusBadge value={row.original.status} t={t} />,
      size: 180,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: proposalStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "autonomyTier",
      header: t("Autonomy"),
      cell: ({ row }) => <TierBadge value={row.original.autonomyTier} t={t} />,
      size: 160,
      meta: {
        label: t("Autonomy"),
        apiField: "autonomyTier",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "confidence",
      header: t("Confidence"),
      cell: ({ row }) => {
        const percent = Math.round(Math.min(1, Math.max(0, row.original.confidence)) * 100);
        return (
          <span className="flex items-center gap-2">
            <Progress value={percent} className="h-1.5 w-16" />
            <span className="text-muted-foreground text-xs tabular-nums">{percent}%</span>
          </span>
        );
      },
      size: 140,
      meta: { label: t("Confidence"), apiField: "confidence", filterable: false, sortable: true },
    },
    {
      accessorKey: "rationale",
      header: t("Why"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.rationale} truncateLength={90} />
      ),
      size: 360,
      meta: {
        label: t("Why"),
        apiField: "rationale",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "runId",
      header: t("Run"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-[11px]">{row.original.runId}</span>
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
  ];
}
