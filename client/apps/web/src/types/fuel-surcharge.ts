import { z } from "zod";
import { decimalStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const fuelIndexSourceSchema = z.enum(["EIA", "Custom"]);
export type FuelIndexSource = z.infer<typeof fuelIndexSourceSchema>;

export const fuelTypeSchema = z.enum(["Diesel", "Gasoline"]);
export type FuelType = z.infer<typeof fuelTypeSchema>;

export const fuelSurchargeProgramMethodSchema = z.enum([
  "PerMileStep",
  "PerMileMPG",
  "TablePerMile",
  "TablePercent",
  "TableFlat",
]);
export type FuelSurchargeProgramMethod = z.infer<typeof fuelSurchargeProgramMethodSchema>;

export const fuelSurchargePercentBasisSchema = z.enum(["Linehaul", "LinehaulPlusAccessorials"]);
export type FuelSurchargePercentBasis = z.infer<typeof fuelSurchargePercentBasisSchema>;

export const fuelSurchargeDateBasisSchema = z.enum(["PickupDate", "TenderDate"]);
export type FuelSurchargeDateBasis = z.infer<typeof fuelSurchargeDateBasisSchema>;

export const fuelSurchargeStepRoundingSchema = z.enum(["Up", "Down", "Nearest"]);
export type FuelSurchargeStepRounding = z.infer<typeof fuelSurchargeStepRoundingSchema>;

export const fuelSurchargeRateRoundingSchema = z.enum(["HalfUp", "Up", "Down"]);
export type FuelSurchargeRateRounding = z.infer<typeof fuelSurchargeRateRoundingSchema>;

export const fuelSurchargeFallbackSchema = z.enum(["UseLatestAvailable", "Skip"]);
export type FuelSurchargeFallback = z.infer<typeof fuelSurchargeFallbackSchema>;

export const fuelSurchargeProgramStatusSchema = z.enum(["Active", "Inactive"]);
export type FuelSurchargeProgramStatus = z.infer<typeof fuelSurchargeProgramStatusSchema>;

export const fuelIndexSchema = z.object({
  ...tenantInfoSchema.shape,
  name: z
    .string()
    .min(1, { message: "Name is required" })
    .max(100, { message: "Name must be less than 100 characters" }),
  code: z
    .string()
    .min(1, { message: "Code is required" })
    .max(50, { message: "Code must be less than 50 characters" }),
  description: optionalStringSchema,
  source: fuelIndexSourceSchema,
  fuelType: fuelTypeSchema.default("Diesel"),
  region: optionalStringSchema,
  eiaSeriesId: optionalStringSchema,
  currency: z
    .string()
    .length(3, { message: "Currency must be 3 characters" })
    .default("USD"),
  isActive: z.boolean().default(true),
});

export type FuelIndex = z.infer<typeof fuelIndexSchema>;

export const fuelSurchargeTableRowSchema = z.object({
  id: optionalStringSchema,
  fuelSurchargeProgramId: optionalStringSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  priceMin: decimalStringSchema,
  priceMax: decimalStringSchema,
  value: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) return undefined;
      const parsed = parseFloat(
        typeof val === "string" || typeof val === "number" ? String(val) : "",
      );
      return isNaN(parsed) ? undefined : parsed;
    },
    z.number({ message: "Value is required" }),
  ),
  sortOrder: z.number().int().min(0).default(0),
});

export type FuelSurchargeTableRowValues = z.infer<typeof fuelSurchargeTableRowSchema>;

const TABLE_METHODS = new Set(["TablePerMile", "TablePercent", "TableFlat"]);

export const fuelSurchargeProgramSchema = z
  .object({
    ...tenantInfoSchema.shape,
    name: z
      .string()
      .min(1, { message: "Name is required" })
      .max(100, { message: "Name must be less than 100 characters" }),
    code: z
      .string()
      .min(1, { message: "Code is required" })
      .max(50, { message: "Code must be less than 50 characters" }),
    description: optionalStringSchema,
    status: fuelSurchargeProgramStatusSchema.default("Active"),
    fuelIndexId: z.string().min(1, { message: "Fuel index is required" }),
    accessorialChargeId: z.string().min(1, { message: "Accessorial charge is required" }),
    method: fuelSurchargeProgramMethodSchema,
    pegPrice: decimalStringSchema,
    increment: decimalStringSchema,
    incrementRate: decimalStringSchema,
    milesPerGallon: decimalStringSchema,
    percentBasis: fuelSurchargePercentBasisSchema.default("Linehaul"),
    stepRounding: fuelSurchargeStepRoundingSchema.default("Up"),
    rateRounding: fuelSurchargeRateRoundingSchema.default("HalfUp"),
    ratePrecision: z.number().int().min(0).max(6).default(4),
    minAmount: decimalStringSchema,
    maxAmount: decimalStringSchema,
    dateBasis: fuelSurchargeDateBasisSchema.default("PickupDate"),
    priceEffectiveDay: z.number().int().min(0).max(6).default(3),
    missingPriceFallback: fuelSurchargeFallbackSchema.default("UseLatestAvailable"),
    effectiveStartDate: z.number().int().nullish(),
    effectiveEndDate: z.number().int().nullish(),
    shipmentTypeIds: z.array(z.string()).nullish(),
    serviceTypeIds: z.array(z.string()).nullish(),
    tractorTypeIds: z.array(z.string()).nullish(),
    trailerTypeIds: z.array(z.string()).nullish(),
    tableRows: z.array(fuelSurchargeTableRowSchema).default([]),
  })
  .superRefine((data, ctx) => {
    const requirePositive = (field: keyof typeof data, label: string) => {
      const value = data[field];
      if (value === null || value === undefined || Number(value) <= 0) {
        ctx.addIssue({
          code: "custom",
          path: [field],
          message: `${label} is required and must be greater than zero`,
        });
      }
    };

    if (data.method === "PerMileStep") {
      if (data.pegPrice === null || data.pegPrice === undefined || Number(data.pegPrice) < 0) {
        ctx.addIssue({
          code: "custom",
          path: ["pegPrice"],
          message: "Peg price is required and must not be negative",
        });
      }
      requirePositive("increment", "Increment");
      requirePositive("incrementRate", "Rate per increment");
    }

    if (data.method === "PerMileMPG") {
      if (data.pegPrice === null || data.pegPrice === undefined || Number(data.pegPrice) < 0) {
        ctx.addIssue({
          code: "custom",
          path: ["pegPrice"],
          message: "Peg price is required and must not be negative",
        });
      }
      requirePositive("milesPerGallon", "Miles per gallon");
    }

    if (TABLE_METHODS.has(data.method)) {
      if (data.tableRows.length === 0) {
        ctx.addIssue({
          code: "custom",
          path: ["tableRows"],
          message: "At least one price band is required for table-based methods",
        });
      }

      const sorted = data.tableRows
        .map((row, index) => ({ row, index }))
        .filter(({ row }) => row.priceMin !== null && row.priceMin !== undefined)
        .sort((a, b) => Number(a.row.priceMin) - Number(b.row.priceMin));

      data.tableRows.forEach((row, index) => {
        if (
          row.priceMin !== null &&
          row.priceMin !== undefined &&
          row.priceMax !== null &&
          row.priceMax !== undefined &&
          Number(row.priceMax) <= Number(row.priceMin)
        ) {
          ctx.addIssue({
            code: "custom",
            path: ["tableRows", index, "priceMax"],
            message: "Price max must be greater than price min",
          });
        }
      });

      for (let i = 1; i < sorted.length; i++) {
        const prev = sorted[i - 1].row;
        const curr = sorted[i].row;
        if (
          prev.priceMax === null ||
          prev.priceMax === undefined ||
          Number(curr.priceMin) < Number(prev.priceMax)
        ) {
          ctx.addIssue({
            code: "custom",
            path: ["tableRows", sorted[i].index, "priceMin"],
            message: "Price bands must not overlap",
          });
        }
      }
    }

    if (
      data.minAmount !== null &&
      data.minAmount !== undefined &&
      data.maxAmount !== null &&
      data.maxAmount !== undefined &&
      Number(data.minAmount) > Number(data.maxAmount)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["minAmount"],
        message: "Minimum amount must not exceed maximum amount",
      });
    }
  })
  .transform((data) => ({
    ...data,
    tableRows: data.tableRows.map((row, index) => ({
      ...row,
      sortOrder: index,
    })),
  }));

export type FuelSurchargeProgram = z.infer<typeof fuelSurchargeProgramSchema>;
export type FuelSurchargeProgramFormValues = FuelSurchargeProgram;
