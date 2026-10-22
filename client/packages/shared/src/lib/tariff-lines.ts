// tariff-lines.ts writes the basis of a stored charge line the way the bill wrote it: the
// quantity it billed in its unit, times the rate in its unit (`812 mi × $2.35/mi`). A
// shipment's freight lines, a tariff's accessorial charges and an invoice's lines all store
// a quantity, a rate and their unit text beside the amount.

import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatTariffValue, type TariffValueFormatOptions } from "./tariff-value";
import { tariffDecimal } from "@trenova/shared/rules/tariff-wire";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import {
  quantityUnitOfRate,
  readStoredTariffBill,
  type StoredTariffBill,
} from "@trenova/shared/types/tariff-form";

export type StoredChargeLine = {
  quantity?: number | string | null;
  /** The quantity's unit (`mi`, `lb`), or none for a count. */
  quantityUnit?: string | null;
  rate?: number | string | null;
  /** The rate's unit as Tariff writes it (`$/mi`, `$/cwt`, `$` for a per-item amount). */
  rateUnit?: string | null;
};

function quantityValue(line: StoredChargeLine): TariffWireValue | null {
  const amount = tariffDecimal(line.quantity ?? null);
  if (amount === null) return null;
  const unit = line.quantityUnit || quantityUnitOfRate(line.rateUnit);
  return unit ? { $q: amount, u: unit } : { $n: amount };
}

function rateValue(line: StoredChargeLine): TariffWireValue | null {
  const amount = tariffDecimal(line.rate ?? null);
  if (amount === null) return null;
  return { $q: amount, u: line.rateUnit || "$" };
}

/**
 * `812 mi × $2.35/mi` for a line that billed a quantity at a rate, or null for a flat amount
 * (a line with no quantity or no rate).
 */
export function chargeLineBasis(
  line: StoredChargeLine,
  t: TranslateFn,
  options: TariffValueFormatOptions = {},
): string | null {
  const quantity = quantityValue(line);
  const rate = rateValue(line);
  if (quantity === null || rate === null) return null;
  return t(
    "{0} × {1}",
    formatTariffValue(quantity, t, options),
    formatTariffValue(rate, t, options),
  );
}

/** A stored line with its label and amount: a freight line, or a charge with its code. */
export type StoredBilledLine = StoredChargeLine & {
  label?: string | null;
  code?: string | null;
  amount?: number | string | null;
};

export type StoredBilledCharges = {
  freightLines: readonly StoredBilledLine[];
  charges: readonly StoredBilledLine[];
  freightAmount?: number | string | null;
  otherChargeAmount?: number | string | null;
  totalChargeAmount?: number | string | null;
};

/**
 * A bill drawn from what a rating stored on a shipment: its freight lines, then the charges
 * it added (each with its code), with the stored totals. A contract rate preview carries these
 * in place of its bill, so `BillView` draws it line by line like any other bill. Null when a
 * stored line is not one a bill could hold.
 */
export function storedChargesBill(
  stored: StoredBilledCharges,
  currency: string,
): StoredTariffBill | null {
  const line = (entry: StoredBilledLine, freight: boolean) => ({
    kind: "Charge",
    label: entry.label?.trim() || null,
    code: freight ? null : entry.code || null,
    freight,
    amount: tariffDecimal(entry.amount ?? null) ?? "0",
    quantity: tariffDecimal(entry.quantity ?? null),
    quantityUnit: entry.quantityUnit ?? null,
    rate: tariffDecimal(entry.rate ?? null),
    rateUnit: entry.rateUnit ?? null,
  });
  return readStoredTariffBill({
    currency,
    lines: [
      ...stored.freightLines.map((entry) => line(entry, true)),
      ...stored.charges.map((entry) => line(entry, false)),
    ],
    freightTotal: tariffDecimal(stored.freightAmount ?? null) ?? "0",
    accessorialTotal: tariffDecimal(stored.otherChargeAmount ?? null) ?? "0",
    total: tariffDecimal(stored.totalChargeAmount ?? null) ?? "0",
  });
}
