import { z } from "zod";

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
    message: "Leave cannot end before it begins",
    path: ["endsAt"],
  });
export type LeaveCaseFormValues = z.infer<typeof leaveCaseFormSchema>;

/** One day of leave taken. */
export const leaveDayFormSchema = z.object({
  usedOn: z.number(),
  hours: z
    .number()
    .positive("Record more than zero hours")
    .max(24, "A day cannot hold more than 24 hours"),
  notes: z.string().nullable(),
});
export type LeaveDayFormValues = z.infer<typeof leaveDayFormSchema>;

/**
 * The organisation's leave settings. The statutory figures are floors, not
 * targets: an employer may be more generous but cannot offer less than twelve
 * weeks and call it FMLA (29 CFR 825.200(a)).
 */
export const leaveControlFormSchema = z
  .object({
    measurementMethod: leaveMeasurementMethodSchema,
    entitlementWeeks: z
      .number()
      .min(12, "FMLA entitles an eligible employee to at least twelve weeks"),
    militaryCaregiverWeeks: z
      .number()
      .min(26, "Military caregiver leave is at least twenty-six weeks"),
    workweekHours: z.number().positive("A workweek must be more than zero hours").max(168),
    eligibilityMonths: z.number().int().min(0).max(120),
    eligibilityHours: z.number().int().min(0).max(8760),
    certificationDueDays: z
      .number()
      .int()
      .min(15, "The employee must be given at least fifteen days (29 CFR 825.305(b))"),
  })
  .refine((values) => values.militaryCaregiverWeeks >= values.entitlementWeeks, {
    message: "Military caregiver leave cannot be shorter than the ordinary entitlement",
    path: ["militaryCaregiverWeeks"],
  });
export type LeaveControlFormValues = z.infer<typeof leaveControlFormSchema>;
