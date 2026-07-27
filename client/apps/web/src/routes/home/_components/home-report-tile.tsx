import type { ReportDashboardTile } from "@/types/report";
import { useMemo } from "react";
import {
  DashboardTileBody,
  TileFooterLink,
} from "@/routes/reports/dashboards/_components/dashboard-tile";
import { useTileReport } from "@/routes/reports/dashboards/_components/use-tile-report";
import { WidgetShell } from "./widget-shell";

const NO_VALUES: Record<string, unknown> = {};
const NO_FILTERS: never[] = [];

export type HomeReportTileProps = {
  title?: string;
  definitionId?: string;
  cannedKey?: string;
  chartId?: string;
  columnId?: string;
  limit?: number;
};

/**
 * A saved or canned report drawn on the home canvas. It builds the same tile
 * descriptor a report dashboard uses and hands it to the same renderer, so a
 * number on Home is the same number that report exports — there is no second
 * query path to drift.
 */
export function HomeReportTile({
  title,
  definitionId,
  cannedKey,
  chartId,
  columnId,
  limit,
}: HomeReportTileProps) {
  const tile = useMemo<ReportDashboardTile>(
    () => ({
      id: "home_report",
      // The measure wins over the chart: a tile told to show one column is a
      // KPI even if the report also happens to define charts.
      kind: columnId ? "kpi" : chartId ? "chart" : "table",
      title,
      definitionId,
      cannedKey,
      chartId,
      columnId,
      limit,
      x: 0,
      y: 0,
      w: 6,
      h: 5,
    }),
    [title, definitionId, cannedKey, chartId, columnId, limit],
  );

  const report = useTileReport(tile);

  return (
    <WidgetShell
      title={title || report.name || "Report"}
      actions={<TileFooterLink tile={tile} />}
      scroll={false}
      bodyClassName="p-0"
    >
      <DashboardTileBody
        tile={tile}
        params={NO_VALUES}
        filters={NO_FILTERS}
        filterValues={NO_VALUES}
        tileFilters={NO_FILTERS}
        tileFilterValues={NO_VALUES}
      />
    </WidgetShell>
  );
}
