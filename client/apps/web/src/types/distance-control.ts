import { translate } from "@trenova/shared/i18n/runtime";
import { tenantInfoSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";

/** `RatingDistanceUnit` in distancecontrol.go: the only two values `Validate` accepts. */
export const distanceControlUnitSchema = z.enum(["Miles", "Kilometers"]);

export type DistanceControlUnit = z.infer<typeof distanceControlUnitSchema>;

/**
 * The Go entity `distancecontrol.DistanceControl`, field for field, as the server sends it. A
 * profile assignment reads as any string, empty included, so a record made before the
 * organization had a distance profile still opens; the form refuses to save it empty
 * (`distanceControlFormSchema`).
 */
export const distanceControlSchema = z.object({
  ...tenantInfoSchema.shape,
  storeMileage: z.boolean(),
  ratingDistanceUnit: distanceControlUnitSchema,
  postalCodeFallbackToCity: z.boolean(),
  autoCreateStoredMileage: z.boolean(),
  captureJurisdictionMiles: z.boolean(),
  loadedMoveDistanceProfileId: z.string(),
  emptyMoveDistanceProfileId: z.string(),
  payDistanceProfileId: z.string(),
  billingDistanceProfileId: z.string(),
  fuelDistanceProfileId: z.string(),
  etaOutOfRouteDistanceProfileId: z.string(),
});

export type DistanceControl = z.infer<typeof distanceControlSchema>;

export type DistanceProfileAssignment = {
  [K in keyof DistanceControl]-?: K extends `${string}DistanceProfileId` ? K : never;
}[keyof DistanceControl];

/** Each assignment's refusal, in the words of `DistanceControl.Validate`. */
const ASSIGNMENT_REQUIRED: Record<DistanceProfileAssignment, () => string> = {
  loadedMoveDistanceProfileId: () => translate("Loaded move distance profile is required"),
  emptyMoveDistanceProfileId: () => translate("Empty move distance profile is required"),
  payDistanceProfileId: () => translate("Pay distance profile is required"),
  billingDistanceProfileId: () => translate("Billing distance profile is required"),
  fuelDistanceProfileId: () => translate("Fuel distance profile is required"),
  etaOutOfRouteDistanceProfileId: () => translate("ETA/out-of-route distance profile is required"),
};

/**
 * The record as the form checks it before a save, following the server's validator: every
 * profile assignment is required, always. Whether a chosen profile is still an active PC*Miler
 * profile only the server can tell.
 */
export const distanceControlFormSchema = distanceControlSchema.superRefine((values, ctx) => {
  for (const [path, message] of Object.entries(ASSIGNMENT_REQUIRED) as [
    DistanceProfileAssignment,
    () => string,
  ][]) {
    if (!values[path]?.trim()) ctx.addIssue({ code: "custom", path: [path], message: message() });
  }
});
