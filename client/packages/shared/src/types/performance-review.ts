import { z } from "zod";

export const performanceReviewStatusSchema = z.enum([
  "Draft",
  "Submitted",
  "Acknowledged",
  "Closed",
]);
export type PerformanceReviewStatus = z.infer<typeof performanceReviewStatusSchema>;

export const reviewGoalStatusSchema = z.enum(["Open", "Done", "Dropped"]);
export type ReviewGoalStatus = z.infer<typeof reviewGoalStatusSchema>;

export const PERFORMANCE_REVIEW_STATUS_LABELS: Record<PerformanceReviewStatus, string> = {
  Draft: "Draft",
  Submitted: "Submitted",
  Acknowledged: "Signed",
  Closed: "Closed",
};

/** What each status is waiting on, in the reviewer's words. */
export const PERFORMANCE_REVIEW_STATUS_HINTS: Record<PerformanceReviewStatus, string> = {
  Draft: "Waiting on you to submit it.",
  Submitted: "Waiting on the worker to sign off in Dash.",
  Acknowledged: "Signed by the worker — close it when you are ready.",
  Closed: "Filed. The next review is scheduled from the template cadence.",
};

export const REVIEW_GOAL_STATUS_LABELS: Record<ReviewGoalStatus, string> = {
  Open: "Open",
  Done: "Done",
  Dropped: "Dropped",
};

/** The 1–5 scale, with the words that go on each mark. */
export const REVIEW_SCORE_LABELS: Record<number, string> = {
  1: "Needs work",
  2: "Below",
  3: "Meets",
  4: "Strong",
  5: "Exceptional",
};

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

export const reviewItemFormSchema = z.object({
  key: z
    .string()
    .trim()
    .min(1, { message: "Key is required" })
    .max(50, { message: "Key cannot exceed 50 characters" })
    .regex(/^[a-z0-9_-]+$/, { message: "Lower-case letters, digits, dashes and underscores" }),
  label: z
    .string()
    .trim()
    .min(1, { message: "Label is required" })
    .max(100, { message: "Label cannot exceed 100 characters" }),
  description: optionalTrimmed(255, "Description cannot exceed 255 characters"),
  weight: z
    .number()
    .int()
    .min(1, { message: "Weight must be at least 1" })
    .max(10, { message: "Weight cannot exceed 10" }),
});
export type ReviewItemFormValues = z.infer<typeof reviewItemFormSchema>;

export const reviewTemplateFormSchema = z
  .object({
    code: z
      .string()
      .trim()
      .min(1, { message: "Code is required" })
      .max(50, { message: "Code cannot exceed 50 characters" })
      .regex(/^[A-Za-z0-9_-]+$/, { message: "Letters, digits, dashes and underscores only" }),
    name: z
      .string()
      .trim()
      .min(1, { message: "Name is required" })
      .max(100, { message: "Name cannot exceed 100 characters" }),
    description: optionalTrimmed(1000, "Description cannot exceed 1000 characters"),
    status: z.enum(["Active", "Inactive"]),
    isDefault: z.boolean(),
    cadenceMonths: z.number().int().min(1, { message: "At least one month" }).nullable(),
    items: z.array(reviewItemFormSchema).min(1, { message: "Add at least one rating item" }),
  })
  .superRefine((values, ctx) => {
    if (values.isDefault && values.status !== "Active") {
      ctx.addIssue({
        code: "custom",
        path: ["isDefault"],
        message: "Only an active template can be the default",
      });
    }
    const seen = new Set<string>();
    values.items.forEach((item, index) => {
      if (seen.has(item.key)) {
        ctx.addIssue({
          code: "custom",
          path: ["items", index, "key"],
          message: "Item keys must be unique",
        });
      }
      seen.add(item.key);
    });
  });
export type ReviewTemplateFormValues = z.infer<typeof reviewTemplateFormSchema>;

export const createReviewFormSchema = z
  .object({
    templateId: z.string().min(1, { message: "Choose a template" }),
    title: optionalTrimmed(120, "Title cannot exceed 120 characters"),
    periodStart: z.number().int().positive({ message: "Choose the period start" }),
    periodEnd: z.number().int().positive({ message: "Choose the period end" }),
  })
  .superRefine((values, ctx) => {
    if (values.periodEnd < values.periodStart) {
      ctx.addIssue({
        code: "custom",
        path: ["periodEnd"],
        message: "The period must end after it starts",
      });
    }
  });
export type CreateReviewFormValues = z.infer<typeof createReviewFormSchema>;

export const reviewDraftFormSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, { message: "Give the review a title" })
    .max(120, { message: "Title cannot exceed 120 characters" }),
  periodStart: z.number().int().positive(),
  periodEnd: z.number().int().positive(),
  ratings: z.array(
    z.object({
      key: z.string(),
      label: z.string(),
      weight: z.number().int(),
      score: z.number().int().min(1).max(5).nullable(),
      comment: optionalTrimmed(2000, "Comment cannot exceed 2000 characters"),
    }),
  ),
  summary: optionalTrimmed(4000, "Summary cannot exceed 4000 characters"),
  strengths: optionalTrimmed(4000, "Strengths cannot exceed 4000 characters"),
  improvements: optionalTrimmed(4000, "Improvements cannot exceed 4000 characters"),
  goals: z.array(
    z.object({
      id: z.string().nullable(),
      title: z
        .string()
        .trim()
        .min(1, { message: "Goal needs a title" })
        .max(255, { message: "Goal cannot exceed 255 characters" }),
      dueAt: z.number().int().positive().nullable(),
      status: reviewGoalStatusSchema,
    }),
  ),
});
export type ReviewDraftFormValues = z.infer<typeof reviewDraftFormSchema>;

/**
 * The weighted mean the server computes, mirrored so the editor can show a
 * live score before saving. Returns null while nothing is rated.
 */
export function computeReviewScore(
  ratings: readonly { weight: number; score: number | null }[],
): number | null {
  let weighted = 0;
  let weights = 0;
  for (const rating of ratings) {
    if (rating.score == null || rating.weight <= 0) continue;
    weighted += rating.score * rating.weight;
    weights += rating.weight;
  }
  if (weights === 0) return null;
  return Math.round((weighted / weights) * 100) / 100;
}
