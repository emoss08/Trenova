import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { ExtractionEvalRunRow } from "@/lib/graphql/extraction-eval";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatShare, formatUsd } from "../quality-model";
import { RunStatusBadge } from "./extraction-badges";
import { RUN_STATUS, choicesOf } from "./extraction-model";

export function getRunColumns(t: TranslateFn): ColumnDef<ExtractionEvalRunRow>[] {
  return [
    {
      accessorKey: "createdAt",
      header: t("Started"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Started"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "providerName",
      header: t("Provider"),
      cell: ({ row }) => row.original.providerName,
      size: 180,
      meta: {
        label: t("Provider"),
        apiField: "providerName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "servedModel",
      header: t("Model"),
      cell: ({ row }) => row.original.servedModel || row.original.providerModel || "—",
      size: 220,
      meta: {
        label: t("Model"),
        apiField: "servedModel",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <RunStatusBadge value={row.original.status} t={t} />,
      size: 140,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: choicesOf(RUN_STATUS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "accuracy",
      header: t("Accuracy"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.scoredCount > 0 ? formatShare(row.original.accuracy) : "—"}
        </span>
      ),
      size: 110,
      meta: {
        label: t("Accuracy"),
        apiField: "accuracy",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      id: "cases",
      accessorKey: "casesCompleted",
      header: t("Cases"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {t("{0} of {1}", row.original.casesCompleted, row.original.casesTotal)}
        </span>
      ),
      size: 110,
      meta: { label: t("Cases"), apiField: "casesCompleted", filterable: false, sortable: true },
    },
    {
      accessorKey: "avgLatencyMs",
      header: t("Avg latency"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.avgLatencyMs > 0 ? t("{0} ms", row.original.avgLatencyMs) : "—"}
        </span>
      ),
      size: 120,
      meta: {
        label: t("Avg latency"),
        apiField: "avgLatencyMs",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "costUsd",
      header: t("Cost"),
      cell: ({ row }) => <span className="tabular-nums">{formatUsd(row.original.costUsd)}</span>,
      size: 100,
      meta: { label: t("Cost"), apiField: "costUsd", filterable: false, sortable: true },
    },
  ];
}
