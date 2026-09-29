import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { ExtractionShadowResultRow } from "@/lib/graphql/extraction-shadow";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { formatShare, formatUsd } from "../quality-model";
import { ShadowStatusBadge, ShadowVerdictBadge } from "./extraction-badges";
import { choicesOf } from "./extraction-model";
import { SHADOW_STATUS, SHADOW_VERDICT } from "./shadow-model";

export function getShadowResultColumns(t: TranslateFn): ColumnDef<ExtractionShadowResultRow>[] {
  return [
    {
      accessorKey: "createdAt",
      header: t("Shadowed"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 170,
      meta: {
        label: t("Shadowed"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "providerName",
      header: t("Candidate"),
      cell: ({ row }) => row.original.servedModel || row.original.providerName,
      size: 200,
      meta: {
        label: t("Candidate"),
        apiField: "providerName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <ShadowStatusBadge value={row.original.status} t={t} />,
      size: 130,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: choicesOf(SHADOW_STATUS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "verdict",
      header: t("Against production"),
      cell: ({ row }) =>
        row.original.verdict ? (
          <ShadowVerdictBadge value={row.original.verdict} t={t} />
        ) : (
          <span className="text-muted-foreground">{t("Not confirmed yet")}</span>
        ),
      size: 170,
      meta: {
        label: t("Against production"),
        apiField: "verdict",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: choicesOf(SHADOW_VERDICT),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "accuracy",
      header: t("Candidate accuracy"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.scoredCount > 0 ? formatShare(row.original.accuracy) : "—"}
        </span>
      ),
      size: 150,
      meta: {
        label: t("Candidate accuracy"),
        apiField: "accuracy",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "baselineAccuracy",
      header: t("Production accuracy"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.baselineScoredCount > 0 ? formatShare(row.original.baselineAccuracy) : "—"}
        </span>
      ),
      size: 150,
      meta: {
        label: t("Production accuracy"),
        apiField: "baselineAccuracy",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "latencyMs",
      header: t("Latency"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.latencyMs > 0 ? t("{0} ms", row.original.latencyMs) : "—"}
        </span>
      ),
      size: 110,
      meta: { label: t("Latency"), apiField: "latencyMs", filterable: false, sortable: true },
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
