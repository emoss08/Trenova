import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const timesheetStatusSchema = z.enum([
  "Open",
  "Submitted",
  "Approved",
  "Rejected",
  "Locked",
]);
export type TimesheetStatusValue = z.infer<typeof timesheetStatusSchema>;

export const timeEntrySourceSchema = z.enum(["Clock", "Portal", "Manual", "Import"]);
export type TimeEntrySourceValue = z.infer<typeof timeEntrySourceSchema>;

const MINUTES_IN_DAY = 24 * 60;

/**
 * A period recorded or corrected by hand. The reason is required rather than
 * optional: a wage record altered by somebody with no reason recorded is not a
 * record anybody can defend.
 */
export const recordTimeEntryFormSchema = z
  .object({
    clockedInAt: z
      .number()
      .int()
      .positive({ error: () => translate("A start time is required") }),
    clockedOutAt: z
      .number()
      .int()
      .positive({ error: () => translate("A finish time is required") }),
    breakMinutes: z
      .number()
      .int()
      .min(0, { error: () => translate("A break cannot be negative") })
      .max(MINUTES_IN_DAY),
    note: z.string().max(500).nullable(),
    reason: z
      .string()
      .min(1, { error: () => translate("Changing somebody's hours needs a reason") })
      .max(500),
  })
  .refine((values) => values.clockedOutAt > values.clockedInAt, {
    error: () => translate("An entry cannot end before it began"),
    path: ["clockedOutAt"],
  })
  // A punch nobody closed until the next day is a forgotten clock-out rather
  // than a day somebody worked straight through.
  .refine((values) => (values.clockedOutAt - values.clockedInAt) / 60 <= MINUTES_IN_DAY, {
    error: () => translate("An entry cannot be longer than a day — correct the finish time"),
    path: ["clockedOutAt"],
  })
  .refine((values) => values.breakMinutes < (values.clockedOutAt - values.clockedInAt) / 60, {
    error: () => translate("The break is as long as the entry — nothing would be paid"),
    path: ["breakMinutes"],
  });
export type RecordTimeEntryFormValues = z.infer<typeof recordTimeEntryFormSchema>;

/** Taking a punch off the record. The reason is required for the same reason a correction's is. */
export const removeTimeEntryFormSchema = z.object({
  reason: z
    .string()
    .min(1, { error: () => translate("Removing somebody's hours needs a reason") })
    .max(500),
});
export type RemoveTimeEntryFormValues = z.infer<typeof removeTimeEntryFormSchema>;

/**
 * Handing a payroll run out. The period is a range of weeks rather than one, so
 * a fortnightly or monthly payroll takes several sheets in one file.
 */
export const payrollExportFormSchema = z
  .object({
    periodStart: z
      .number()
      .int()
      .positive({ error: () => translate("A period start is required") }),
    periodEnd: z
      .number()
      .int()
      .positive({ error: () => translate("A period end is required") }),
    note: z.string().max(500).nullable(),
  })
  .refine((values) => values.periodEnd > values.periodStart, {
    error: () => translate("A period cannot end before it begins"),
    path: ["periodEnd"],
  });
export type PayrollExportFormValues = z.infer<typeof payrollExportFormSchema>;

/**
 * Taking a payroll run back. Voiding one reopens every week in it, so the
 * reason is required: nobody can explain the reopening afterwards otherwise.
 */
export const voidPayrollExportFormSchema = z.object({
  reason: z
    .string()
    .min(1, { error: () => translate("Voiding a run needs a reason") })
    .max(500),
});
export type VoidPayrollExportFormValues = z.infer<typeof voidPayrollExportFormSchema>;
