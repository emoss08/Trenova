import { z } from "zod";

export const benefitPlanTypeSchema = z.enum([
  "Medical",
  "Dental",
  "Vision",
  "Life",
  "Disability",
  "Retirement",
  "Other",
]);

export const coverageTierSchema = z.enum([
  "Employee",
  "EmployeeSpouse",
  "EmployeeChildren",
  "Family",
]);
export type CoverageTierValue = z.infer<typeof coverageTierSchema>;

/**
 * A benefit plan. Amounts are in minor units, and the pay code is required
 * because a contribution nobody can categorise is a deduction nobody can
 * explain when it turns up on a settlement.
 */
export const benefitPlanFormSchema = z
  .object({
    code: z.string().min(1, "A code is required").max(20),
    name: z.string().min(1, "A name is required").max(100),
    description: z.string().nullable(),
    planType: benefitPlanTypeSchema,
    carrier: z.string().max(150).nullable(),
    policyNumber: z.string().max(100).nullable(),
    payCodeId: z.string().min(1, "A pay code is required"),
    planYear: z.number().int().min(2000).max(2200),
    employeeCostMinor: z.number().int().min(0),
    employerCostMinor: z.number().int().min(0),
    waitingPeriodDays: z.number().int().min(0).max(365),
    status: z.enum(["Active", "Inactive"]),
  })
  // A plan that costs nobody anything would produce a zero deduction on every
  // settlement forever.
  .refine((values) => values.employeeCostMinor > 0 || values.employerCostMinor > 0, {
    message: "A plan has to cost somebody something",
    path: ["employeeCostMinor"],
  });
export type BenefitPlanFormValues = z.infer<typeof benefitPlanFormSchema>;

/**
 * Putting somebody on a plan, or recording that they declined it. A waiver
 * still produces an enrollment, because "declined" and "nobody asked" are
 * different facts and only one of them is a problem at audit.
 */
export const benefitEnrollmentFormSchema = z
  .object({
    benefitPlanId: z.string().min(1, "Choose a plan"),
    coverageTier: coverageTierSchema,
    waive: z.boolean(),
    waivedReason: z.string().max(255).nullable(),
    employeeCostMinor: z.number().int().min(0).nullable(),
    effectiveFrom: z.number(),
    notes: z.string().nullable(),
  })
  .refine((values) => !values.waive || Boolean(values.waivedReason?.trim()), {
    message: "Say why the cover was declined",
    path: ["waivedReason"],
  });
export type BenefitEnrollmentFormValues = z.infer<typeof benefitEnrollmentFormSchema>;
