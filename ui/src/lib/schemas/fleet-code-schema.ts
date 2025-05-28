import { Status } from "@/types/common";
import { z } from "zod";

export const fleetCodeSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  revenueGoal: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseFloat(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().optional()),
  deadheadGoal: z.preprocess((val) => {
    if (val === "" || val === null || val === undefined) {
      return undefined;
    }
    const parsed = parseFloat(String(val));
    return isNaN(parsed) ? undefined : parsed;
  }, z.number().optional()),
  color: z.string().optional(),
  managerId: z.string().nullable(),
  // TODO(wolfred): We need to add the manager field to the schema once we add the user schema
});

export type FleetCodeSchema = z.infer<typeof fleetCodeSchema>;
