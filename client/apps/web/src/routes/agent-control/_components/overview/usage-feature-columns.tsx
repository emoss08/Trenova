import {
  aiUsageFeatureLabel,
  formatLatencyCompact,
  formatMillions,
  formatUsd,
} from "@/lib/ai-usage-format";
import type { AIUsageFeatureRow } from "@/lib/graphql/ai-usage-features-table";
import type { AiUsageFeature } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";

const FEATURES: AiUsageFeature[] = [
  "AgentTurn",
  "AgentEvaluation",
  "TableQuery",
  "FormulaGenerate",
  "FormulaExplain",
  "ShipmentImportChat",
  "DocumentIntelligenceRoute",
  "DocumentIntelligenceExtract",
];

/** The share of the busiest feature's calls a bar may never fall below, so a failure shows. */
const MIN_FAILED_BAR_PERCENT = 3;

export function getUsageFeatureColumns(t: TranslateFn, busiest: number): ColumnDef<AIUsageFeatureRow>[] {
  const scale = Math.max(busiest, 1);

  return [
    {
      id: "feature",
      accessorKey: "feature",
      header: t("Feature"),
      cell: ({ row }) => (
        <b className="rg">{aiUsageFeatureLabel(row.original.feature ?? null, t)}</b>
      ),
      size: 230,
      meta: {
        label: t("Feature"),
        apiField: "feature",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: FEATURES.map((feature) => ({
          value: feature,
          label: aiUsageFeatureLabel(feature, t),
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "calls",
      accessorKey: "calls",
      header: t("Calls"),
      cell: ({ row }) => {
        const { calls, failed } = row.original;
        return (
          <span className="ucl">
            <span className="mono">{calls.toLocaleString()}</span>
            <span className="ubar">
              <i style={{ width: `${(calls / scale) * 100}%` }} />
              {failed > 0 && (
                <i
                  className="f"
                  style={{ width: `${Math.max(MIN_FAILED_BAR_PERCENT, (failed / scale) * 100)}%` }}
                />
              )}
            </span>
          </span>
        );
      },
      size: 190,
      meta: { label: t("Calls"), apiField: "calls", filterable: true, sortable: true, filterType: "number" },
    },
    {
      id: "failed",
      accessorKey: "failed",
      header: t("Failed"),
      cell: ({ row }) =>
        row.original.failed > 0 ? (
          <span className="mono t-d">{row.original.failed.toLocaleString()}</span>
        ) : (
          <span className="dim">—</span>
        ),
      size: 80,
      meta: { label: t("Failed"), apiField: "failed", filterable: true, sortable: true, filterType: "number" },
    },
    {
      id: "tokens",
      accessorFn: (row) => row.inputTokens + row.outputTokens,
      header: t("Tokens"),
      cell: ({ row }) => (
        <span className="mono">
          {`${formatMillions(row.original.inputTokens + row.original.outputTokens)}M`}
        </span>
      ),
      size: 90,
      meta: { label: t("Tokens"), apiField: "tokens", filterable: false, sortable: true, filterType: "number" },
    },
    {
      id: "costUsd",
      accessorKey: "costUsd",
      header: t("Spend"),
      cell: ({ row }) =>
        row.original.pricedCalls > 0 ? (
          <span className="mono">{formatUsd(row.original.costUsd) ?? "—"}</span>
        ) : (
          <span className="dim">—</span>
        ),
      size: 80,
      meta: { label: t("Spend"), apiField: "costUsd", filterable: false, sortable: true, filterType: "number" },
    },
    {
      id: "latencyP50Ms",
      accessorKey: "latencyP50Ms",
      header: t("Median"),
      cell: ({ row }) =>
        row.original.latencyP50Ms > 0 ? (
          <span className="mono">{formatLatencyCompact(row.original.latencyP50Ms)}</span>
        ) : (
          <span className="dim">—</span>
        ),
      size: 90,
      meta: {
        label: t("Median"),
        apiField: "latencyP50Ms",
        filterable: false,
        sortable: true,
        filterType: "number",
      },
    },
  ];
}
