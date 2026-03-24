import { z } from "zod";
import {
  decimalStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";
import { userSchema } from "./user";

export const fleetCodeSchema = z.object({
  ...tenantInfoSchema.shape,

  status: statusSchema,
  code: z
    .string({ error: "Code must be a string" })
    .min(1, { error: "Code is required" })
    .max(10, { error: "Code must be less than 10 characters" }),
  description: optionalStringSchema,
  revenueGoal: decimalStringSchema,
  deadheadGoal: decimalStringSchema,
  color: optionalStringSchema,
  managerId: z.string().min(1, {
    error: "Manager is required",
  }),
  manager: userSchema.nullish(),
});

export type FleetCode = z.infer<typeof fleetCodeSchema>;

export const bulkUpdateFleetCodeStatusRequestSchema = z.object({
  fleetCodeIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateFleetCodeStatusRequest = z.infer<
  typeof bulkUpdateFleetCodeStatusRequestSchema
>;

export const bulkUpdateFleetCodeStatusResponseSchema = z.array(fleetCodeSchema);

export type BulkUpdateFleetCodeStatusResponse = z.infer<
  typeof bulkUpdateFleetCodeStatusResponseSchema
>;
