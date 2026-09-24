import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  agentTypeLabel,
  RunStatusBadge,
  runStatusChoices,
  TriggerBadge,
  triggerChoices,
} from "./agent-badges";
import { AgentSubjectCell } from "./agent-subject-cell";

export function getRunColumns(t: TranslateFn): ColumnDef<AgentRunRow>[] {
  return [
    {
      accessorKey: "agentType",
      header: t("Agent"),
      cell: ({ row }) => (
        <span className="font-medium">{agentTypeLabel(row.original.agentType, t)}</span>
      ),
      size: 170,
      meta: {
        label: t("Agent"),
        apiField: "agentType",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "trigger",
      header: t("Started by"),
      cell: ({ row }) => <TriggerBadge value={row.original.trigger} t={t} />,
      size: 130,
      meta: {
        label: t("Started by"),
        apiField: "trigger",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: triggerChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <RunStatusBadge value={row.original.status} t={t} />,
      size: 170,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: runStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "subjectType",
      header: t("Subject"),
      cell: ({ row }) => (
        <AgentSubjectCell
          subjectType={row.original.subjectType}
          subjectId={row.original.subjectId}
        />
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
      accessorKey: "summary",
      header: t("Summary"),
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.summary || row.original.errorMessage}
          truncateLength={80}
        />
      ),
      size: 320,
      meta: {
        label: t("Summary"),
        apiField: "summary",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "modelIdentifier",
      header: t("Model"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">
          {row.original.modelIdentifier || "—"}
        </span>
      ),
      size: 180,
      meta: {
        label: t("Model"),
        apiField: "modelIdentifier",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "startedAt",
      header: t("Started"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.startedAt ?? undefined} />,
      size: 170,
      meta: {
        label: t("Started"),
        apiField: "startedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "completedAt",
      header: t("Finished"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.completedAt ?? undefined} />,
      size: 170,
      meta: {
        label: t("Finished"),
        apiField: "completedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Created"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
