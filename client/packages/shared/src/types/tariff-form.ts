// tariff-form.ts holds the Tariff shapes a record that a form edits carries: a rate agreement
// rule's and agreement's input values, a shipment's input values and its last bill.
//
// react-hook-form's path types walk every value of a form's record recursively, and the wire
// value type is recursive (a list of records of lists…), so a form holding one never finishes
// type-checking. These schemas validate exactly what `./tariff` validates but give the form a
// type its paths can walk: input values as `unknown` (`TariffInputsFields` owns the fields
// beneath them), and a bill whose quantities and rates are the flat numbers and quantities a
// bill line actually holds. The server stores a bill in its own shape (`tarifftypes.Bill`:
// decimals with their unit text beside them); `storedTariffBillSchema` reads it into the wire
// form every bill view draws.

import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import {
  TARIFF_DECIMAL_PATTERN,
  tariffChoiceSchema,
  tariffInputValuesSchema,
  type TariffBillLine,
  type TariffChoice,
  type TariffWireNumber,
  type TariffWireQuantity,
} from "./tariff";

function withoutUnset(value: unknown): unknown {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return value;
  return Object.fromEntries(Object.entries(value).filter(([, entry]) => entry !== undefined));
}

/**
 * A tariff's input values as a form holds them. An input a form leaves unset holds
 * `undefined`, which is dropped here (as JSON would drop it), so the record carries only what
 * this level sets and every other input keeps inheriting.
 */
export const tariffFormInputValuesSchema = z.preprocess(
  withoutUnset,
  tariffInputValuesSchema,
) as unknown as z.ZodType<Record<string, unknown>, Record<string, unknown>>;
export type TariffFormInputValues = Record<string, unknown>;

/** A bill line's quantity or rate: a count or a quantity with its unit, never a list or record. */
export type TariffLineMeasure = TariffWireNumber | TariffWireQuantity;

/** A bill line as the server stores it, read as a `TariffBillLine` with its measures named. */
export type StoredTariffBillLine = {
  kind: TariffBillLine["kind"];
  label: string | null;
  code: string | null;
  freight: boolean;
  amount: TariffWireQuantity;
  quantity: TariffLineMeasure | null;
  rate: TariffLineMeasure | null;
  range: null;
};

/**
 * A bill as the server stores it (a shipment's rating detail, a quote's trace, a contract
 * preview), read in the module's wire form so `BillView` draws it like any other bill, with the
 * rating currency it was billed in and the exchange to the billing currency when there was one.
 */
export type StoredTariffBill = {
  lines: StoredTariffBillLine[];
  notes: { text: string; range: null }[];
  freightTotal: TariffWireQuantity;
  accessorialTotal: TariffWireQuantity;
  total: TariffWireQuantity;
  converted: null;
  currency: string;
  exchange: { currency: string; rate: string } | null;
};

/** Exact decimal text as Go's decimal marshals it: a JSON string, or a number from a fixture. */
const storedDecimalSchema = z
  .union([z.string(), z.number()])
  .transform((value) => String(value))
  .refine((text) => TARIFF_DECIMAL_PATTERN.test(text), {
    error: () => translate("Must be a decimal"),
  });

const storedLineSchema = z.object({
  kind: z.enum(["Charge", "Minimum", "Maximum", "Rounding"]),
  label: z.string().nullish(),
  code: z.string().nullish(),
  freight: z.boolean().optional(),
  amount: storedDecimalSchema,
  quantity: storedDecimalSchema.nullish(),
  quantityUnit: z.string().nullish(),
  rate: storedDecimalSchema.nullish(),
  rateUnit: z.string().nullish(),
});

/**
 * The unit a rate implies its quantity is measured in: `$/mi` is per mile. A rate per item
 * (`$`) or per a counted thing (`$/stop`, `$/pallet`, `$/piece`: money with the count folded
 * in) bills a plain count.
 */
export function quantityUnitOfRate(rateUnit: string | null | undefined): string | null {
  if (!rateUnit) return null;
  const slash = rateUnit.indexOf("/");
  if (slash === -1) return null;
  const unit = rateUnit.slice(slash + 1);
  return unit === "stop" || unit === "pallet" || unit === "piece" ? null : unit;
}

function storedLine(line: z.output<typeof storedLineSchema>): StoredTariffBillLine {
  const quantityUnit = line.quantityUnit || quantityUnitOfRate(line.rateUnit);
  const measured = line.quantity != null && line.rate != null;
  return {
    kind: line.kind.toLowerCase() as TariffBillLine["kind"],
    label: line.label || null,
    code: line.code || null,
    freight: line.freight ?? !line.code,
    amount: { $q: line.amount, u: "$" },
    quantity: measured
      ? quantityUnit
        ? { $q: line.quantity as string, u: quantityUnit }
        : { $n: line.quantity as string }
      : null,
    rate: measured ? { $q: line.rate as string, u: line.rateUnit || "$" } : null,
    range: null,
  };
}

const quantitySchema = z.object({ $q: z.string(), u: z.string() });
const measureSchema = z.union([z.object({ $n: z.string() }), quantitySchema]).nullable();

/**
 * A stored bill already read into wire form. Parsing is idempotent: a record the client has
 * parsed once (a form reset from a parsed response) parses again unchanged.
 */
const readBillSchema = z.object({
  currency: z.string(),
  lines: z.array(
    z.object({
      kind: z.enum(["charge", "minimum", "maximum", "rounding"]),
      label: z.string().nullable(),
      code: z.string().nullable(),
      freight: z.boolean(),
      amount: quantitySchema,
      quantity: measureSchema,
      rate: measureSchema,
      range: z.null(),
    }),
  ),
  notes: z.array(z.object({ text: z.string(), range: z.null() })),
  freightTotal: quantitySchema,
  accessorialTotal: quantitySchema,
  total: quantitySchema,
  converted: z.null(),
  exchange: z.object({ currency: z.string(), rate: z.string() }).nullable(),
}) satisfies z.ZodType<StoredTariffBill>;

/**
 * Reads a stored bill into the wire form the bill views draw: as REST stores it
 * (`tarifftypes.Bill`, notes as `{ text }`, each line flagged `freight`) or as GraphQL reads it
 * (`TariffBill`, notes as text, a line freight when it has no code).
 */
const rawStoredBillSchema = z
  .object({
    currency: z.string().default(""),
    lines: z.array(storedLineSchema).default([]),
    notes: z
      .array(z.union([z.string(), z.object({ text: z.string() })]))
      .nullish()
      .transform((value) =>
        (value ?? []).map((note) => (typeof note === "string" ? note : note.text)),
      ),
    freightTotal: storedDecimalSchema,
    accessorialTotal: storedDecimalSchema,
    total: storedDecimalSchema,
    exchange: z
      .object({ currency: z.string(), rate: storedDecimalSchema })
      .nullish()
      .transform((value) => value ?? null),
  })
  .transform((bill): StoredTariffBill => ({
    currency: bill.currency,
    lines: bill.lines.map(storedLine),
    notes: bill.notes.map((text) => ({ text, range: null })),
    freightTotal: { $q: bill.freightTotal, u: "$" },
    accessorialTotal: { $q: bill.accessorialTotal, u: "$" },
    total: { $q: bill.total, u: "$" },
    converted: null,
    exchange: bill.exchange,
  }));

export const storedTariffBillSchema: z.ZodType<StoredTariffBill, unknown> = z.union([
  readBillSchema,
  rawStoredBillSchema,
]);

/**
 * A bill a record carries, read whether it arrived as the server stores it or already read
 * (GraphQL reads reach a form unparsed); null when there is none or it is not a bill.
 */
export function readStoredTariffBill(value: unknown): StoredTariffBill | null {
  if (value === null || value === undefined) return null;
  const parsed = storedTariffBillSchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

const storedChoiceOptionSchema = z.object({
  label: z.string(),
  status: z.string().transform((value) => value.toLowerCase()),
  subtotal: storedDecimalSchema.nullish(),
  refusal: z.string().nullish(),
});

/**
 * A choice as the server stores it on a rating (`tarifftypes.Choice`, GraphQL `TariffChoice`):
 * each option's subtotal a decimal in the rating currency and its refusal already in the
 * reader's words. Read into the module's wire form, so one view draws a preview's choices and a
 * shipment's alike.
 */
const rawStoredChoiceSchema = z
  .object({
    label: z.string(),
    kind: z.string().transform((value) => value.toLowerCase()),
    chosen: z.string().nullish(),
    options: z.array(storedChoiceOptionSchema).default([]),
  })
  .transform((choice): TariffChoice => ({
    label: choice.label,
    kind: choice.kind === "worst" ? "worst" : "best",
    range: null,
    chosen: choice.chosen || null,
    options: choice.options.map((option) => ({
      label: option.label,
      range: null,
      status: (["chosen", "lost", "refused"] as const).includes(
        option.status as TariffChoice["options"][number]["status"],
      )
        ? (option.status as TariffChoice["options"][number]["status"])
        : "lost",
      subtotal: option.subtotal == null ? null : { $q: option.subtotal, u: "$" },
      refusal: option.refusal ? { message: option.refusal, range: null } : null,
    })),
  }));

/** Every choice a stored rating made, read whether it arrived stored or already read. */
export const storedTariffChoicesSchema: z.ZodType<TariffChoice[], unknown> = z
  .array(z.union([tariffChoiceSchema, rawStoredChoiceSchema]))
  .nullish()
  .transform((value) => value ?? []);

/** The choices a record's rating detail carries; empty when there are none or they do not read. */
export function readStoredTariffChoices(value: unknown): TariffChoice[] {
  const parsed = storedTariffChoicesSchema.safeParse(value);
  return parsed.success ? parsed.data : [];
}
