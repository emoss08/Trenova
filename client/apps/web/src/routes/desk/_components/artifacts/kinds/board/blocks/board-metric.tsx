import "../../../../../_styles/board-beyond.css";
import type { DeskBoardMetric } from "@/lib/graphql/desk-board-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArtIcon } from "../../../desk-art-kinds";
import {
  Checks,
  Readout,
  type OpenRecord,
  type OpensInPlace,
  type ReadoutFigure,
} from "../board-parts";
import { BoardGrid, gridDraws } from "./board-grid";
import { BoardLog } from "./board-log";
import { BoardMetricTable, tableDraws } from "./board-metric-table";
import { BoardSplit, splitDraws } from "./board-split";
import { BarsChart, barsDraw } from "./chart-bars";
import { ColumnsChart, columnsDraw } from "./chart-columns";
import { DotsChart, dotsDraw } from "./chart-dots";
import { SpansChart, spansDraw } from "./chart-spans";
import { TrendChart, trendDraws } from "./chart-trend";

/** A metric's figures as a readout draws figures: each read now, labelled as the catalog labels it. */
export function metricFigures(metric: DeskBoardMetric): ReadoutFigure[] {
  return metric.figures.map((figure) => ({
    label: figure.label,
    value: figure.value ?? null,
    was: figure.was ?? null,
    wasAt: figure.wasAt ?? null,
    tone: figure.tone,
  }));
}

/** A form the server filled with nothing to draw, said where the form would stand. */
export function NothingToDraw({ form }: { form: string }) {
  const t = useT();
  const text =
    form === "table"
      ? t("The table has no rows now.")
      : form === "log"
        ? t("Nothing happened in this period.")
        : form === "checks"
          ? t("There is nothing to check.")
          : form === "figures"
            ? t("There is nothing to read now.")
            : t("The chart has nothing to draw.");
  return (
    <div className="dk-bd-empty">
      <span>
        <ArtIcon name="check" size={14} stroke={2.4} />
      </span>
      <span>{text}</span>
    </div>
  );
}

/**
 * A metric or chart block, drawn in the form the catalog gives it: a trend, bars, columns, dots
 * or spans as the board's own SVG; a split, a grid, a log or a table in their own parts; checks
 * as the board's checks and figures as its readout. A form whose data is absent says it has
 * nothing to draw rather than leaving an empty box.
 */
export function BoardMetric({
  metric,
  answeredAt,
  now,
  open,
  opensInPlace,
}: {
  metric: DeskBoardMetric;
  answeredAt: number;
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const chart = metric.chart;
  const title = metric.title;
  switch (metric.form) {
    case "trend":
      return chart && trendDraws(chart) ? (
        <TrendChart chart={chart} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "bars":
      return chart && barsDraw(chart) ? (
        <BarsChart chart={chart} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "columns":
      return chart && columnsDraw(chart) ? (
        <ColumnsChart chart={chart} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "dots":
      return chart && dotsDraw(chart) ? (
        <DotsChart chart={chart} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "spans":
      return chart && spansDraw(chart) ? (
        <SpansChart chart={chart} title={title} now={now} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "split":
      return splitDraws(metric.split) ? (
        <BoardSplit split={metric.split} figures={metric.figures} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "grid":
      return gridDraws(metric.grid) ? (
        <BoardGrid grid={metric.grid} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "log":
      return metric.log.length > 0 ? (
        <BoardLog entries={metric.log} now={now} open={open} opensInPlace={opensInPlace} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "table":
      return tableDraws(metric.table) ? (
        <BoardMetricTable table={metric.table} title={title} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    case "checks":
      return metric.checks.length > 0 ? (
        <Checks checks={metric.checks} open={open} opensInPlace={opensInPlace} />
      ) : (
        <NothingToDraw form={metric.form} />
      );
    default:
      return metric.figures.length > 0 ? (
        <Readout figures={metricFigures(metric)} answeredAt={answeredAt} now={now} />
      ) : null;
  }
}
