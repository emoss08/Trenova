import { optionalIdSchema } from "@trenova/shared/types/helpers";
import { z } from "zod";

export const checklistKindSchema = z.enum(["ReadyToBill", "ReadyToClose"]);
export const checklistItemModeSchema = z.enum(["Required", "Optional", "Off"]);
export const customCheckSchema = z.enum(["Manual", "Document"]);

/** A step the organization added: its name, what ticks it, what it asks the agent. */
export const customChecklistItemSchema = z.object({
  label: z.string(),
  check: customCheckSchema,
  documentTypeId: optionalIdSchema,
  stepLabel: z.string().optional().default(""),
  prompt: z.string().optional().default(""),
});

/** One step of a template, in the order the case shows it. */
export const templateItemSchema = z.object({
  key: z.string(),
  mode: checklistItemModeSchema,
  custom: customChecklistItemSchema.nullish(),
});

/**
 * How the organization, or one customer of it, wants a kind of case
 * checklist laid out. The organization's template has no customer; an
 * unsaved one (the default) has no id.
 */
export const checklistTemplateSchema = z.object({
  id: optionalIdSchema,
  kind: checklistKindSchema,
  customerId: optionalIdSchema,
  customerName: z.string().optional().default(""),
  items: z.array(templateItemSchema),
  version: z.number().default(0),
  updatedAt: z.number().optional().default(0),
});

export const checklistTemplatesSchema = z.object({
  kind: checklistKindSchema,
  organization: checklistTemplateSchema,
  customers: z
    .array(checklistTemplateSchema)
    .nullish()
    .transform((customers) => customers ?? []),
  /** Built-in steps whose setting follows a rule kept elsewhere: movable, always required. */
  locked: z
    .array(z.string())
    .nullish()
    .transform((locked) => locked ?? []),
});

export type ChecklistKind = z.infer<typeof checklistKindSchema>;
export type ChecklistItemMode = z.infer<typeof checklistItemModeSchema>;
export type CustomCheck = z.infer<typeof customCheckSchema>;
export type CustomChecklistItem = z.infer<typeof customChecklistItemSchema>;
export type TemplateItem = z.infer<typeof templateItemSchema>;
export type ChecklistTemplate = z.infer<typeof checklistTemplateSchema>;
export type ChecklistTemplates = z.infer<typeof checklistTemplatesSchema>;

/** What a save sends: the template's steps, and which template it changes or makes. */
export type SaveChecklistTemplate = {
  id?: string;
  version: number;
  kind: ChecklistKind;
  customerId?: string;
  items: TemplateItem[];
};
