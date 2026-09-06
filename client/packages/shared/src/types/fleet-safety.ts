import { z } from "zod";

export const csaBasicSchema = z.enum([
  "UnsafeDriving",
  "HOSCompliance",
  "DriverFitness",
  "ControlledSubstances",
  "VehicleMaintenance",
  "HazmatCompliance",
  "CrashIndicator",
]);
export type CSABasicValue = z.infer<typeof csaBasicSchema>;

/**
 * One violation cited on a safety event. The severity weight is the FMCSA's
 * own 1-to-10 scale, so it is bounded rather than free: a weight nobody could
 * look up against the published tables is not evidence of anything.
 */
export const safetyViolationFormSchema = z.object({
  basic: csaBasicSchema,
  code: z.string().max(20).nullable(),
  description: z.string().min(1, "Describe the violation").max(255),
  severityWeight: z
    .number()
    .int()
    .min(1, "Severity weight is 1 to 10")
    .max(10, "Severity weight is 1 to 10"),
  outOfService: z.boolean(),
});
export type SafetyViolationFormValues = z.infer<typeof safetyViolationFormSchema>;

export const driverDigestCadenceSchema = z.enum(["Immediate", "Daily", "Weekly"]);
export type DriverDigestCadence = z.infer<typeof driverDigestCadenceSchema>;
