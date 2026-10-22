import { isNumericDisplay, type ReportColumnDisplayLike } from "@trenova/shared/lib/report-format";
import { formatReportColumns, reportCellText, type ReportCell } from "@trenova/shared/rules/report";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";

/** A report column as the grid and the charts read it. */
export type ReportCellColumn = {
  readonly display?: ReportColumnDisplayLike | null;
  /** The catalog value type; decimals arrive as strings and read as numbers only with it. */
  readonly type?: string;
};

/** Rows formatted per call: a screen of a virtualised grid, and some of the next. */
export const REPORT_PAGE_ROWS = 200;

/**
 * The value type the module reads a column's values with. A column without one (a chart fed
 * by a caller that only kept the display) is read by its style, so a decimal string in a
 * currency column is still a number.
 */
export function cellValueType(column: ReportCellColumn): string {
  if (column.type) return column.type;
  if (isNumericDisplay(column.display)) return "decimal";
  return column.display?.style === "date" ? "epoch" : "";
}

export type ReportPages = {
  readonly cell: (rowIndex: number, columnIndex: number) => ReportCell;
  readonly text: (rowIndex: number, columnIndex: number) => string;
};

/**
 * Formats a result's rows through the rules module a page at a time: the first time a row is
 * asked for, its whole page of rows (every column) is formatted in one `report.format` call
 * and kept. Build one per result (memoised on the module, the columns and the rows), so
 * scrolling formats each page once and a re-render formats nothing. The cache is the memo of a
 * pure function of those three inputs, never state a render can observe changing.
 */
export function createReportPages(
  rules: RulesRuntime,
  columns: readonly ReportCellColumn[],
  rows: readonly (readonly unknown[])[],
  pageRows: number = REPORT_PAGE_ROWS,
): ReportPages {
  const pages = new Map<number, ReportCell[][]>();

  const page = (index: number): ReportCell[][] => {
    const cached = pages.get(index);
    if (cached) return cached;
    const slice = rows.slice(index * pageRows, (index + 1) * pageRows);
    const formatted = formatReportColumns(
      rules,
      columns.map((column, columnIndex) => ({
        display: column.display,
        valueType: cellValueType(column),
        values: slice.map((row) => row[columnIndex]),
      })),
    );
    pages.set(index, formatted);
    return formatted;
  };

  const cell = (rowIndex: number, columnIndex: number): ReportCell =>
    page(Math.floor(rowIndex / pageRows))[columnIndex][rowIndex % pageRows];

  return {
    cell,
    text: (rowIndex, columnIndex) =>
      reportCellText(cell(rowIndex, columnIndex), columns[columnIndex]?.display),
  };
}

/** Formats one row (a totals row, a KPI's value) in one call, a cell per column. */
export function formatReportRow(
  rules: RulesRuntime,
  columns: readonly ReportCellColumn[],
  row: readonly unknown[],
): ReportCell[] {
  return formatReportColumns(
    rules,
    columns.map((column, index) => ({
      display: column.display,
      valueType: cellValueType(column),
      values: [row[index]],
    })),
  ).map((values) => values[0]);
}

/**
 * Text for values of one column, formatted on demand and kept by value. A chart knows most of
 * the values it will label up front (every point) and passes them to be formatted in one
 * call; what it asks for later (the axis ticks Recharts chooses) is formatted the first time
 * each distinct value is asked for, and kept.
 */
export function createValueFormatter(
  rules: RulesRuntime,
  column: ReportCellColumn,
  known: readonly unknown[] = [],
): (value: unknown) => string {
  const cache = new Map<unknown, string>();
  const valueType = cellValueType(column);

  const fill = (values: readonly unknown[]) => {
    const missing = [...new Set(values)].filter((value) => !cache.has(value));
    if (missing.length === 0) return;
    const [cells] = formatReportColumns(rules, [
      { display: column.display, valueType, values: missing },
    ]);
    missing.forEach((value, index) => {
      cache.set(value, reportCellText(cells[index], column.display));
    });
  };

  fill(known);
  return (value) => {
    fill([value]);
    return cache.get(value) ?? "";
  };
}
