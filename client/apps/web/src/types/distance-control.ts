import { z } from "zod";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";

export const distanceControlSchema = z.object({
  ...tenantInfoSchema.shape,
  storeMileage: z.boolean(),
  storedDistanceUnits: z.enum(["Miles", "Kilometers"]),
  postalCodeFallbackToCity: z.boolean(),
  autoCreateStoredMileage: z.boolean(),
  loadedMoveDistanceProfileId: z.string().min(1),
  emptyMoveDistanceProfileId: z.string().min(1),
  payDistanceProfileId: z.string().min(1),
  billingDistanceProfileId: z.string().min(1),
  fuelDistanceProfileId: z.string().min(1),
  etaOutOfRouteDistanceProfileId: z.string().min(1),
  distanceCalculatorShortestDistanceProfileId: z.string().min(1),
  distanceCalculatorPracticalDistanceProfileId: z.string().min(1),
});

export type DistanceControl = z.infer<typeof distanceControlSchema>;
