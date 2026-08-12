import { z } from "zod";

/** Mirrors the reason length check in Override.Validate. */
export const MIN_OVERRIDE_REASON_LENGTH = 10;

const MAX_LEAD_TIME_DAYS = 60;

/**
 * A carrier's stricter posture on one jurisdiction.
 *
 * Every limit is nullable and nil means "defer to the statutory rule", so the
 * schema has to keep absent distinguishable from a value — a coerced zero would
 * read as an override of zero feet.
 *
 * The schema does not check a limit against the statute, because the statute is
 * not in scope here. The server rejects a looser override with a field error,
 * and the engine refuses to apply one regardless.
 */
export const jurisdictionRuleOverrideSchema = z
  .object({
    id: z.string().optional(),
    stateId: z.string().min(1, { message: "State is required" }),

    maxWidthFeet: z.number().positive().nullish(),
    maxHeightFeet: z.number().positive().nullish(),
    maxLengthFeet: z.number().positive().nullish(),
    maxWeightPounds: z.number().int().positive().nullish(),

    permitLeadTimeDays: z
      .number()
      .int()
      .min(0)
      .max(MAX_LEAD_TIME_DAYS, { message: "Lead time must be 60 days or fewer" })
      .nullish(),

    daylightOnly: z.boolean().nullish(),
    holidayRestricted: z.boolean().nullish(),

    reason: z.string().min(MIN_OVERRIDE_REASON_LENGTH, {
      message: "Explain why this jurisdiction is overridden, at least 10 characters",
    }),

    version: z.number().int().optional(),
    createdAt: z.number().int().optional(),
    updatedAt: z.number().int().optional(),

    state: z.custom<{ id: string; name: string; abbreviation: string }>().nullish(),
  })
  // Mirrors Override.HasAnyOverride. A reason attached to nothing is not an
  // override, and saving one would leave a row that changes no behaviour.
  .refine(
    (v) =>
      v.maxWidthFeet != null ||
      v.maxHeightFeet != null ||
      v.maxLengthFeet != null ||
      v.maxWeightPounds != null ||
      v.permitLeadTimeDays != null ||
      v.daylightOnly != null ||
      v.holidayRestricted != null,
    {
      message: "An override must change at least one limit or restriction",
      path: ["stateId"],
    },
  );

export type JurisdictionRuleOverride = z.infer<typeof jurisdictionRuleOverrideSchema>;
