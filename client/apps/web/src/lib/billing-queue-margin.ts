import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";

/** Below this margin a shipment reads as thin when its profile sets no target. */
const DEFAULT_THIN_MARGIN_PERCENT = 10;

/**
 * A shipment's margin as the billing queue shows it: the dollars it keeps and the share
 * of revenue they are, "$842.50 · 22%", and whether that falls short of the profile's
 * target margin (or 10% without one). Null when the margin is not known.
 */
export function shipmentMargin(
  estimate:
    | {
        profit?: string | number | null;
        marginPercent?: string | number | null;
        profile?: { targetMarginPercent?: string | number | null } | null;
      }
    | null
    | undefined,
  t: TranslateFn,
): { text: string; thin: boolean } | null {
  if (!estimate || estimate.marginPercent == null || estimate.profit == null) {
    return null;
  }
  const percent = Number(estimate.marginPercent);
  const target = estimate.profile?.targetMarginPercent;

  return {
    text: t("{0} · {1}%", formatCurrency(Number(estimate.profit)), Math.round(percent)),
    thin: percent < (target != null ? Number(target) : DEFAULT_THIN_MARGIN_PERCENT),
  };
}
