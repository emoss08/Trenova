import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  decimalStringSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const fleetCodeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  name: z.string().min(1, "Name is required"),
  description: optionalStringSchema,
  revenueGoal: decimalStringSchema,
  deadheadGoal: decimalStringSchema,
  color: optionalStringSchema,
  managerId: nullableStringSchema,
  // TODO(wolfred): We need to add the manager field to the schema once we add the user schema
});

export type FleetCodeSchema = z.infer<typeof fleetCodeSchema>;
