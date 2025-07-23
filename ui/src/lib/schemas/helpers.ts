/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
  .nullish();

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
  .nullish();

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
  .nullish();

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
export const optionalNullableString = z.string().nullish();

/**
 * Helper for Go pointer fields that might come as null or undefined
 * Makes the field optional and nullable
 */
export const optionalIntegerSchema = z.number().int().nullish();

/**
 * Helper for Go PULID fields (always strings, never null in practice)
 */
export const pulidSchema = z.string().min(1);

/**
 * Helper for Go nullable PULID fields (*pulid.ID)
 */
export const nullablePulidSchema = z
  .union([z.string().min(1), z.null()])
  .nullish();

/**
 * Helper for Go timestamps (Unix timestamps as int64)
 */
export const timestampSchema = z.number().int().positive().optional();

/**
 * Helper for nullable Go timestamps
 */
export const nullableTimestampSchema = z
  .union([z.number().int().positive(), z.null()])
  .nullish();

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
  (val) => val === null || (val !== undefined && val >= -100 && val <= 200),
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
