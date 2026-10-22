// Typed calls for the rules module's charge allocation operations (`allocation.*`): what a
// charge comes to, how it divides between payers, and where a split stands while somebody
// types it. The wire shapes and every rule are in docs/engineering/rules-engine.md, "Charge
// allocation"; testdata/allocation.json pins them.
//
// Each call takes every charge of one shipment, order or form at once. Amounts come back as
// exact decimal strings; a screen converts one to a number only to hand it to a formatter.

import type { CatalogMessage } from "@trenova/shared/i18n/message";
import type { RulesRuntime } from "./runtime";
import {
  type DecimalInput,
  type DecimalString,
  wireDecimal,
  wireInteger,
  wireList,
  wireText,
} from "./wire";

export type AllocationMethod = "Percent" | "Amount";

/** Which payer subtotal a charge's shares join. */
export type AllocationChargeKind = "Freight" | "Accessorial" | "OrderCharge";

/** One allocation row as a form or GraphQL holds it; other fields are ignored. */
export type AllocationRowInput = {
  readonly id?: string | null;
  readonly billToCustomerId?: string | null;
  readonly method?: string | null;
  readonly percent?: DecimalInput;
  readonly amount?: DecimalInput;
  readonly sequence?: number | null;
};

/**
 * One charge. Its total is `total` when given (freight, an order charge), otherwise its
 * accessorial line (`method`, `amount`, `unit`) priced against the request's `base`. A charge
 * with neither has an unknown total, which only `allocationRemainders` accepts.
 */
export type AllocationChargeInput = {
  readonly kind?: AllocationChargeKind;
  readonly total?: DecimalInput;
  readonly method?: string | null;
  readonly amount?: DecimalInput;
  readonly unit?: number | null;
  readonly allocations?: readonly (AllocationRowInput | null | undefined)[] | null;
};

export type AllocationChargesInput = {
  /** What a `Percentage` charge is a percent of: the shipment's freight. Zero when absent. */
  readonly base?: DecimalInput;
  readonly charges: readonly AllocationChargeInput[];
};

export type AllocationSplitInput = AllocationChargesInput & {
  /** Who pays a charge with no allocation rows. */
  readonly defaultPayerId: string;
};

/** One payer's slice of one charge. */
export type AllocationShare = {
  /** The row's index in the charge's `allocations`; absent for an unallocated charge. */
  readonly allocation?: number;
  readonly allocationId?: string;
  readonly billToCustomerId: string;
  /** How the charge was split; null when it was billed whole. */
  readonly method: AllocationMethod | null;
  readonly amount: DecimalString;
  /** Null for an amount share of a zero charge. */
  readonly percent: DecimalString | null;
  /** Whether the share is not the whole rounded charge. */
  readonly partial: boolean;
};

/** One charge divided, or the reason it cannot be. */
export type AllocationChargeSplit = {
  readonly total: DecimalString;
  readonly shares?: readonly AllocationShare[];
  readonly error?: CatalogMessage;
};

/** What one payer owes across every charge. */
export type AllocationPayerTotal = {
  readonly payerId: string;
  readonly freightAmount: DecimalString;
  readonly accessorialAmount: DecimalString;
  readonly totalAmount: DecimalString;
};

export type AllocationSplit = {
  readonly charges: readonly AllocationChargeSplit[];
  /** The default payer first, then the others by ID; null when any charge failed. */
  readonly payers: readonly AllocationPayerTotal[] | null;
  /** Null when any charge failed. */
  readonly isSplit: boolean | null;
};

/** Where one charge's split stands. */
export type AllocationRemainder = {
  readonly method: AllocationMethod;
  /** Percent points or currency already assigned, exact. */
  readonly allocated: DecimalString;
  /** What is left; negative when over. Null for amount rows against an unknown total. */
  readonly remaining: DecimalString | null;
  readonly complete: boolean;
  readonly over: boolean;
  readonly mixedMethods: boolean;
  readonly duplicatePayers: boolean;
  readonly missingPayers: boolean;
  /** The first reason the rows cannot be saved. */
  readonly problem: CatalogMessage | null;
  readonly valid: boolean;
};

export type AllocationChargeTotals = {
  readonly totals: readonly DecimalString[];
  readonly sum: DecimalString;
};

function wireRow(row: AllocationRowInput | null | undefined) {
  if (row === null || row === undefined) return null;
  return {
    id: wireText(row.id) || null,
    billToCustomerId: wireText(row.billToCustomerId),
    method: wireText(row.method),
    percent: wireDecimal(row.percent),
    amount: wireDecimal(row.amount),
    sequence: wireInteger(row.sequence),
  };
}

function wireCharge(charge: AllocationChargeInput) {
  return {
    kind: charge.kind ?? null,
    total: wireDecimal(charge.total),
    method: wireText(charge.method),
    amount: wireDecimal(charge.amount),
    unit: wireInteger(charge.unit),
    allocations: wireList(charge.allocations)?.map(wireRow) ?? null,
  };
}

function wireCharges(input: AllocationChargesInput) {
  return { base: wireDecimal(input.base), charges: input.charges.map(wireCharge) };
}

/** `allocation.split`: every charge divided between its payers, and each payer's totals. */
export function splitAllocations(
  rules: RulesRuntime,
  input: AllocationSplitInput,
): AllocationSplit {
  return rules.call("allocation.split", {
    defaultPayerId: input.defaultPayerId,
    ...wireCharges(input),
  }) as AllocationSplit;
}

/** `allocation.remainder`: one state per charge, in request order. */
export function allocationRemainders(
  rules: RulesRuntime,
  input: AllocationChargesInput,
): readonly AllocationRemainder[] {
  const output = rules.call("allocation.remainder", wireCharges(input)) as {
    charges: AllocationRemainder[];
  };
  return output.charges;
}

/** `allocation.chargeTotal`: each charge's exact total, in request order, and their sum. */
export function chargeTotals(
  rules: RulesRuntime,
  input: AllocationChargesInput,
): AllocationChargeTotals {
  return rules.call("allocation.chargeTotal", wireCharges(input)) as AllocationChargeTotals;
}
