import { z } from "zod";
import { driverTypeSchema, workerLeaveTypeSchema, workerTypeSchema } from "./worker";

export const employmentEventKindSchema = z.enum([
  "Hired",
  "ProbationEnded",
  "Promoted",
  "Transferred",
  "LeaveStarted",
  "LeaveEnded",
  "Suspended",
  "Reinstated",
  "Terminated",
  "Rehired",
  "RateChanged",
]);
export type EmploymentEventKind = z.infer<typeof employmentEventKindSchema>;

export const EMPLOYMENT_EVENT_LABELS: Record<EmploymentEventKind, string> = {
  Hired: "Hired",
  ProbationEnded: "Probation ended",
  Promoted: "Promoted",
  Transferred: "Transferred",
  LeaveStarted: "Leave started",
  LeaveEnded: "Leave ended",
  Suspended: "Suspended",
  Reinstated: "Reinstated",
  Terminated: "Terminated",
  Rehired: "Rehired",
  RateChanged: "Rate changed",
};

/** Kinds that need a written reason before they can be recorded. */
export const EMPLOYMENT_EVENT_REQUIRES_REASON: ReadonlySet<EmploymentEventKind> = new Set([
  "Terminated",
  "Suspended",
  "LeaveStarted",
  "RateChanged",
]);

/** Human labels for the stable keys carried in fromValues / toValues. */
export const EMPLOYMENT_VALUE_LABELS: Record<string, string> = {
  hireDate: "Hire date",
  terminationDate: "Termination date",
  status: "Status",
  fleetCodeId: "Fleet",
  fleetCode: "Fleet",
  managerId: "Manager",
  manager: "Manager",
  driverType: "Driver type",
  workerType: "Worker type",
  rate: "Rate",
  rateUnit: "Unit",
  leaveType: "Leave type",
  canBeAssigned: "Dispatchable",
};

/** Keys whose value is an id and should not be shown when a labelled twin exists. */
export const EMPLOYMENT_VALUE_HIDDEN_KEYS: ReadonlySet<string> = new Set([
  "fleetCodeId",
  "managerId",
]);

const optionalTrimmed = (max: number, message: string) =>
  z
    .string()
    .nullable()
    .optional()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().max(max, { message }).nullable());

export const employmentEventFormSchema = z
  .object({
    kind: employmentEventKindSchema,
    effectiveAt: z.number().int().positive({ message: "Choose the effective date" }),
    reason: optionalTrimmed(255, "Reason cannot exceed 255 characters"),
    notes: optionalTrimmed(4000, "Notes cannot exceed 4000 characters"),
    fleetCodeId: z.string().nullable().optional(),
    managerId: z.string().nullable().optional(),
    driverType: driverTypeSchema.nullable().optional(),
    workerType: workerTypeSchema.nullable().optional(),
    rate: optionalTrimmed(40, "Rate cannot exceed 40 characters"),
    rateUnit: optionalTrimmed(40, "Unit cannot exceed 40 characters"),
    leaveType: workerLeaveTypeSchema.nullable().optional(),
  })
  .superRefine((values, ctx) => {
    if (values.kind === "LeaveStarted" && !values.leaveType) {
      ctx.addIssue({
        code: "custom",
        path: ["leaveType"],
        message: "Choose the kind of leave",
      });
    }
    if (EMPLOYMENT_EVENT_REQUIRES_REASON.has(values.kind) && !values.reason) {
      ctx.addIssue({
        code: "custom",
        path: ["reason"],
        message: `A reason is required when recording ${EMPLOYMENT_EVENT_LABELS[values.kind].toLowerCase()}`,
      });
    }
    if (values.kind === "Transferred" && !values.fleetCodeId && !values.managerId) {
      ctx.addIssue({
        code: "custom",
        path: ["fleetCodeId"],
        message: "Choose the fleet the worker is moving to",
      });
    }
    if (values.kind === "Promoted" && !values.driverType && !values.workerType) {
      ctx.addIssue({
        code: "custom",
        path: ["driverType"],
        message: "Choose the new driver type or worker type",
      });
    }
    if (values.kind === "RateChanged" && !values.rate) {
      ctx.addIssue({
        code: "custom",
        path: ["rate"],
        message: "Enter the new rate",
      });
    }
  });
export type EmploymentEventFormValues = z.infer<typeof employmentEventFormSchema>;

export const employmentEventAmendSchema = z.object({
  effectiveAt: z.number().int().positive({ message: "Choose the effective date" }),
  reason: optionalTrimmed(255, "Reason cannot exceed 255 characters"),
  notes: optionalTrimmed(4000, "Notes cannot exceed 4000 characters"),
  amendmentNote: z
    .string()
    .trim()
    .min(1, { message: "Say why the event is being amended" })
    .max(255, { message: "Amendment note cannot exceed 255 characters" }),
});
export type EmploymentEventAmendValues = z.infer<typeof employmentEventAmendSchema>;
