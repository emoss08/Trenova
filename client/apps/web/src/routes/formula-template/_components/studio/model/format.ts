import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  FormulaRoundingMode,
  TestExpressionResponse,
} from "@trenova/shared/types/formula-template";

const numberFormat = new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 });

/** A value as the studio prints it in mono: numbers grouped, flags as words, text as typed. */
export function formatStudioValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "number") return numberFormat.format(value);
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "string") {
    const asNumber = Number(value);
    return value.trim() !== "" && Number.isFinite(asNumber) ? numberFormat.format(asNumber) : value;
  }
  return JSON.stringify(value);
}

/** Money at a given precision, for the places that show more than cents. */
export function formatMoney(amount: number, decimals = 2): string {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(amount);
}

export function numericAmount(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const amount = typeof value === "number" ? value : Number(value);
  return Number.isFinite(amount) ? amount : null;
}

/** The preview's amounts: what the formula produced, after the range, and what is billed. */
export type PreviewAmounts = {
  raw: number;
  clamped: number;
  billed: number;
  bound: "min" | "max" | null;
  limit: number | null;
  rounded: boolean;
};

export function previewAmounts(result: TestExpressionResponse | undefined): PreviewAmounts | null {
  if (!result?.valid) return null;
  const billed = numericAmount(result.result);
  if (billed === null) return null;
  const raw = result.receipt?.rawAmount ?? result.guardrail?.rawAmount ?? billed;
  const guardrail = result.guardrail;
  const bound = guardrail?.applied ? (guardrail.bound ?? null) : null;
  const limit =
    bound === "min"
      ? (guardrail?.minCharge ?? null)
      : bound === "max"
        ? (guardrail?.maxCharge ?? null)
        : null;
  const clamped = bound && limit !== null ? limit : (result.rounding?.unroundedAmount ?? raw);
  return {
    raw,
    clamped,
    billed,
    bound,
    limit,
    rounded: result.rounding?.applied ?? false,
  };
}

export function roundingLabel(mode: FormulaRoundingMode | undefined, t: TranslateFn) {
  switch (mode) {
    case "HalfEven":
      return t("half even");
    case "Up":
      return t("up");
    case "Down":
      return t("down");
    case "None":
      return t("not rounded");
    default:
      return t("half up");
  }
}

/** "Rounds half up to 2 places", or "Not rounded". */
export function roundingSentence(
  mode: FormulaRoundingMode | undefined,
  precision: number,
  t: TranslateFn,
): string {
  if (mode === "None") return t("Not rounded");
  return t(
    "Rounds {0} to {1, plural, one {# place} other {# places}}",
    roundingLabel(mode, t),
    precision,
  );
}
