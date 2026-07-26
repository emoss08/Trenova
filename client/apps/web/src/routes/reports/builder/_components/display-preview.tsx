import { resolveReportDisplay, type ReportColumnDisplay } from "@trenova/shared/lib/report-format";
import type { ReportColumnSpec, ReportDisplaySpec } from "@/types/report";

const BUCKET_DATE_STYLES: Record<string, ReportColumnDisplay["dateStyle"]> = {
  day: "date",
  week: "date",
  month: "monthYear",
  quarter: "quarter",
  year: "year",
};

/** The display contract the server will resolve for this column. */
export function resolvePreviewDisplay(
  valueType: string,
  formatHint: string | undefined,
  column: ReportColumnSpec,
  overrides: ReportDisplaySpec,
): ReportColumnDisplay {
  const countingMeasure = column.agg === "count" || column.agg === "count_distinct";

  return resolveReportDisplay({
    type: valueType,
    hint: countingMeasure ? "count" : formatHint,
    dateStyle: column.bucket ? BUCKET_DATE_STYLES[column.bucket] : undefined,
    overrides,
  });
}

/** A representative value so the editor can show a live sample. */
export function sampleValueFor(display: ReportColumnDisplay, valueType: string): unknown {
  switch (display.style) {
    case "date":
      return 1772587800;
    case "bool":
      return true;
    case "duration":
      return 9330;
    case "text":
      return valueType === "epoch" ? "1772587800" : "Sample";
    default:
      return valueType === "int" ? 1284 : 1082.0672643987443;
  }
}
