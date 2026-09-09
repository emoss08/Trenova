import { z } from "zod";
import {
  nonNegativeDecimalString,
  optionalNonNegativeDecimalString,
  positiveDecimalString,
} from "./decimal";
import { LAST_FOUR_PATTERN } from "./fuel-card";
import {
  fuelCardProviderSchema,
  fuelQuantityUnitSchema,
  iftaFuelTypeSchema,
} from "./fuel-ifta-enums";

export const FUEL_QUANTITY_SCALE = 3;
export const FUEL_UNIT_PRICE_SCALE = 4;
export const FUEL_AMOUNT_SCALE = 2;

const CURRENCY_PATTERN = /^[A-Z]{3}$/;

const optionalText = (max: number, what: string) =>
  z
    .string()
    .max(max, { message: `Keep ${what} under ${max} characters` })
    .nullable();

const optionalLastFour = z
  .string()
  .nullable()
  .transform((value) => {
    const trimmed = value?.trim() ?? "";
    return trimmed === "" ? null : trimmed;
  })
  .pipe(
    z
      .string()
      .regex(LAST_FOUR_PATTERN, { message: "Enter the last four digits of the card" })
      .nullable(),
  );

export function createFuelPurchaseFormSchema(now: number) {
  return z.object({
    tractorId: z.string().min(1, { message: "Choose the tractor that was fuelled" }),
    workerId: z.string().nullable(),
    jurisdictionId: z
      .string()
      .min(1, { message: "Choose the jurisdiction the fuel was bought in" }),
    purchasedAt: z
      .number()
      .int()
      .positive({ message: "Enter when the fuel was bought" })
      .max(now, { message: "A purchase cannot be in the future" }),
    vendor: optionalText(200, "the vendor"),
    vendorCity: optionalText(100, "the city"),
    fuelType: iftaFuelTypeSchema,
    quantity: positiveDecimalString(
      FUEL_QUANTITY_SCALE,
      "Enter a quantity above zero with up to three decimals",
    ),
    quantityUnit: fuelQuantityUnitSchema,
    unitPrice: optionalNonNegativeDecimalString(
      FUEL_UNIT_PRICE_SCALE,
      "Enter a price with up to four decimals",
    ),
    totalAmount: nonNegativeDecimalString(
      FUEL_AMOUNT_SCALE,
      "Enter the amount paid with up to two decimals",
    ),
    currencyCode: z
      .string()
      .trim()
      .toUpperCase()
      .regex(CURRENCY_PATTERN, { message: "Use a three-letter currency code" }),
    odometer: z
      .number()
      .int({ message: "Odometer is whole miles" })
      .min(0, { message: "Odometer cannot be negative" })
      .nullable(),
    fuelCardId: z.string().nullable(),
    cardLastFour: optionalLastFour,
    transactionReference: optionalText(100, "the reference"),
    taxPaid: z.boolean(),
    notes: optionalText(2000, "notes"),
  });
}

export type FuelPurchaseFormValues = z.infer<ReturnType<typeof createFuelPurchaseFormSchema>>;

export const fuelPurchaseImportSetupSchema = z.object({
  provider: fuelCardProviderSchema,
  defaultFuelCardId: z.string().nullable(),
  defaultFuelType: iftaFuelTypeSchema.nullable(),
  defaultCurrency: z
    .string()
    .trim()
    .toUpperCase()
    .regex(CURRENCY_PATTERN, { message: "Use a three-letter currency code" }),
});

export type FuelPurchaseImportSetupValues = z.infer<typeof fuelPurchaseImportSetupSchema>;
