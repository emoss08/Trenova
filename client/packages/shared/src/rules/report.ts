// Typed calls for the rules module's report formatting operations (`report.*`): how a report
// value is rounded, grouped, banded, toned and written, and how a column's display is resolved
// from its catalog type, hint and the author's overrides. The rules are in
// docs/engineering/rules-engine.md, "Report formatting"; testdata/report.json pins them.
//
// `report.format` works on columns: a caller sends every value it is about to show (a page of
// grid rows, every point of a chart) in one call, never one value at a time. A number in a date
// column comes back as epoch seconds for the browser to write with `Intl` in the report's time
// zone (`reportCellText`), since the module has no time zone data.

import {
  formatReportDate,
  type ReportBand,
  type ReportColumnDisplay,
  type ReportColumnDisplayLike,
  type ReportDateStyle,
  type ReportDisplayRule,
  type ReportDisplayTone,
} from "@trenova/shared/lib/report-format";
import type { RulesRuntime } from "./runtime";
import type { DecimalString } from "./wire";

/** The author's overrides of a column's display (`reportfmt.Spec`); every field optional. */
export type ReportDisplaySpecInput = {
  readonly style?: string;
  readonly decimals?: number;
  readonly grouping?: boolean;
  readonly currency?: string;
  readonly negative?: string;
  readonly notation?: string;
  readonly prefix?: string;
  readonly suffix?: string;
  readonly dateStyle?: string;
  readonly boolStyle?: string;
  readonly durationUnit?: string;
  readonly durationStyle?: string;
  readonly nullText?: string;
  readonly rules?: readonly ReportDisplayRule[] | null;
};

/** What a column's display is resolved from (`reportfmt.ResolveInput`). */
export type ReportResolveInput = {
  /** The catalog value type: string, int, decimal, epoch, bool, … */
  readonly type: string;
  /** The catalog format hint: money, weight, percent, duration, distance, count. */
  readonly hint?: string;
  /** The date style a dimension's date bucket implies. */
  readonly dateStyle?: ReportDateStyle;
  readonly band?: ReportBand | null;
  readonly spec?: ReportDisplaySpecInput | null;
};

/** A resolved display as `report.resolve` writes it: band bounds and rule values are exact. */
export type ResolvedReportDisplay = ReportColumnDisplay & {
  band?: { width?: DecimalString; edges?: DecimalString[] } | null;
  rules?: { op: string; value: DecimalString; upper?: DecimalString; tone: string }[] | null;
};

/** One column to format: its display (or what to resolve it from), value type and values. */
export type ReportFormatColumn = {
  /** The resolved display; a missing display renders values raw. */
  readonly display?: ReportColumnDisplayLike | null;
  /** Resolves the display in the same call instead. */
  readonly resolve?: ReportResolveInput;
  /** The catalog value type, so decimals that GraphQL sent as strings read as numbers. */
  readonly valueType?: string;
  readonly values: readonly unknown[];
};

/** One formatted value: its text, or a date for the browser to write, and its tone. */
export type ReportCell =
  | { readonly text: string; readonly tone: ReportDisplayTone }
  | { readonly date: DecimalString; readonly tone: ReportDisplayTone };

type WireCell = { text?: string; date?: DecimalString; tone?: ReportDisplayTone };

function wireValue(value: unknown): unknown {
  return value === undefined ? null : value;
}

/** `report.format`: every value of every column, in request order. */
export function formatReportColumns(
  rules: RulesRuntime,
  columns: readonly ReportFormatColumn[],
): ReportCell[][] {
  const output = rules.call("report.format", {
    columns: columns.map((column) => ({
      // A column with no display is sent without one, which renders its values raw.
      ...(column.resolve
        ? { resolve: column.resolve }
        : column.display
          ? { display: column.display }
          : {}),
      valueType: column.valueType ?? "",
      values: column.values.map(wireValue),
    })),
  }) as { columns: { values: WireCell[] }[] };

  return output.columns.map((column) =>
    column.values.map((cell): ReportCell => {
      const tone = cell.tone ?? "";
      return cell.date === undefined ? { text: cell.text ?? "", tone } : { date: cell.date, tone };
    }),
  );
}

/** `report.resolve`: one resolved display per column, in request order. */
export function resolveReportDisplays(
  rules: RulesRuntime,
  columns: readonly ReportResolveInput[],
): ResolvedReportDisplay[] {
  const output = rules.call("report.resolve", { columns }) as {
    displays: ResolvedReportDisplay[];
  };
  return output.displays;
}

/** The text of a formatted cell: its own text, or its date written in `timeZone`. */
export function reportCellText(
  cell: ReportCell,
  display: Pick<ReportColumnDisplayLike, "dateStyle"> | null | undefined,
  timeZone?: string,
): string {
  if ("text" in cell) return cell.text;
  return formatReportDate(Number(cell.date), display?.dateStyle ?? "", timeZone);
}
