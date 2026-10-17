import type { AnalysisValueKind } from "@/types/agent-extension";
import { dateTimeFormatter, formatNumber, numberFormatter } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatISODateMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";

const MISSING = "—";
const ISO_DATE_LENGTH = 10;

/** Kinds whose values line up on the right, as numbers do in a table. */
const NUMERIC_KINDS: ReadonlySet<AnalysisValueKind> = new Set([
  "number",
  "money",
  "count",
  "percent",
]);

export function isNumericKind(kind: AnalysisValueKind): boolean {
  return NUMERIC_KINDS.has(kind);
}

/**
 * An analysis value as a person reads it: money as money in its currency, a percent from its
 * fraction, a date in the reader's language. A date is read at the offset the analysis wrote it
 * with, so a day stays the organization's day; a value missing or of the wrong shape reads as a
 * dash.
 */
export function formatAnalysisValue(
  value: unknown,
  kind: AnalysisValueKind,
  t: TranslateFn,
  currency = "",
): string {
  if (value === null || value === undefined || value === "") {
    return MISSING;
  }

  if (typeof value === "number") {
    if (!Number.isFinite(value)) {
      return MISSING;
    }
    switch (kind) {
      case "money":
        return formatCurrency(value, currency || "USD");
      case "count":
        return formatNumber(value, { maximumFractionDigits: 0 });
      case "percent":
        return numberFormatter({ style: "percent", maximumFractionDigits: 1 }).format(value);
      default:
        return formatNumber(value, { maximumFractionDigits: Math.abs(value) >= 100 ? 0 : 2 });
    }
  }

  if (typeof value === "boolean") {
    return value ? t("Yes") : t("No");
  }

  const text = String(value);
  if (kind === "date") {
    return formatISODateMedium(text.slice(0, ISO_DATE_LENGTH), text);
  }
  if (kind === "datetime") {
    const at = new Date(text);
    return Number.isNaN(at.getTime())
      ? text
      : dateTimeFormatter({ dateStyle: "medium", timeStyle: "short" }).format(at);
  }

  return text;
}
