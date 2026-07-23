export const DEFAULT_TARGET_MARGIN_PCT = 10;

export type MarginTone = "danger" | "warning" | "success";

export function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function getMarginTone(
  marginPct: number,
  targetPct: number = DEFAULT_TARGET_MARGIN_PCT,
): MarginTone {
  if (marginPct < 0) return "danger";
  if (marginPct < targetPct) return "warning";
  return "success";
}

export type CompactProfitabilityEstimate = {
  shipmentId: string;
  loadedMiles: number;
  deadheadMiles: number;
  totalMiles: number;
  costPerMile: string;
  estimatedCost: string;
  profit: string;
  marginPercent?: string | null;
  breakEvenRpm?: string | null;
  targetMarginPercent?: string | null;
  missingDistance: boolean;
};

export function resolveTargetMarginPct(
  targetMarginPercent: string | null | undefined,
): number {
  if (targetMarginPercent === null || targetMarginPercent === undefined) {
    return DEFAULT_TARGET_MARGIN_PCT;
  }
  const parsed = Number(targetMarginPercent);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : DEFAULT_TARGET_MARGIN_PCT;
}
