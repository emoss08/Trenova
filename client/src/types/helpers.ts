import { z } from "zod";

export const decimalStringSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseFloat(val))),
    z.number(),
    z.null(),
  ])
  .nullish();
export const optionalStringSchema = z.string().optional();
export const versionSchema = z.number().int().min(0).optional();
export const timestampSchema = z.number().int().positive().optional();
export const nullableStringSchema = z
  .union([z.string().transform((val) => (val === "" ? null : val)), z.null()])
  .nullish();

export const tenantInfoSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
});

export type TenantInfo = z.infer<typeof tenantInfoSchema>;

export const statusSchema = z.enum(["Active", "Inactive"]);

export type Status = z.infer<typeof statusSchema>;

export const equipmentStatusSchema = z.enum(["Available", "OutOfService", "AtMaintenance", "Sold"]);
export type EquipmentStatus = z.infer<typeof equipmentStatusSchema>;

export const nullableIntegerSchema = z
  .union([
    z.string().transform((val) => (val === "" ? null : parseInt(val, 10))),
    z.number().int(),
    z.null(),
  ])
  .nullish();

export type NullableInteger = z.infer<typeof nullableIntegerSchema>;
