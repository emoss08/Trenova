import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentMemoryRow } from "@/lib/graphql/agent-memories";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  MemoryKindBadge,
  MemoryStatusBadge,
  memoryKindChoices,
  memorySourceChoices,
  memorySourceLabel,
  memoryStatusChoices,
} from "../activity/agent-badges";

const SUBJECT_LABEL: Record<NonNullable<AgentMemoryRow["subjectType"]>, string> = {
  Customer: "customer",
  Location: "location",
  Worker: "driver",
  Carrier: "carrier",
};

/** Who the memory reaches: one record, one tool, or every agent. */
export function memoryScope(row: AgentMemoryRow, t: TranslateFn): string {
  const parts: string[] = [];
  if (row.subjectType) {
    const kind = t(SUBJECT_LABEL[row.subjectType]);
    parts.push(row.subjectLabel !== "" ? `${row.subjectLabel} (${kind})` : kind);
  }
  if (row.toolName !== "") {
    parts.push(t("tool {0}", row.toolName));
  }

  return parts.length === 0 ? t("Every agent") : parts.join(" · ");
}

export function getMemoryColumns(t: TranslateFn): ColumnDef<AgentMemoryRow>[] {
  return [
    {
      accessorKey: "content",
      header: t("Memory"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.content} truncateLength={120} />
      ),
      size: 420,
      meta: {
        label: t("Memory"),
        apiField: "content",
        filterable: true,
        sortable: false,
        filterType: "text",
      },
    },
    {
      accessorKey: "kind",
      header: t("Kind"),
      cell: ({ row }) => <MemoryKindBadge value={row.original.kind} t={t} />,
      size: 130,
      meta: {
        label: t("Kind"),
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: memoryKindChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "scope",
      accessorKey: "subjectLabel",
      header: t("About"),
      cell: ({ row }) => <span className="text-xs">{memoryScope(row.original, t)}</span>,
      size: 220,
      meta: {
        label: t("About"),
        apiField: "subjectLabel",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <MemoryStatusBadge value={row.original.status} t={t} />,
      size: 120,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: memoryStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "source",
      header: t("Recorded by"),
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {memorySourceLabel(row.original.source, t)}
        </span>
      ),
      size: 130,
      meta: {
        label: t("Recorded by"),
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: memorySourceChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "useCount",
      header: t("Read"),
      cell: ({ row }) => (
        <span className="text-xs tabular-nums">
          {t("{0, plural, one {# time} other {# times}}", row.original.useCount)}
        </span>
      ),
      size: 110,
      meta: { label: t("Read"), apiField: "useCount", filterable: false, sortable: true },
    },
    {
      accessorKey: "toolName",
      header: t("Tool"),
      cell: ({ row }) =>
        row.original.toolName !== "" ? (
          <span className="font-mono text-xs">{row.original.toolName}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 180,
      meta: {
        label: t("Tool"),
        apiField: "toolName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Recorded"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Recorded"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "expiresAt",
      header: t("Until"),
      cell: ({ row }) =>
        row.original.expiresAt ? (
          <HoverCardTimestamp timestamp={row.original.expiresAt} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 170,
      meta: {
        label: t("Until"),
        apiField: "expiresAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "lastUsedAt",
      header: t("Last read"),
      cell: ({ row }) =>
        row.original.lastUsedAt ? (
          <HoverCardTimestamp timestamp={row.original.lastUsedAt} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 170,
      meta: {
        label: t("Last read"),
        apiField: "lastUsedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
