import type { DeskBoardMetric } from "@/lib/graphql/desk-board-card";

/** A chart the server shaped for a board: ticks, series, marks, a band and lines, and rows. */
export type DeskBoardChart = NonNullable<DeskBoardMetric["chart"]>;
export type DeskBoardChartRow = DeskBoardChart["rows"][number];
export type DeskBoardChartTick = DeskBoardChart["ticks"][number];
export type DeskBoardChartLine = NonNullable<DeskBoardChart["target"]>;
/** Parts of a whole: used, over or credit, against the total. */
export type DeskBoardSplit = NonNullable<DeskBoardMetric["split"]>;
/** Cover against need: rows by columns of cells. */
export type DeskBoardGrid = NonNullable<DeskBoardMetric["grid"]>;
export type DeskBoardGridCell = DeskBoardGrid["rows"][number]["cells"][number];
export type DeskBoardLogEntry = DeskBoardMetric["log"][number];
export type DeskBoardMetricTable = NonNullable<DeskBoardMetric["table"]>;
export type DeskBoardMetricCell = DeskBoardMetricTable["rows"][number][number];

/** The forms a metric is drawn in, as the catalog names them (`boardcatalog.Form`). */
export const METRIC_FORMS = [
  "trend",
  "bars",
  "columns",
  "split",
  "grid",
  "dots",
  "spans",
  "log",
  "checks",
  "table",
  "figures",
] as const;

export type MetricForm = (typeof METRIC_FORMS)[number];

export function isMetricForm(value: string): value is MetricForm {
  return (METRIC_FORMS as readonly string[]).includes(value);
}
