import type { ReportRunDeskCard, ReportRunDeskParameter } from "@/lib/graphql/report-desk-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { translateLabel } from "@trenova/shared/i18n/labels";
import type { ReportPreviewArtifact } from "../../artifact-payloads";
import { sheetColumn, type SheetHints, type SheetSource } from "./report-sheet";

/** A run's stored sample as the sheet reads it; null when the run kept none. */
export function sourceOfRun(card: ReportRunDeskCard): SheetSource | null {
  if (!card.sample) {
    return null;
  }

  return {
    columns: card.sample.columns.map((column) => sheetColumn(column)),
    rows: card.sample.rows,
    totals: card.sample.totals,
    rowCount: card.run.rowCount,
    more: card.sample.more || card.run.rowCount > card.sample.rows.length,
    truncated: card.run.truncated,
  };
}

/** A preview's rows as the sheet reads them: every row it read is in the payload. */
export function sourceOfPreview(preview: ReportPreviewArtifact): SheetSource {
  return {
    columns: preview.columns.map((column) =>
      sheetColumn({ id: column.id, label: column.label, type: column.type, format: column.format }),
    ),
    rows: preview.rows,
    totals: preview.totals,
    rowCount: preview.rowCount,
    more: preview.rowCount > preview.rows.length,
    truncated: preview.truncated,
  };
}

export function hintsOf(card: ReportRunDeskCard): SheetHints {
  return { chart: card.chart, sort: card.sort };
}

/** Whether a run is still on its way. */
export function isRunActive(status: string): boolean {
  return status === "queued" || status === "running";
}

/** Where the report a run or preview came from opens in Reports; empty for an unsaved definition. */
export function reportsPath(source: { definitionId?: string | null; cannedKey?: string | null }) {
  if (source.definitionId) {
    return `/reports/explore/${source.definitionId}`;
  }
  if (source.cannedKey) {
    return `/reports/explore?canned=${encodeURIComponent(source.cannedKey)}`;
  }
  return "";
}

function dayFormat(timeZone: string | undefined, withYear: boolean) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    ...(withYear ? { year: "numeric" } : {}),
    ...(timeZone ? { timeZone } : {}),
  });
}

function yearOf(seconds: number, timeZone: string | undefined): string {
  return new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    ...(timeZone ? { timeZone } : {}),
  }).format(new Date(seconds * 1000));
}

/**
 * A window a parameter filtered on, as days: its first day to its last. The
 * window's end is exclusive (midnight after its last day), so the last day is
 * the second before it.
 */
export function windowText(
  start: number,
  end: number,
  t: TranslateFn,
  options: { timeZone?: string; withYear?: boolean } = {},
): string {
  const last = Math.max(start, end - 1);
  const crossesYear = yearOf(start, options.timeZone) !== yearOf(last, options.timeZone);
  const first = dayFormat(options.timeZone, crossesYear).format(new Date(start * 1000));
  const final = dayFormat(options.timeZone, crossesYear || options.withYear === true).format(
    new Date(last * 1000),
  );
  return first === final ? final : t("{0} – {1}", first, final);
}

const MAX_LISTED_VALUES = 3;

/** What a parameter was set to, as its chip says it, with the window it resolved to. */
export function parameterText(
  parameter: ReportRunDeskParameter,
  t: TranslateFn,
  timeZone?: string,
): string {
  const value = valueText(parameter, t);
  if (parameter.windowStart == null || parameter.windowEnd == null) {
    return value;
  }
  const window = windowText(parameter.windowStart, parameter.windowEnd, t, { timeZone });
  return value === "" ? window : t("{0} · {1}", value, window);
}

function valueText(parameter: ReportRunDeskParameter, t: TranslateFn): string {
  const value = parameter.value;
  if (value === null || value === undefined || value === "") {
    return t("Any");
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return t("Any");
    }
    if (parameter.refEntity) {
      return t("{0, plural, one {# selected} other {# selected}}", value.length);
    }
    const named = value.slice(0, MAX_LISTED_VALUES).map((item) => translateLabel(String(item)));
    return value.length > MAX_LISTED_VALUES
      ? t("{0} and {1} more", named.join(", "), value.length - MAX_LISTED_VALUES)
      : named.join(", ");
  }
  if (typeof value === "boolean") {
    return value ? t("Yes") : t("No");
  }
  if (typeof value === "number") {
    return value.toLocaleString();
  }
  if (parameter.refEntity) {
    return t("{0, plural, one {# selected} other {# selected}}", 1);
  }
  return translateLabel(String(value));
}

/** The window the run's sheet covers: the first parameter that resolved to one. */
export function sheetWindow(card: ReportRunDeskCard): { start: number; end: number } | null {
  const windowed = card.parameters.find(
    (parameter) => parameter.windowStart != null && parameter.windowEnd != null,
  );
  return windowed ? { start: windowed.windowStart ?? 0, end: windowed.windowEnd ?? 0 } : null;
}

/** How long a run took, as the stamp says it: "3.2 s", "1 min 4 s". */
export function durationText(ms: number, t: TranslateFn): string {
  if (ms < 60_000) {
    return t("{0} s", (ms / 1000).toFixed(1));
  }
  const seconds = Math.round(ms / 1000);
  return t("{0} min {1} s", Math.floor(seconds / 60), seconds % 60);
}
