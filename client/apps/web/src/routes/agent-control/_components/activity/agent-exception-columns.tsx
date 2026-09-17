import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentExceptionRow } from "@/lib/graphql/agent-activity-tables";
import { toTitleCase } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { ResolutionBadge, resolutionChoices, SeverityBadge, severityChoices } from "./agent-badges";

function categoryLabel(value: string): string {
  return toTitleCase(value.replace(/([a-z])([A-Z])/g, "$1 $2"));
}

export function getExceptionColumns(t: TranslateFn): ColumnDef<AgentExceptionRow>[] {
  return [
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) => (
        <span className="font-medium">{categoryLabel(row.original.category)}</span>
      ),
      size: 220,
      meta: {
        label: t("Category"),
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "severity",
      header: t("Severity"),
      cell: ({ row }) => <SeverityBadge value={row.original.severity} t={t} />,
      size: 120,
      meta: {
        label: t("Severity"),
        apiField: "severity",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: severityChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "resolutionState",
      header: t("State"),
      cell: ({ row }) => <ResolutionBadge value={row.original.resolutionState} t={t} />,
      size: 130,
      meta: {
        label: t("State"),
        apiField: "resolutionState",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: resolutionChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "subjectType",
      header: t("Subject"),
      cell: ({ row }) => (
        <span className="flex flex-col leading-tight">
          <span>{row.original.subjectType}</span>
          <span className="text-muted-foreground font-mono text-[11px]">
            {row.original.subjectId}
          </span>
        </span>
      ),
      size: 220,
      meta: {
        label: t("Subject"),
        apiField: "subjectType",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "attemptSummary",
      header: t("What the agent tried"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.attemptSummary} truncateLength={90} />
      ),
      size: 360,
      meta: {
        label: t("What the agent tried"),
        apiField: "attemptSummary",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "blastRadius",
      header: t("Affected"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.blastRadius}</span>,
      size: 100,
      meta: { label: t("Affected"), apiField: "blastRadius", filterable: false, sortable: true },
    },
    {
      accessorKey: "createdAt",
      header: t("Raised"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Raised"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
