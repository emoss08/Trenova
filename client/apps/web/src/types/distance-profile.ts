import { z } from "zod";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";

export const distanceProfileSchema = z.object({
  ...tenantInfoSchema.shape,
  name: z.string().min(1, { error: "Name is required" }),
  description: z.string().optional(),
  status: z.enum(["Active", "Inactive"]),
  isDefault: z.boolean(),
  provider: z.enum(["PCMiler"]),
  dataVersion: z.string().min(1, { error: "Data version is required" }),
  region: z.enum(["NA"]),
  routingType: z.string().min(1, { error: "Routing type is required" }),
  distanceUnits: z.string().min(1, { error: "Distance units are required" }),
  locationGranularity: z.string().min(1, { error: "Location granularity is required" }),
  profileName: z.string().optional(),
  highwayOnly: z.boolean(),
  tollRoads: z.boolean(),
  bordersOpen: z.boolean(),
  includeTollData: z.boolean(),
});

export type DistanceProfile = z.infer<typeof distanceProfileSchema>;
