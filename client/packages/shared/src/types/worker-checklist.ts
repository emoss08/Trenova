import { z } from "zod";

export const checklistKindSchema = z.enum(["Onboarding", "Offboarding", "Custom"]);
export type ChecklistKind = z.infer<typeof checklistKindSchema>;

export const checklistTriggerSchema = z.enum(["Hired", "Rehired", "Terminated", "Manual"]);
export type ChecklistTrigger = z.infer<typeof checklistTriggerSchema>;

export const checklistItemKindSchema = z.enum([
  "Document",
  "Credential",
  "Task",
  "Equipment",
  "PortalAccess",
]);
export type ChecklistItemKind = z.infer<typeof checklistItemKindSchema>;

export const checklistOwnerSchema = z.enum(["HR", "Safety", "Dispatch", "Payroll", "IT", "Fleet"]);
export type ChecklistOwner = z.infer<typeof checklistOwnerSchema>;

export const checklistStatusSchema = z.enum(["Open", "Completed", "Cancelled"]);
export type ChecklistStatus = z.infer<typeof checklistStatusSchema>;

export const checklistItemStatusSchema = z.enum(["Pending", "Done", "Skipped", "NotApplicable"]);
export type ChecklistItemStatus = z.infer<typeof checklistItemStatusSchema>;

export const CHECKLIST_KIND_LABELS: Record<ChecklistKind, string> = {
  Onboarding: "Onboarding",
  Offboarding: "Offboarding",
  Custom: "Custom",
};

export const CHECKLIST_TRIGGER_LABELS: Record<ChecklistTrigger, string> = {
  Hired: "When hired",
  Rehired: "When rehired",
  Terminated: "When terminated",
  Manual: "Started by hand",
};

export const CHECKLIST_ITEM_KIND_LABELS: Record<ChecklistItemKind, string> = {
  Document: "Document",
  Credential: "Credential",
  Task: "Task",
  Equipment: "Equipment",
  PortalAccess: "Portal access",
};

export const CHECKLIST_OWNER_LABELS: Record<ChecklistOwner, string> = {
  HR: "HR",
  Safety: "Safety",
  Dispatch: "Dispatch",
  Payroll: "Payroll",
  IT: "IT",
  Fleet: "Fleet",
};

export const CHECKLIST_ITEM_STATUS_LABELS: Record<ChecklistItemStatus, string> = {
  Pending: "Pending",
  Done: "Done",
  Skipped: "Skipped",
  NotApplicable: "Not applicable",
};

/** Kinds that complete themselves from evidence and cannot be ticked by hand. */
export const AUTO_SATISFIED_ITEM_KINDS: ReadonlySet<ChecklistItemKind> = new Set([
  "Credential",
  "Document",
  "PortalAccess",
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

export const checklistTemplateItemFormSchema = z
  .object({
    label: z
      .string()
      .trim()
      .min(1, { message: "Label is required" })
      .max(150, { message: "Label cannot exceed 150 characters" }),
    description: optionalTrimmed(1000, "Description cannot exceed 1000 characters"),
    kind: checklistItemKindSchema,
    required: z.boolean(),
    dueOffsetDays: z
      .number()
      .int()
      .min(0, { message: "Cannot be negative" })
      .max(365, { message: "Cannot exceed 365 days" }),
    owner: checklistOwnerSchema,
    credentialTypeId: z.string().nullable().optional(),
    documentTypeId: z.string().nullable().optional(),
  })
  .superRefine((item, ctx) => {
    if (item.kind === "Credential" && !item.credentialTypeId) {
      ctx.addIssue({
        code: "custom",
        path: ["credentialTypeId"],
        message: "Choose which credential this item waits for",
      });
    }
    if (item.kind === "Document" && !item.documentTypeId) {
      ctx.addIssue({
        code: "custom",
        path: ["documentTypeId"],
        message: "Choose which document type this item waits for",
      });
    }
  });
export type ChecklistTemplateItemFormValues = z.infer<typeof checklistTemplateItemFormSchema>;

export const checklistTemplateFormSchema = z
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
    kind: checklistKindSchema,
    trigger: checklistTriggerSchema,
    status: z.enum(["Active", "Inactive"]),
    isDefault: z.boolean(),
    items: z.array(checklistTemplateItemFormSchema).min(1, { message: "Add at least one item" }),
  })
  .superRefine((values, ctx) => {
    if (values.isDefault && values.trigger === "Manual") {
      ctx.addIssue({
        code: "custom",
        path: ["isDefault"],
        message: "A manual checklist cannot be the default — pick the event that should start it",
      });
    }
    if (values.isDefault && values.status !== "Active") {
      ctx.addIssue({
        code: "custom",
        path: ["isDefault"],
        message: "Only an active template can be the default",
      });
    }
  });
export type ChecklistTemplateFormValues = z.infer<typeof checklistTemplateFormSchema>;

export const checklistItemNoteSchema = z.object({
  note: optionalTrimmed(500, "Note cannot exceed 500 characters"),
});
export type ChecklistItemNoteValues = z.infer<typeof checklistItemNoteSchema>;
