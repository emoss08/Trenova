import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";
import {
  compareDecimalStrings,
  decimalString,
  isDecimalString,
} from "@trenova/shared/types/decimal";

export const leaveCaseStatusSchema = z.enum(["Pending", "Approved", "Denied", "Closed"]);
export type LeaveCaseStatus = z.infer<typeof leaveCaseStatusSchema>;

export const leaveFrequencySchema = z.enum(["Continuous", "Intermittent", "ReducedSchedule"]);
export type LeaveFrequency = z.infer<typeof leaveFrequencySchema>;

export const leaveCertificationStatusSchema = z.enum([
  "NotRequired",
  "Requested",
  "Received",
  "Insufficient",
  "Overdue",
  "Waived",
]);
export type LeaveCertificationStatus = z.infer<typeof leaveCertificationStatusSchema>;

export const leaveMeasurementMethodSchema = z.enum([
  "CalendarYear",
  "HireAnniversary",
  "ForwardFromFirstUse",
  "RollingBackward",
]);
export type LeaveMeasurementMethod = z.infer<typeof leaveMeasurementMethodSchema>;

export const leaveTypeFormSchema = z.enum([
  "FMLA",
  "Medical",
  "Military",
  "Parental",
  "Personal",
  "Other",
]);

/** Opening or correcting a leave case. */
export const leaveCaseFormSchema = z
  .object({
    leaveType: leaveTypeFormSchema,
    frequency: leaveFrequencySchema,
    reason: z.string().max(255).nullable(),
    militaryCaregiver: z.boolean(),
    requestedAt: z.number(),
    startsAt: z.number(),
    endsAt: z.number().nullable(),
    eligibilityHoursWorked: z.number().min(0).nullable(),
    notes: z.string().nullable(),
  })
  .refine((values) => values.endsAt === null || values.endsAt >= values.startsAt, {
    error: () => translate("Leave cannot end before it begins"),
    path: ["endsAt"],
  });
export type LeaveCaseFormValues = z.infer<typeof leaveCaseFormSchema>;

/** One day of leave taken. */
export const leaveDayFormSchema = z.object({
  usedOn: z.number(),
  hours: z
    .number()
    .positive({ error: () => translate("Record more than zero hours") })
    .max(24, { error: () => translate("A day cannot hold more than 24 hours") }),
  notes: z.string().nullable(),
});
export type LeaveDayFormValues = z.infer<typeof leaveDayFormSchema>;

/** The server keeps weeks and workweek hours as NUMERIC(5, 2). */
export const LEAVE_CONTROL_DECIMAL_SCALE = 2;

const leaveDecimalSchema = (
  missing: () => string,
  checks: { floor: string; floorMessage: () => string; max: string; maxMessage: () => string },
) =>
  z.string({ error: missing }).pipe(
    decimalString(
      LEAVE_CONTROL_DECIMAL_SCALE,
      translate("Enter a number with at most two decimals"),
    )
      .refine((value) => compareDecimalStrings(value, checks.floor) >= 0, {
        error: checks.floorMessage,
      })
      .refine((value) => compareDecimalStrings(value, checks.max) <= 0, {
        error: checks.maxMessage,
      }),
  );

/**
 * The organisation's leave settings, as the `leaveControl` query sends them and the form
 * edits them; weeks and hours stay the decimal strings the server sends. The statutory
 * figures are floors, not targets: an employer may be more generous but cannot offer less
 * than twelve weeks and call it FMLA (29 CFR 825.200(a)). Nothing may exceed what a year
 * holds (52 weeks, 168 hours a week).
 */
export const leaveControlSchema = z
  .object({
    id: z.string(),
    version: z.number().int().min(0),
    measurementMethod: leaveMeasurementMethodSchema,
    entitlementWeeks: leaveDecimalSchema(() => translate("Enter the weeks of leave"), {
      floor: "12",
      floorMessage: () => translate("FMLA entitles an eligible employee to at least twelve weeks"),
      max: "52",
      maxMessage: () => translate("Leave cannot be more than 52 weeks a year"),
    }),
    militaryCaregiverWeeks: leaveDecimalSchema(
      () => translate("Enter the weeks of military caregiver leave"),
      {
        floor: "26",
        floorMessage: () => translate("Military caregiver leave is at least twenty-six weeks"),
        max: "52",
        maxMessage: () => translate("Leave cannot be more than 52 weeks a year"),
      },
    ),
    workweekHours: leaveDecimalSchema(() => translate("Enter the hours in a workweek"), {
      floor: "0.01",
      floorMessage: () => translate("A workweek must be more than zero hours"),
      max: "168",
      maxMessage: () => translate("A workweek cannot be more than 168 hours"),
    }),
    eligibilityMonths: z
      .number({ error: () => translate("Enter the months of service") })
      .int()
      .min(0, { error: () => translate("Months of service cannot be negative") })
      .max(120, { error: () => translate("Months of service cannot be more than 120") }),
    eligibilityHours: z
      .number({ error: () => translate("Enter the hours worked") })
      .int()
      .min(0, { error: () => translate("Hours worked cannot be negative") })
      .max(8760, { error: () => translate("A year holds at most 8,760 hours") }),
    certificationDueDays: z
      .number({ error: () => translate("Enter the days to return a certification") })
      .int()
      .min(15, {
        error: () =>
          translate("The employee must be given at least fifteen days (29 CFR 825.305(b))"),
      }),
  })
  .refine(
    (values) =>
      !isDecimalString(values.militaryCaregiverWeeks) ||
      !isDecimalString(values.entitlementWeeks) ||
      compareDecimalStrings(values.militaryCaregiverWeeks, values.entitlementWeeks) >= 0,
    {
      error: () =>
        translate("Military caregiver leave cannot be shorter than the ordinary entitlement"),
      path: ["militaryCaregiverWeeks"],
    },
  );
export type LeaveControl = z.infer<typeof leaveControlSchema>;
