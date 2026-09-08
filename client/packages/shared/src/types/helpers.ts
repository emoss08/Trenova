import { z } from "zod";

export const decimalStringSchema = z
  .union([
    z
      .string()
      .transform((val) => (val.trim() === "" ? null : Number(val)))
      .refine((val) => val === null || Number.isFinite(val), {
        error: "Must be a valid number",
      }),
    z.number().refine((val) => Number.isFinite(val), { error: "Must be a valid number" }),
    z.null(),
  ])
  .nullish();
export const optionalStringSchema = z.string().optional();

/**
 * Rate-style fields arrive as plain numbers from form inputs but as decimal
 * strings when prefilled from GraphQL, so both are accepted and normalized to
 * a number — mirroring the `decimalStringSchema` pattern.
 */
export const decimalNumberSchema = (requiredError: string, negativeError: string) =>
  z
    .union(
      [
        z
          .string()
          .transform((val) => (val.trim() === "" ? Number.NaN : Number(val)))
          .refine((val) => Number.isFinite(val), { error: requiredError }),
        z.number().refine((val) => Number.isFinite(val), { error: requiredError }),
      ],
      { error: requiredError },
    )
    .refine((val) => val >= 0, { error: negativeError });
export const nullableTextSchema = z
  .string()
  .nullish()
  .transform((value) => value ?? "");
export const versionSchema = z.number().int().min(0).optional();
export const timestampSchema = z.number().int().positive().optional();
// Server-managed audit timestamps. They are round-tripped through the form for
// display but stripped before submit, so they must never block validation — a
// record with an unset (0) timestamp still has to be editable.
export const auditTimestampSchema = z.number().int().min(0).optional();
export const nullableStringSchema = z
  .union([z.string().transform((val) => (val === "" ? null : val)), z.null()])
  .nullish();
/**
 * Optional enum columns are empty strings on the wire, not nulls: Go zero-values
 * the column and both transports serialize that zero value verbatim (gqlgen
 * marshals a non-pointer enum straight through, so even a nullable GraphQL enum
 * field arrives as ""). Normalize it to null ahead of the enum check so an unset
 * value never reads as an invalid option.
 */
export const nullableEnumSchema = <T extends z.ZodType>(schema: T) =>
  z.preprocess((value) => (value === "" ? null : value), schema.nullish());

export const stringArraySchema = z
  .array(z.string())
  .nullish()
  .transform((value) => value ?? []);

export const nullableArraySchema = <T extends z.ZodTypeAny>(schema: T) =>
  z
    .array(schema)
    .nullish()
    .transform((value) => value ?? []);

export const tenantInfoSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: auditTimestampSchema,
  updatedAt: auditTimestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
});

export type TenantInfo = z.infer<typeof tenantInfoSchema>;

export const statusSchema = z.enum(["Active", "Inactive"]);

export type Status = z.infer<typeof statusSchema>;

export const equipmentStatusSchema = z.enum(["Available", "OutOfService", "AtMaintenance", "Sold"]);
export type EquipmentStatus = z.infer<typeof equipmentStatusSchema>;

/**
 * Organization capability flags gate what the UI *shows*, never what the API
 * allows. An older payload that predates a flag — or omits it from a narrow
 * projection — must never hide a feature, so an absent or null value fails open
 * to enabled.
 */
export const capabilityFlagSchema = z
  .boolean()
  .nullish()
  .transform((value) => value ?? true);

export const nullableIntegerSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseInt(val, 10))),
    z.number().int(),
    z.null(),
  ])
  .nullish();

export type NullableInteger = z.infer<typeof nullableIntegerSchema>;

/**
 * A relation hanging off another record is a display projection, not an
 * editable subform: a query selects the columns it renders and leaves the rest
 * out. Validating such a relation against the owning entity's full schema fails
 * every row the API actually returns.
 *
 * Omission reaches the client two different ways. GraphQL drops an unselected
 * field entirely, but REST marshals the whole Go struct, so a column the
 * repository never selected arrives as its zero value — `""` for a string or
 * enum, `null` for a pointer — rather than absent. A plain `.partial()` only
 * covers the first case and still trips `min(1)` and enum checks on the second,
 * so zero values are stripped before the partial schema sees them.
 */
export const relationSchema = <T extends z.ZodObject<z.ZodRawShape>>(schema: T) =>
  z.preprocess((value) => {
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      return value;
    }

    const projected: Record<string, unknown> = {};
    for (const [key, columnValue] of Object.entries(value as Record<string, unknown>)) {
      if (columnValue !== null && columnValue !== "") {
        projected[key] = columnValue;
      }
    }

    return projected;
  }, schema.partial().nullish());
