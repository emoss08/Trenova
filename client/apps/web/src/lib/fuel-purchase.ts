import { toUserWallClock } from "@trenova/shared/lib/date";
import { decimalPattern, multiplyDecimalStrings } from "@trenova/shared/types/decimal";
import { FUEL_AMOUNT_SCALE } from "@trenova/shared/types/fuel-purchase";

const LOOSE_DECIMAL = decimalPattern(10);

function decimalOrNull(value: string | null | undefined): string | null {
  const trimmed = value?.trim() ?? "";
  return LOOSE_DECIMAL.test(trimmed) ? trimmed : null;
}

export function computeFuelTotal(
  quantity: string | null | undefined,
  unitPrice: string | null | undefined,
): string | null {
  const q = decimalOrNull(quantity);
  const p = decimalOrNull(unitPrice);
  if (q === null || p === null) return null;
  return multiplyDecimalStrings(q, p, FUEL_AMOUNT_SCALE);
}

export function purchaseQuarterLabel(purchasedAt: number, timezone?: string): string {
  const local = toUserWallClock(purchasedAt, timezone) ?? new Date(purchasedAt * 1000);
  return `Q${Math.floor(local.getMonth() / 3) + 1} ${local.getFullYear()}`;
}
