import type { AccessorialChargeMethod } from "@trenova/shared/types/accessorial-charge";
import type {
  BillingSplitSummaryRow,
  ChargeAllocation,
  ChargeAllocationMethod,
  Shipment,
} from "@trenova/shared/types/shipment";

/** Money in cents; matches the server's two-decimal rounding of every share. */
export function toMinorUnits(value: number): number {
  return Math.round(value * 100);
}

function fromMinorUnits(minor: number): number {
  return minor / 100;
}

/** Banker's rounding, which is how the server rounds every share to cents. */
function roundHalfEven(value: number): number {
  const floor = Math.floor(value);
  const diff = value - floor;
  if (Math.abs(diff - 0.5) < 1e-9) {
    return floor % 2 === 0 ? floor : floor + 1;
  }
  return Math.round(value);
}

function toNumber(value: number | string | null | undefined): number {
  if (value == null || value === "") return 0;
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

type ChargeLike = {
  method?: AccessorialChargeMethod | string | null;
  amount?: number | string | null;
  unit?: number | null;
};

/**
 * What one accessorial charge comes to, the way the server computes it: flat and
 * per-unit charges multiply by their unit count, a percentage charge is that
 * percent of the freight it is based on.
 */
export function chargeLineTotal(charge: ChargeLike, freightBasis: number = 0): number {
  const amount = toNumber(charge.amount);
  const unit = charge.unit ?? 1;
  switch (charge.method) {
    case "PerUnit":
      return amount * Math.max(unit, 1);
    case "Flat":
      return amount * Math.max(unit, 1);
    case "Percentage":
      return (freightBasis * amount) / 100;
    default:
      return amount;
  }
}

export type AllocationLike = Pick<ChargeAllocation, "billToCustomerId" | "method"> & {
  percent?: number | string | null;
  amount?: number | string | null;
  sequence?: number | null;
};

export type SplitShare = {
  billToCustomerId: string;
  /** Cents; the shares always sum back to the charge exactly. */
  amountMinor: number;
  amount: number;
  percent: number | null;
  partial: boolean;
};

function ordered<T extends AllocationLike>(allocations: readonly T[]): T[] {
  return allocations
    .map((allocation, index) => ({ allocation, index }))
    .sort((a, b) => {
      const left = a.allocation.sequence ?? 0;
      const right = b.allocation.sequence ?? 0;
      return left === right ? a.index - b.index : left - right;
    })
    .map(({ allocation }) => allocation);
}

/**
 * Divides one charge among its allocations exactly as the server does: percent
 * shares are rounded to cents with the remainder folded into the last row by
 * sequence, amount shares pass through. No allocations means the whole charge
 * goes to the default payer.
 */
export function splitCharges(
  lineTotal: number,
  allocations: readonly AllocationLike[],
  defaultPayerId: string,
): SplitShare[] {
  const totalMinor = toMinorUnits(lineTotal);
  const rows = ordered(allocations).filter((row) => row.billToCustomerId);
  if (rows.length === 0) {
    return [
      {
        billToCustomerId: defaultPayerId,
        amountMinor: totalMinor,
        amount: fromMinorUnits(totalMinor),
        percent: 100,
        partial: false,
      },
    ];
  }

  const method = rows[0].method;
  const shares: SplitShare[] = [];
  let allocated = 0;
  rows.forEach((row, index) => {
    let minor: number;
    if (method === "Amount") {
      minor = toMinorUnits(toNumber(row.amount));
    } else if (index === rows.length - 1) {
      minor = totalMinor - allocated;
    } else {
      minor = roundHalfEven((totalMinor * toNumber(row.percent)) / 100);
    }
    allocated += minor;
    const percent =
      method === "Amount"
        ? totalMinor === 0
          ? null
          : Number(((minor * 100) / totalMinor).toFixed(6))
        : toNumber(row.percent);
    shares.push({
      billToCustomerId: row.billToCustomerId,
      amountMinor: minor,
      amount: fromMinorUnits(minor),
      percent,
      partial: minor !== totalMinor,
    });
  });

  return shares;
}

export type AllocationRemainder = {
  method: ChargeAllocationMethod;
  /** Percent points, or currency units, already assigned. */
  allocated: number;
  /** What is left to reach 100% or the charge total; negative when over. */
  remaining: number;
  isComplete: boolean;
  isOver: boolean;
  mixedMethods: boolean;
  duplicatePayers: boolean;
  missingPayers: boolean;
};

/**
 * Where a split stands while somebody is still typing it. Percent rows are
 * measured against 100, amount rows against the charge total, both in the
 * smallest unit so `33.33 + 33.33 + 33.34` is exactly complete.
 */
export function allocationRemainder(
  allocations: readonly AllocationLike[],
  lineTotal: number | null,
): AllocationRemainder {
  const rows = allocations.filter((row) => row != null);
  const method: ChargeAllocationMethod = rows[0]?.method === "Amount" ? "Amount" : "Percent";
  const mixedMethods = rows.some((row) => row.method !== method);
  const payers = rows.map((row) => row.billToCustomerId).filter(Boolean);
  const duplicatePayers = new Set(payers).size !== payers.length;
  const missingPayers = rows.some((row) => !row.billToCustomerId);

  if (method === "Amount") {
    const allocatedMinor = rows.reduce((sum, row) => sum + toMinorUnits(toNumber(row.amount)), 0);
    const targetMinor = toMinorUnits(lineTotal ?? 0);
    const remainingMinor = targetMinor - allocatedMinor;
    return {
      method,
      allocated: fromMinorUnits(allocatedMinor),
      remaining: fromMinorUnits(remainingMinor),
      isComplete: lineTotal != null && remainingMinor === 0 && rows.length > 0,
      isOver: remainingMinor < 0,
      mixedMethods,
      duplicatePayers,
      missingPayers,
    };
  }

  const allocatedHundredths = rows.reduce(
    (sum, row) => sum + Math.round(toNumber(row.percent) * 10000),
    0,
  );
  const remainingHundredths = 1000000 - allocatedHundredths;
  return {
    method,
    allocated: allocatedHundredths / 10000,
    remaining: remainingHundredths / 10000,
    isComplete: remainingHundredths === 0 && rows.length > 0,
    isOver: remainingHundredths < 0,
    mixedMethods,
    duplicatePayers,
    missingPayers,
  };
}

/**
 * Whether a set of rows is a legal split of a charge: one method, every row
 * naming a distinct payer with a positive share, and the rows adding up.
 * An empty list is legal — it bills the whole charge to the default payer.
 */
export function allocationsValid(
  allocations: readonly AllocationLike[],
  lineTotal: number | null,
): boolean {
  if (allocations.length === 0) return true;
  const state = allocationRemainder(allocations, lineTotal);
  if (state.mixedMethods || state.duplicatePayers || state.missingPayers) return false;
  const positive = allocations.every((row) =>
    state.method === "Amount" ? toNumber(row.amount) > 0 : toNumber(row.percent) > 0,
  );
  if (!positive) return false;
  return state.isComplete;
}

export type PayerLabelResolver = (payerId: string) => { name: string; code: string } | null;

type SummaryInput = {
  customerId: string;
  billToCustomerId?: string | null;
  freightChargeAmount?: number | string | null;
  freightAllocations?: readonly AllocationLike[] | null;
  additionalCharges?: readonly (ChargeLike & { allocations?: readonly AllocationLike[] | null })[];
};

/**
 * What each payer owes once every allocation is applied, mirroring the server's
 * billingSplitSummary so the form shows the invoices a save would produce. The
 * shipment's own payer leads.
 */
export function summarizeBillingByPayer(
  input: SummaryInput,
  resolvePayer: PayerLabelResolver = () => null,
): BillingSplitSummaryRow[] {
  const defaultPayerId = input.billToCustomerId || input.customerId;
  const freight = toNumber(input.freightChargeAmount);
  const totals = new Map<string, { freightMinor: number; accessorialMinor: number }>();
  const order: string[] = [];
  let isSplit = false;

  const add = (payerId: string, kind: "freight" | "accessorial", minor: number) => {
    let row = totals.get(payerId);
    if (!row) {
      row = { freightMinor: 0, accessorialMinor: 0 };
      totals.set(payerId, row);
      order.push(payerId);
    }
    if (kind === "freight") row.freightMinor += minor;
    else row.accessorialMinor += minor;
  };

  add(defaultPayerId, "freight", 0);
  for (const share of splitCharges(freight, input.freightAllocations ?? [], defaultPayerId)) {
    add(share.billToCustomerId, "freight", share.amountMinor);
    if (share.partial || share.billToCustomerId !== defaultPayerId) isSplit = true;
  }
  for (const charge of input.additionalCharges ?? []) {
    const lineTotal = chargeLineTotal(charge, freight);
    for (const share of splitCharges(lineTotal, charge.allocations ?? [], defaultPayerId)) {
      add(share.billToCustomerId, "accessorial", share.amountMinor);
      if (share.partial || share.billToCustomerId !== defaultPayerId) isSplit = true;
    }
  }

  const others = order.filter((id) => id !== defaultPayerId).sort();
  return [defaultPayerId, ...others].map((payerId) => {
    const row = totals.get(payerId)!;
    const label = resolvePayer(payerId);
    return {
      payerId,
      payerName: label?.name ?? "",
      payerCode: label?.code ?? "",
      isPrimary: payerId === defaultPayerId,
      freightAmount: fromMinorUnits(row.freightMinor),
      accessorialAmount: fromMinorUnits(row.accessorialMinor),
      totalAmount: fromMinorUnits(row.freightMinor + row.accessorialMinor),
      isSplit,
    };
  });
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

/**
 * One row as the API takes it: numbers become decimal strings, the client-only
 * customer snapshot is dropped, and sequence follows list position so the
 * remainder always lands on the last row shown.
 */
export function toChargeAllocationInput(
  allocation: AllocationLike & { id?: string | null; version?: number | null },
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
