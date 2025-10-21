import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { userSchema } from "./user-schema";

export const fleetCodeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  status: z.enum(Status),
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

export type FleetCodeSchema = z.infer<typeof fleetCodeSchema>;
