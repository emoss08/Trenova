import { z } from "zod";
import { decimalStringSchema, optionalStringSchema } from "@trenova/shared/types/helpers";

export const jurisdictionRuleStatusSchema = z.enum(["Active", "Inactive", "Draft"]);
export type JurisdictionRuleStatus = z.infer<typeof jurisdictionRuleStatusSchema>;

export const jurisdictionVerificationStateSchema = z.enum(["Unverified", "Verified", "Disputed"]);
export type JurisdictionVerificationState = z.infer<typeof jurisdictionVerificationStateSchema>;

/** Mirrors jurisdictionrule.MinSourceNoteLength on the server. */
export const MIN_SOURCE_NOTE_LENGTH = 10;

/**
 * Mirrors the bounds in JurisdictionRule.validateLimits. These are sanity rails,
 * not statute — the server enforces the same numbers, and the point of checking
 * here is that a typo lands on the field that is wrong rather than coming back
 * as a whole-form error.
 */
const MAX_REASONABLE_WIDTH_FEET = 60;
const MAX_REASONABLE_HEIGHT_FEET = 40;
const MAX_REASONABLE_LENGTH_FEET = 300;
const MAX_LEAD_TIME_DAYS = 60;
const MAX_VALIDITY_DAYS = 365;

export const jurisdictionRuleSchema = z
  .object({
    id: z.string().optional(),
    stateId: z.string().min(1, { message: "State is required" }),
    status: jurisdictionRuleStatusSchema.default("Active"),

    maxWidthFeet: z
      .number()
      .positive({ message: "Maximum width must be greater than zero" })
      .max(MAX_REASONABLE_WIDTH_FEET, { message: "Maximum width must be under 60 feet" }),
    maxHeightFeet: z
      .number()
      .positive({ message: "Maximum height must be greater than zero" })
      .max(MAX_REASONABLE_HEIGHT_FEET, { message: "Maximum height must be under 40 feet" }),
    maxLengthFeet: z
      .number()
      .positive({ message: "Maximum length must be greater than zero" })
      .max(MAX_REASONABLE_LENGTH_FEET, { message: "Maximum length must be under 300 feet" }),
    maxWeightPounds: z
      .number()
      .int()
      .positive({ message: "Maximum weight must be greater than zero" }),

    superloadWidthFeet: z.number().positive().nullish(),
    superloadWeightPounds: z.number().int().positive().nullish(),

    daylightOnly: z.boolean().default(false),
    rushHourRestricted: z.boolean().default(false),
    weekendRestricted: z.boolean().default(false),
    holidayRestricted: z.boolean().default(true),

    permitLeadTimeDays: z
      .number()
      .int()
      .min(0, { message: "Lead time cannot be negative" })
      .max(MAX_LEAD_TIME_DAYS, { message: "Lead time must be 60 days or fewer" })
      .default(1),
    permitValidityDays: z
      .number()
      .int()
      .positive({ message: "Validity must be at least one day" })
      .max(MAX_VALIDITY_DAYS, { message: "Validity must be 365 days or fewer" })
      .default(5),
    permitBaseFee: decimalStringSchema,
    permitPerMileFee: decimalStringSchema,

    sourceNote: optionalStringSchema,
    sourceUrl: optionalStringSchema,

    // Read-only through the form. Verification is earned through the verify
    // action, which records who confirmed the row and against what, and the
    // server resets this whenever a limit changes.
    verificationState: jurisdictionVerificationStateSchema.default("Unverified"),
    verifiedAt: z.number().int().nullish(),

    effectiveStartDate: z.number().int().nullish(),
    effectiveEndDate: z.number().int().nullish(),

    version: z.number().int().optional(),
    createdAt: z.number().int().optional(),
    updatedAt: z.number().int().optional(),

    state: z.custom<{ id: string; name: string; abbreviation: string }>().nullish(),
  })
  // Mirrors validateLimits: a superload threshold at or below the permitted
  // maximum would classify every permitted load as a superload.
  .refine((v) => v.superloadWidthFeet == null || v.superloadWidthFeet > v.maxWidthFeet, {
    message: "Superload width must be greater than the permitted maximum width",
    path: ["superloadWidthFeet"],
  })
  .refine((v) => v.superloadWeightPounds == null || v.superloadWeightPounds > v.maxWeightPounds, {
    message: "Superload weight must be greater than the permitted maximum weight",
    path: ["superloadWeightPounds"],
  })
  .refine(
    (v) =>
      v.effectiveStartDate == null ||
      v.effectiveEndDate == null ||
      v.effectiveEndDate > v.effectiveStartDate,
    {
      message: "The effective end date must be after the start date",
      path: ["effectiveEndDate"],
    },
  );

export type JurisdictionRule = z.infer<typeof jurisdictionRuleSchema>;

export const verifyJurisdictionRuleSchema = z.object({
  verificationState: z.enum(["Verified", "Disputed"]),
  sourceNote: z.string().min(MIN_SOURCE_NOTE_LENGTH, {
    message: "Record what this was checked against, at least 10 characters",
  }),
  sourceUrl: optionalStringSchema,
});
export type VerifyJurisdictionRuleInput = z.infer<typeof verifyJurisdictionRuleSchema>;
