import { z } from "zod";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const distanceProfileSchema = z.object({
  ...tenantInfoSchema.shape,
  name: z.string().min(1, { error: () => translate("Name is required") }),
  description: z.string().optional(),
  status: z.enum(["Active", "Inactive"]),
  isDefault: z.boolean(),
  provider: z.enum(["PCMiler"]),
  dataVersion: z.string().min(1, { error: () => translate("Data version is required") }),
  region: z.enum(["NA"]),
  routingType: z.string().min(1, { error: () => translate("Routing type is required") }),
  distanceUnits: z.string().min(1, { error: () => translate("Distance units are required") }),
  locationGranularity: z
    .string()
    .min(1, { error: () => translate("Location granularity is required") }),
  profileName: z.string().optional(),
  highwayOnly: z.boolean(),
  tollRoads: z.boolean(),
  bordersOpen: z.boolean(),
  includeTollData: z.boolean(),
});

export type DistanceProfile = z.infer<typeof distanceProfileSchema>;
