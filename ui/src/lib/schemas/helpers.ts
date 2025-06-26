import * as z from "zod/v4";

// Helper to handle decimal strings that come from the backend

/**
 * Helper to handle decimal strings that come from Go backends using shopspring/decimal
 * Converts string decimals to numbers, handles empty strings as null
 */
export const decimalStringSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseFloat(val))),
    z.number(),
    z.null(),
  ])
  .nullable();

/**
 * Helper to handle optional strings from Go backends
 * Converts empty strings to null for proper nullable handling
 */
export const optionalStringSchema = z.string().optional();

/**
 * Helper to handle nullable strings from Go backends
 * Converts empty strings to null for proper nullable handling
 */
export const nullableStringSchema = z
  .union([z.string().transform((val) => (val === "" ? null : val)), z.null()])
  .optional()
  .nullable();

/**
 * Helper to handle nullable integers from Go backends
 * Handles string numbers, empty strings as null, and ensures integer conversion
 */
export const nullableIntegerSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseInt(val, 10))),
    z.number().int(),
    z.null(),
  ])
  .optional()
  .nullable();

/**
 * Helper to handle nullable big integers (int64) from Go backends
 * Handles string numbers, empty strings as null, and ensures integer conversion
 */
export const nullableBigIntegerSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseInt(val, 10))),
    z.number().int(),
    z.null(),
  ])
  .nullable();

/**
 * Helper for Go pointer fields that might come as null or undefined
 * Makes the field optional and nullable
 */
export const optionalNullableString = z.string().nullable().optional();

/**
 * Helper for Go pointer fields that might come as null or undefined
 * Makes the field optional and nullable
 */
export const optionalIntegerSchema = z.number().int().nullable().optional();

/**
 * Helper for Go PULID fields (always strings, never null in practice)
 */
export const pulidSchema = z.string().min(1);

/**
 * Helper for Go nullable PULID fields (*pulid.ID)
 */
export const nullablePulidSchema = z
  .union([z.string().min(1), z.null()])
  .nullable()
  .optional();

/**
 * Helper for Go timestamps (Unix timestamps as int64)
 */
export const timestampSchema = z.number().int().positive().optional();

/**
 * Helper for nullable Go timestamps
 */
export const nullableTimestampSchema = z
  .union([z.number().int().positive(), z.null()])
  .nullable()
  .optional();

/**
 * Helper for Go boolean fields that might come as strings
 */
export const booleanStringSchema = z
  .union([z.boolean(), z.string().transform((val) => val === "true"), z.null()])
  .nullable();

/**
 * Helper for temperature validation (Fahrenheit range)
 */
export const temperatureSchema = nullableIntegerSchema.refine(
  (val) => val === null || (val >= -100 && val <= 200),
  { message: "Temperature must be between -100 and 200 degrees Fahrenheit" },
);

/**
 * Create an enum schema from Go constants
 */
export const createEnumSchema = <T extends readonly [string, ...string[]]>(
  values: T,
) => z.enum(values);

/**
 * Helper for Go version fields (always int64, never null)
 */
export const versionSchema = z.number().int().min(0).optional();
