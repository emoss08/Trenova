import {
  isNumericDisplay,
  reportNumericValue,
  type ReportColumnDisplayLike,
} from "@trenova/shared/lib/report-format";

/**
 * A report result as the sheet reads it, whichever way it arrived: a run's
 * stored sample (raw values with each column's resolved display) or a
 * preview's rows (raw values with the catalog type and format hint).
 */
export type SheetColumn = {
  id: string;
  label: string;
  /** The catalog value type: string, int, decimal, epoch, … */
  type: string;
  /** The catalog format hint (money, percent, …), when the column has no resolved display. */
  hint: string;
  display: ReportColumnDisplayLike | null;
  numeric: boolean;
  dated: boolean;
  percent: boolean;
};

export type SheetSource = {
  columns: SheetColumn[];
  rows: unknown[][];
  totals: unknown[] | null;
  /** Rows in the whole result. */
  rowCount: number;
  /** The result holds rows the source does not. */
  more: boolean;
  /** The result hit its row cap, so the rows and totals are of the kept rows. */
  truncated: boolean;
};

export type SheetHints = {
  chart: { type: string; xColumnId: string | null; seriesIds: readonly string[] } | null;
  sort: readonly { columnId: string; direction: string }[];
};

/** The rows a sheet shows of a run; the rest are in the file. */
export const SHEET_ROWS = 8;
/** The columns a sheet shows; the rest are in the file. */
export const SHEET_COLUMNS = 6;
/** The headline figures a run shows over its sheet. */
export const MAX_KPIS = 4;

const NUMERIC_TYPES = new Set(["int", "decimal"]);

/** A column as the sheet reads it, from its value type, format hint and resolved display. */
export function sheetColumn(input: {
  id: string;
  label: string;
  type: string;
  format?: string | null;
  display?: ReportColumnDisplayLike | null;
}): SheetColumn {
  const display = input.display ?? null;
  const hint = input.format ?? "";
  const dated = display ? display.style === "date" : input.type === "epoch";

  return {
    id: input.id,
    label: input.label || input.id,
    type: input.type,
    hint,
    display,
    numeric: !dated && (display ? isNumericDisplay(display) : NUMERIC_TYPES.has(input.type)),
    dated,
    percent: display ? display.style === "percent" : hint === "percent",
  };
}

export function numberAt(row: readonly unknown[] | null, index: number): number | null {
  return row ? reportNumericValue(row[index]) : null;
}

/**
 * How a result is pictured.
 * - ranked: a label per row and a measure to rank by, drawn as a bar in each row;
 * - trend: periods along the bottom, a measure as a line and another as columns under it;
 * - table: the rows alone.
 */
export type Picture =
  | { kind: "ranked"; label: number; measure: number; ranked: boolean }
  | { kind: "trend"; period: number; line: number; columns: number | null }
  | { kind: "table" };

const RANKED_CHARTS = new Set(["bar", "hbar", "pie", "donut"]);
const TREND_CHARTS = new Set(["line", "area"]);
const TABLE_CHARTS = new Set(["map", "scatter", "kpi"]);

/**
 * The picture a result reads best as. The report's own chart decides when it
 * has one: bars, pies and donuts rank, lines and areas trend, and a map, a
 * scatter or a lone figure leaves the table. Without one, a result led by
 * dates trends and a result led by a label ranks by its sorted measure, or
 * its first.
 */
export function pictureOf(source: SheetSource, hints: SheetHints): Picture {
  const { columns } = source;
  const index = (id: string | null | undefined) =>
    id ? columns.findIndex((column) => column.id === id) : -1;
  const measures = columns.map((column, at) => (column.numeric ? at : -1)).filter((at) => at >= 0);
  if (measures.length === 0 || source.rows.length < 2) {
    return { kind: "table" };
  }

  const chart = hints.chart;
  if (chart && TABLE_CHARTS.has(chart.type)) {
    return { kind: "table" };
  }
  const series = (chart?.seriesIds ?? [])
    .map((id) => index(id))
    .filter((at) => at >= 0 && columns[at].numeric);
  const sorted = hints.sort[0];
  const sortedMeasure =
    sorted && columns[index(sorted.columnId)]?.numeric ? index(sorted.columnId) : -1;

  const trendsOn = (period: number): Picture | null => {
    if (period < 0 || !columns[period].dated || source.more || source.rows.length < 3) {
      return null;
    }
    const [line = measures[0], second] = series.length > 0 ? series : measures;
    return { kind: "trend", period, line, columns: second ?? null };
  };
  const ranksOn = (label: number, measure: number): Picture | null => {
    if (label < 0 || columns[label].numeric || columns[label].dated || measure < 0) {
      return null;
    }
    if (columns[measure].percent) {
      return null;
    }
    const positive = source.rows.every((row) => (numberAt(row, measure) ?? 0) >= 0);
    if (!positive) {
      return null;
    }
    const ranked = sortedMeasure === measure && sorted.direction.toLowerCase() === "desc";
    return { kind: "ranked", label, measure, ranked };
  };

  if (chart && TREND_CHARTS.has(chart.type)) {
    return trendsOn(index(chart.xColumnId)) ?? { kind: "table" };
  }
  if (chart && RANKED_CHARTS.has(chart.type)) {
    return ranksOn(index(chart.xColumnId), series[0] ?? measures[0]) ?? { kind: "table" };
  }

  const lead = columns.findIndex((column) => !column.numeric);
  if (lead >= 0 && columns[lead].dated) {
    return trendsOn(lead) ?? { kind: "table" };
  }
  return ranksOn(lead, sortedMeasure >= 0 ? sortedMeasure : measures[0]) ?? { kind: "table" };
}

/** The columns a sheet shows: the picture's own first, then the rest in order, up to the cap. */
export function shownColumns(source: SheetSource, picture: Picture): number[] {
  const all = source.columns.map((_, at) => at);
  const leading =
    picture.kind === "ranked" ? [picture.label] : picture.kind === "trend" ? [picture.period] : [];
  const rest = all.filter((at) => !leading.includes(at));

  return [...leading, ...rest].slice(0, SHEET_COLUMNS);
}

/** A value's share of a total, when both are measured and the total is above zero. */
export function shareOf(value: number | null, total: number | null): number | null {
  if (value === null || total === null || total <= 0 || value < 0) {
    return null;
  }
  return value / total;
}

/** The total a ranked measure's shares are of: the grand total, or none when there is none. */
export function rankedTotal(source: SheetSource, picture: Picture): number | null {
  if (picture.kind !== "ranked") {
    return null;
  }
  return numberAt(source.totals, picture.measure);
}

export type Delta = { change: number; direction: "up" | "down" | "flat" };

/**
 * How a figure moved since the previous run of the same report: its change as
 * a fraction of the earlier figure. A figure that was zero or absent before has
 * no change to state, and a percentage moving is not a percentage of itself.
 */
export function deltaOf(
  current: number | null,
  previous: number | null,
  percent: boolean,
): Delta | null {
  if (current === null || previous === null || previous === 0 || percent) {
    return null;
  }
  const change = (current - previous) / Math.abs(previous);
  const direction = Math.abs(change) < 0.0005 ? "flat" : change > 0 ? "up" : "down";
  return { change, direction };
}

export type Kpi =
  | { kind: "total"; column: number; delta: Delta | null }
  | { kind: "rows"; delta: Delta | null }
  | { kind: "peak"; period: number; measure: number; row: number };

/**
 * The headline figures over a sheet: the grand total of each measure it has
 * one for, then how many rows the result holds; a trend gives its last place
 * to the period its line peaks in, since a count of periods says nothing.
 */
export function kpisOf(
  source: SheetSource,
  picture: Picture,
  previous: { totals: ReadonlyMap<string, unknown>; rowCount: number | null } | null,
): Kpi[] {
  const kpis: Kpi[] = [];
  const room = MAX_KPIS - 1;
  source.columns.forEach((column, at) => {
    const total = numberAt(source.totals, at);
    if (!column.numeric || total === null || kpis.length >= room) {
      return;
    }
    const before = previous ? reportNumericValue(previous.totals.get(column.id)) : null;
    kpis.push({
      kind: "total",
      column: at,
      delta: source.truncated ? null : deltaOf(total, before, column.percent),
    });
  });

  if (picture.kind === "trend") {
    let peak = -1;
    source.rows.forEach((row, at) => {
      const value = numberAt(row, picture.line);
      if (
        value !== null &&
        (peak < 0 || value > (numberAt(source.rows[peak], picture.line) ?? 0))
      ) {
        peak = at;
      }
    });
    if (peak >= 0) {
      kpis.push({ kind: "peak", period: picture.period, measure: picture.line, row: peak });
      return kpis;
    }
  }

  kpis.push({
    kind: "rows",
    delta:
      source.truncated || !previous || previous.rowCount === null
        ? null
        : deltaOf(source.rowCount, previous.rowCount, false),
  });
  return kpis;
}

/** A trend's points, oldest first: the period's raw value and each measure's figure. */
export type TrendPoint = { row: number; line: number | null; column: number | null };

export function trendPoints(
  source: SheetSource,
  picture: Extract<Picture, { kind: "trend" }>,
): TrendPoint[] {
  const order = source.rows.map((row, at) => ({ at, key: row[picture.period] }));
  order.sort((a, b) => compareRaw(a.key, b.key));
  return order.map(({ at }) => ({
    row: at,
    line: numberAt(source.rows[at], picture.line),
    column: picture.columns === null ? null : numberAt(source.rows[at], picture.columns),
  }));
}

function compareRaw(a: unknown, b: unknown): number {
  const left = reportNumericValue(a);
  const right = reportNumericValue(b);
  if (left !== null && right !== null) {
    return left - right;
  }
  return String(a ?? "").localeCompare(String(b ?? ""));
}
