import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportPreview } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import type { ReportChartSpec, ReportDashboardFilter, ReportDashboardTile } from "@/types/report";
import { CircleAlertIcon, ExternalLinkIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { irToInput } from "../../builder/_components/builder-state";
import { ReportChart, type CategorySelect } from "../../_components/report-chart";
import { ResultGrid } from "../../_components/result-grid";
import { applyDashboardFilters, resolveTileParams } from "./dashboard-state";
import { useTileReport } from "./use-tile-report";

type DashboardTileBodyProps = {
  tile: ReportDashboardTile;
  /** Values for the dashboard's name-bound parameters. */
  params: Record<string, unknown>;
  /** The dashboard's field filters, and the value chosen for each. */
  filters: ReportDashboardFilter[];
  filterValues: Record<string, unknown>;
  /** Filters scoped to this tile alone, keyed by field. */
  tileFilters: ReportDashboardFilter[];
  tileFilterValues: Record<string, unknown>;
  /** Set when clicking a chart category should filter the dashboard. */
  categorySelect?: CategorySelect;
};

function TileMessage({ children, tone }: { children: React.ReactNode; tone?: "error" }) {
  return (
    <div
      className={cn(
        "flex min-h-0 flex-1 items-center justify-center p-4 text-center text-xs",
        tone === "error" ? "text-destructive" : "text-muted-foreground",
      )}
    >
      {children}
    </div>
  );
}

function TileSkeleton() {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-1.5 p-3">
      {Array.from({ length: 5 }, (_, i) => (
        <Skeleton key={i} className="h-5" style={{ opacity: 1 - i * 0.15 }} />
      ))}
    </div>
  );
}

/** The chart a tile draws, or a synthesized KPI spec for a single measure. */
function useTileChart(
  tile: ReportDashboardTile,
  charts: ReportChartSpec[] | undefined,
): ReportChartSpec | null {
  return useMemo(() => {
    if (tile.kind === "kpi") {
      return tile.columnId
        ? {
            id: `${tile.id}_kpi`,
            type: "kpi",
            title: tile.title,
            seriesIds: [tile.columnId],
          }
        : null;
    }
    if (tile.kind !== "chart") return null;
    const available = charts ?? [];
    return available.find((chart) => chart.id === tile.chartId) ?? available[0] ?? null;
  }, [tile, charts]);
}

export function DashboardTileBody({
  tile,
  params,
  filters,
  filterValues,
  tileFilters,
  tileFilterValues,
  categorySelect,
}: DashboardTileBodyProps) {
  const report = useTileReport(tile);
  const chart = useTileChart(tile, report.ir?.charts);

  // A tile renders from its report's own definition rather than through the
  // saved dashboard, so a tile you just added previews immediately instead of
  // waiting to be persisted. Authorization is unchanged: the preview compiles
  // and authorizes the same IR the report itself would run.
  const input = useMemo(() => {
    if (tile.kind === "text" || !report.ir) return null;
    const scoped = tile.limit ? { ...report.ir, limit: tile.limit } : report.ir;
    // Shared filters first, then this tile's own — both override a matching
    // condition rather than stacking a second one on the same field.
    const shared = applyDashboardFilters(scoped, filters, filterValues);
    return irToInput(applyDashboardFilters(shared, tileFilters, tileFilterValues));
  }, [tile.kind, tile.limit, report.ir, filters, filterValues, tileFilters, tileFilterValues]);

  const tileParams = useMemo(
    () => resolveTileParams(tile, params, report.ir?.parameters ?? []),
    [tile, params, report.ir],
  );

  const data = useReportPreview(input, tileParams);

  if (tile.kind === "text") {
    return (
      <div className="min-h-0 flex-1 overflow-auto p-4 text-sm whitespace-pre-wrap text-foreground/90">
        {tile.text}
      </div>
    );
  }

  if (report.loading) {
    return <TileSkeleton />;
  }

  // Without a resolved report there is nothing to query, and a disabled query
  // would otherwise sit on "pending" forever.
  if (!report.ir) {
    return <TileMessage>This tile&apos;s report is no longer available.</TileMessage>;
  }

  if (data.isError) {
    return (
      <TileMessage tone="error">
        <span className="flex flex-col items-center gap-1">
          <CircleAlertIcon className="size-4" />
          {graphQLErrorMessage(data.error, "This tile could not be loaded")}
        </span>
      </TileMessage>
    );
  }

  if (data.isPending) {
    return <TileSkeleton />;
  }

  const columns = data.data?.columns ?? [];
  const rows = Array.isArray(data.data?.rows) ? (data.data.rows as unknown[][]) : [];
  const totals = Array.isArray(data.data?.totals) ? (data.data.totals as unknown[]) : null;

  if (tile.kind === "table") {
    return (
      <ResultGrid
        columns={columns}
        rows={rows}
        totals={totals}
        density="compact"
        emptyMessage="No rows."
      />
    );
  }

  if (!chart) {
    return (
      <TileMessage>
        {tile.kind === "kpi"
          ? "Choose the measure this KPI shows."
          : "This report has no chart configured."}
      </TileMessage>
    );
  }

  return (
    <ReportChart
      chart={chart}
      columns={columns}
      rows={rows}
      className="min-h-0 flex-1"
      categorySelect={categorySelect}
    />
  );
}

export function TileFooterLink({ tile }: { tile: ReportDashboardTile }) {
  const href = tile.definitionId
    ? `/reports/explore/${tile.definitionId}`
    : tile.cannedKey
      ? `/reports/explore?canned=${encodeURIComponent(tile.cannedKey)}`
      : null;

  if (!href) return null;

  return (
    <Button
      variant="ghost"
      size="icon"
      className="size-6 opacity-0 transition-opacity group-hover/tile:opacity-100"
      render={<Link to={href} />}
      aria-label="Open this report"
      title="Open this report"
    >
      <ExternalLinkIcon className="size-3.5" />
    </Button>
  );
}
