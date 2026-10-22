/**
 * Shaping charge allocations between the form and GraphQL, and money in cents. None of this
 * is an allocation rule: what a charge comes to, how it divides between payers and whether a
 * split adds up are answered by the rules module (`@trenova/shared/rules/allocation`).
 */
import type {
  ChargeAllocation,
  ChargeAllocationMethod,
  Shipment,
} from "@trenova/shared/types/shipment";

/**
 * Money in cents, for comparing and sending an amount a person typed with at most two places
 * (a credit memo application, an invoice line). Not for dividing a charge: that is the module's.
 */
export function toMinorUnits(value: number): number {
  return Math.round(value * 100);
}

function toNumber(value: number | string | null | undefined): number {
  if (value == null || value === "") return 0;
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

type NestableShipment = Pick<Shipment, "chargeAllocations"> & {
  additionalCharges: Shipment["additionalCharges"];
};

/**
 * Regroups the flat allocation list the server returns under the charge each
 * row splits, which is the shape the form edits: freight rows on the shipment,
 * accessorial rows on their charge.
 */
export function nestChargeAllocations<T extends NestableShipment>(
  shipment: T,
): T & { freightAllocations: ChargeAllocation[] } {
  const rows = shipment.chargeAllocations ?? [];
  const freightAllocations = rows.filter((row) => row.chargeKind === "Freight");
  const byCharge = new Map<string, ChargeAllocation[]>();
  for (const row of rows) {
    if (row.chargeKind !== "Accessorial" || !row.additionalChargeId) continue;
    const list = byCharge.get(row.additionalChargeId) ?? [];
    list.push(row);
    byCharge.set(row.additionalChargeId, list);
  }
  const additionalCharges = (shipment.additionalCharges ?? []).map((charge) => ({
    ...charge,
    allocations: charge.id ? (byCharge.get(charge.id) ?? []) : (charge.allocations ?? []),
  }));

  return { ...shipment, freightAllocations, additionalCharges };
}

export type ChargeAllocationWireInput = {
  id?: string | null;
  billToCustomerId: string;
  method: ChargeAllocationMethod;
  percent?: string | null;
  amount?: string | null;
  sequence?: number | null;
  version?: number | null;
};

type AllocationLike = Pick<ChargeAllocation, "billToCustomerId" | "method"> & {
  id?: string | null;
  percent?: number | string | null;
  amount?: number | string | null;
  sequence?: number | null;
  version?: number | null;
};

/**
 * One row as the API takes it: numbers become decimal strings, the client-only
 * customer snapshot is dropped, and sequence follows list position so the
 * remainder always lands on the last row shown.
 */
export function toChargeAllocationInput(
  allocation: AllocationLike,
  index: number,
): ChargeAllocationWireInput {
  const percent = allocation.method === "Percent" ? toNumber(allocation.percent) : null;
  const amount = allocation.method === "Amount" ? toNumber(allocation.amount) : null;
  return {
    id: allocation.id || undefined,
    billToCustomerId: allocation.billToCustomerId,
    method: allocation.method,
    percent: percent == null ? null : String(percent),
    amount: amount == null ? null : String(amount),
    sequence: index,
    version: allocation.version ?? undefined,
  };
}

/**
 * Whether a save should carry allocations at all. When nothing was ever loaded
 * or edited the payload omits them and the server leaves the split alone.
 */
export function hasAnyAllocations(input: {
  freightAllocations?: readonly unknown[] | null;
  additionalCharges?: readonly { allocations?: readonly unknown[] | null }[] | null;
}): boolean {
  if ((input.freightAllocations?.length ?? 0) > 0) return true;
  return (input.additionalCharges ?? []).some((charge) => (charge.allocations?.length ?? 0) > 0);
}
