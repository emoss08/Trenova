import { z } from "zod";

export const jobDepartmentSchema = z.enum([
  "Operations",
  "Safety",
  "Maintenance",
  "Billing",
  "Administration",
  "Sales",
  "HumanResources",
  "Executive",
  "Other",
]);
export type JobDepartment = z.infer<typeof jobDepartmentSchema>;

export const approvalScopeSchema = z.enum(["All", "TimeOff", "Expenses"]);
export type ApprovalScopeValue = z.infer<typeof approvalScopeSchema>;

/**
 * A job title. The code is what the unique index is on, so it is bounded here
 * rather than left to the database to reject after a round trip.
 */
export const jobPositionFormSchema = z.object({
  code: z.string().min(1, "A code is required").max(20),
  title: z.string().min(1, "A title is required").max(100),
  description: z.string().nullable(),
  department: jobDepartmentSchema,
  flsaExempt: z.boolean(),
  isDrivingPosition: z.boolean(),
  reportsToPositionId: z.string().nullable(),
  status: z.enum(["Active", "Inactive"]),
});
export type JobPositionFormValues = z.infer<typeof jobPositionFormSchema>;

/**
 * Handing approval to somebody else for a while. An open end date is allowed —
 * a manager handing over an area rather than a fortnight wants one — but a
 * window that closes before it opens is refused.
 */
export const delegationFormSchema = z
  .object({
    delegateId: z.string().min(1, "Choose who is covering"),
    scope: approvalScopeSchema,
    startsAt: z.number(),
    endsAt: z.number().nullable(),
    reason: z.string().max(255).nullable(),
  })
  .refine((values) => values.endsAt === null || values.endsAt >= values.startsAt, {
    message: "A delegation cannot end before it begins",
    path: ["endsAt"],
  });
export type DelegationFormValues = z.infer<typeof delegationFormSchema>;
