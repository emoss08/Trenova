import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AgentEvalCaseRow } from "@/lib/graphql/agent-eval-cases";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { TriggerBadge, triggerChoices } from "../../activity/agent-badges";
import {
  EvalCaseSourceBadge,
  EvalCaseStatusBadge,
  evalCaseSourceChoices,
  evalCaseStatusChoices,
} from "./eval-case-badges";
import { describeExpected, readExpected } from "./eval-case-model";

export function getEvalCaseColumns(t: TranslateFn): ColumnDef<AgentEvalCaseRow>[] {
  return [
    {
      accessorKey: "title",
      header: t("Case"),
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.title || row.original.input}
          truncateLength={90}
        />
      ),
      size: 320,
      meta: {
        label: t("Case"),
        apiField: "title",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EvalCaseStatusBadge value={row.original.status} t={t} />,
      size: 130,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: evalCaseStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "expects",
      accessorKey: "expected",
      header: t("Expects"),
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {describeExpected(readExpected(row.original.expected), t)}
        </span>
      ),
      size: 260,
      meta: { label: t("Expects"), apiField: "expected", filterable: false, sortable: false },
    },
    {
      accessorKey: "source",
      header: t("Captured from"),
      cell: ({ row }) => <EvalCaseSourceBadge value={row.original.source} t={t} />,
      size: 150,
      meta: {
        label: t("Captured from"),
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: evalCaseSourceChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "trigger",
      header: t("Asked as"),
      cell: ({ row }) => <TriggerBadge value={row.original.trigger} t={t} />,
      size: 130,
      meta: {
        label: t("Asked as"),
        apiField: "trigger",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: triggerChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "weight",
      header: t("Weight"),
      cell: ({ row }) => <span className="text-xs tabular-nums">{row.original.weight}</span>,
      size: 90,
      meta: { label: t("Weight"), apiField: "weight", filterable: false, sortable: false },
    },
    {
      accessorKey: "createdAt",
      header: t("Captured"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Captured"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "expiresAt",
      header: t("Expires"),
      cell: ({ row }) =>
        row.original.expiresAt ? (
          <HoverCardTimestamp timestamp={row.original.expiresAt} />
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 170,
      meta: {
        label: t("Expires"),
        apiField: "expiresAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
