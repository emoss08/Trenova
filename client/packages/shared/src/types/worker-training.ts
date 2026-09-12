import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import { driverTypeSchema } from "./worker";
// Re-exported so callers keep importing training health from here; the
// definition lives in a leaf module to keep this file and worker.ts from
// importing each other.
export {
  WORKER_TRAINING_HEALTH_LABELS,
  workerTrainingHealthSchema,
  type WorkerTrainingHealth,
} from "./worker-training-health";

export const trainingCategorySchema = z.enum([
  "Safety",
  "Compliance",
  "Equipment",
  "Orientation",
  "HazardousMaterials",
  "Other",
]);
export type TrainingCategory = z.infer<typeof trainingCategorySchema>;

export const trainingDeliverySchema = z.enum(["Online", "Classroom", "OnTheJob", "Document"]);
export type TrainingDelivery = z.infer<typeof trainingDeliverySchema>;

export const workerTrainingStatusSchema = z.enum([
  "Assigned",
  "InProgress",
  "Completed",
  "Failed",
  "Expired",
  "Waived",
  "Cancelled",
]);
export type WorkerTrainingStatus = z.infer<typeof workerTrainingStatusSchema>;

export const TRAINING_CATEGORY_LABELS: Record<TrainingCategory, string> = {
  Safety: "Safety",
  Compliance: "Compliance",
  Equipment: "Equipment",
  Orientation: "Orientation",
  HazardousMaterials: "Hazardous materials",
  Other: "Other",
};

export const TRAINING_DELIVERY_LABELS: Record<TrainingDelivery, string> = {
  Online: "Online",
  Classroom: "Classroom",
  OnTheJob: "On the job",
  Document: "Read & acknowledge",
};

/** Whether a driver can finish the course from the portal without a score. */
export const TRAINING_DELIVERY_SELF_SERVE: ReadonlySet<TrainingDelivery> = new Set([
  "Online",
  "Document",
]);

export const TRAINING_DELIVERY_HINTS: Record<TrainingDelivery, string> = {
  Online: "The driver opens a link from Dash and confirms when they are done.",
  Classroom: "Taken in person; the office records attendance and the score.",
  OnTheJob: "Coached on the job; a supervisor records the result.",
  Document: "The driver reads a document from Dash and acknowledges it.",
};

export const WORKER_TRAINING_STATUS_LABELS: Record<WorkerTrainingStatus, string> = {
  Assigned: "Assigned",
  InProgress: "In progress",
  Completed: "Completed",
  Failed: "Failed",
  Expired: "Expired",
  Waived: "Waived",
  Cancelled: "Cancelled",
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

const PERCENT_PATTERN = /^\d{1,3}(\.\d{1,2})?$/;

const optionalPercent = z
  .string()
  .nullable()
  .optional()
  .transform((value) => {
    const trimmed = value?.trim() ?? "";
    return trimmed === "" ? null : trimmed;
  })
  .pipe(
    z
      .string()
      .regex(PERCENT_PATTERN, { message: "Enter a percentage with up to two decimals" })
      .refine((value) => Number(value) <= 100, { message: "Cannot exceed 100" })
      .nullable(),
  );

export const trainingCourseFormSchema = z
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
    category: trainingCategorySchema,
    status: z.enum(["Active", "Inactive"]),
    delivery: trainingDeliverySchema,
    contentUrl: optionalTrimmed(500, "Link cannot exceed 500 characters"),
    durationMinutes: z.number().int().min(0, { message: "Cannot be negative" }),
    passingScore: optionalPercent,
    validityMonths: z.number().int().min(1, { message: "Must be at least one month" }).nullable(),
    renewalWindowDays: z
      .number()
      .int()
      .min(0, { message: "Cannot be negative" })
      .max(365, { message: "Cannot exceed 365 days" }),
    isRequired: z.boolean(),
    requiredForDriverTypes: z.array(driverTypeSchema),
    dueDaysAfterAssignment: z
      .number()
      .int()
      .min(0, { message: "Cannot be negative" })
      .max(730, { message: "Cannot exceed two years" }),
    requiresAcknowledgement: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.delivery === "Online" && !values.contentUrl) {
      ctx.addIssue({
        code: "custom",
        path: ["contentUrl"],
        message: translate("Online courses need a link the driver can open"),
      });
    }
    if (values.contentUrl && !/^https?:\/\/\S+$/i.test(values.contentUrl)) {
      ctx.addIssue({
        code: "custom",
        path: ["contentUrl"],
        message: translate("Link must be a full http(s) address"),
      });
    }
  });
export type TrainingCourseFormValues = z.infer<typeof trainingCourseFormSchema>;

export const assignTrainingFormSchema = z.object({
  courseId: z.string().min(1, { message: "Choose a course" }),
  dueAt: z.number().int().positive().nullable(),
  notes: optionalTrimmed(1000, "Notes cannot exceed 1000 characters"),
});
export type AssignTrainingFormValues = z.infer<typeof assignTrainingFormSchema>;

export const completeTrainingFormSchema = z
  .object({
    courseId: z.string().min(1, { message: "Choose a course" }),
    completedAt: z.number().int().positive({ message: "Choose the completion date" }),
    score: optionalPercent,
    notes: optionalTrimmed(1000, "Notes cannot exceed 1000 characters"),
    requiresScore: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.requiresScore && values.score === null) {
      ctx.addIssue({
        code: "custom",
        path: ["score"],
        message: translate("This course is scored; enter the result"),
      });
    }
  });
export type CompleteTrainingFormValues = z.infer<typeof completeTrainingFormSchema>;

export const waiveTrainingFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(3, { message: "Say why the course is waived" })
    .max(255, { message: "Reason cannot exceed 255 characters" }),
});
export type WaiveTrainingFormValues = z.infer<typeof waiveTrainingFormSchema>;
