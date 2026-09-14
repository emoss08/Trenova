import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { RateUnit } from "@trenova/shared/types/accessorial-charge";
import type { InvoiceLine } from "@trenova/shared/types/invoice";

/**
 * Key used for the trailing group holding lines with no shipment attribution —
 * order-level charges such as customs brokerage or an order-wide fuel surcharge.
 */
export const ORDER_CHARGES_GROUP_KEY = "__order_charges__";

export type InvoiceLineGroup = {
  /** `shipmentId`, or {@link ORDER_CHARGES_GROUP_KEY} for unattributed lines. */
  key: string;
  shipmentId: string | null;
  proNumber: string | null;
  bol: string | null;
  lines: InvoiceLine[];
  subtotal: number;
};

function normalize(value: string | null | undefined): string | null {
  if (value == null) return null;
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}

/**
 * Groups invoice lines by the shipment each was billed for.
 *
 * Shipments keep first-appearance order and their lines are ordered by line
 * number. Lines carrying no shipment are order-level charges and always land in
 * a single trailing group — never merged into the first shipment, never dropped.
 *
 * Subtotals are summed from the lines themselves rather than read off the
 * invoice header, so a group total always reconciles against the rows shown
 * beneath it.
 */
export function groupInvoiceLinesByShipment(lines: readonly InvoiceLine[]): InvoiceLineGroup[] {
  const groups = new Map<string, InvoiceLineGroup>();

  for (const line of lines) {
    const shipmentId = normalize(line.shipmentId);
    const key = shipmentId ?? ORDER_CHARGES_GROUP_KEY;

    let group = groups.get(key);
    if (!group) {
      group = {
        key,
        shipmentId,
        proNumber: normalize(line.shipmentProNumber),
        bol: normalize(line.shipmentBol),
        lines: [],
        subtotal: 0,
      };
      groups.set(key, group);
    }

    // A leg's identity may only be stamped on some of its lines, so keep the
    // first non-empty value seen rather than whichever line happened to be first.
    group.proNumber ??= normalize(line.shipmentProNumber);
    group.bol ??= normalize(line.shipmentBol);

    group.lines.push(line);
    group.subtotal += line.amount ?? 0;
  }

  const ordered = [...groups.values()];
  for (const group of ordered) {
    group.lines.sort((a, b) => a.lineNumber - b.lineNumber);
  }

  const orderCharges = ordered.findIndex((group) => group.key === ORDER_CHARGES_GROUP_KEY);
  if (orderCharges >= 0 && orderCharges < ordered.length - 1) {
    ordered.push(...ordered.splice(orderCharges, 1));
  }

  return ordered;
}

function toMinorUnits(value: number): number {
  return Math.round(value * 100);
}

/**
 * Whether a line's quantity and unit price say anything its amount does not.
 *
 * A flat charge is one unit priced at its own amount, so repeating `1 × $900.00`
 * beside `$900.00` is noise. Unit prices carry four decimal places while amounts
 * are money, so the comparison is made in cents.
 */
export function hasUnitBreakdown(line: InvoiceLine): boolean {
  if (line.quantity == null) return false;
  if (line.quantity !== 1) return true;
  return toMinorUnits(line.unitPrice ?? 0) !== toMinorUnits(line.amount ?? 0);
}

export type ChargeComposition = {
  freightShare: number;
  accessorialShare: number;
};

/**
 * How an invoice's charges divide between freight and accessorials, as shares of
 * their sum. A proportion only means something for non-negative amounts, so a
 * credit memo's negative lines, or an invoice with nothing on it, have none.
 */
export function chargeComposition(freight: number, accessorial: number): ChargeComposition | null {
  if (!Number.isFinite(freight) || !Number.isFinite(accessorial)) return null;
  if (freight < 0 || accessorial < 0) return null;

  const total = freight + accessorial;
  if (total <= 0) return null;

  return { freightShare: freight / total, accessorialShare: accessorial / total };
}

function formatQuantity(value: number): string {
  return formatNumber(value, { maximumFractionDigits: 4 });
}

function perUnitQuantity(t: TranslateFn, unit: RateUnit | null, quantity: number): string {
  switch (unit) {
    case "Mile":
      return t("{0, plural, one {# mile} other {# miles}}", quantity);
    case "Hour":
      return t("{0, plural, one {# hour} other {# hours}}", quantity);
    case "Day":
      return t("{0, plural, one {# day} other {# days}}", quantity);
    case "Stop":
      return t("{0, plural, one {# stop} other {# stops}}", quantity);
    default:
      return t("{0, plural, one {# unit} other {# units}}", quantity);
  }
}

/**
 * How a charge line's amount was reached, in the words the billing queue uses:
 * "$75.00 × 2 hours", "10% of line haul ($2,450.00)", "Flat rate".
 *
 * Lines written before invoices recorded the charge method only have their
 * quantity and unit price, so those fall back to "3 × $50.00" when it says more
 * than the amount, and to nothing when it does not.
 */
export function describeChargeCalculation(
  line: InvoiceLine,
  currencyCode: string,
  t: TranslateFn,
): string | null {
  const quantity = line.quantity ?? 1;
  const rate = line.rate ?? null;

  if (line.type === "Accessorial" && line.chargeMethod && rate !== null) {
    switch (line.chargeMethod) {
      case "PerUnit":
        return t(
          "{0} × {1}",
          formatCurrency(rate, currencyCode),
          perUnitQuantity(t, line.rateUnit ?? null, quantity),
        );
      case "Percentage":
        return line.rateBasisAmount != null
          ? t(
              "{0}% of line haul ({1})",
              formatQuantity(rate),
              formatCurrency(line.rateBasisAmount, currencyCode),
            )
          : t("{0}% of line haul", formatQuantity(rate));
      case "Flat":
        return quantity > 1
          ? t("{0} × {1}", formatCurrency(rate, currencyCode), formatQuantity(quantity))
          : t("Flat rate");
    }
  }

  if (!hasUnitBreakdown(line)) return null;

  return t(
    "{0} × {1}",
    formatQuantity(quantity),
    formatCurrency(line.unitPrice ?? 0, currencyCode),
  );
}
