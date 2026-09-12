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
