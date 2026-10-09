import { checklistItemModeSchema, customCheckSchema } from "@/types/case-checklist";
import { z } from "zod";

/** The most steps an organization may add to one checklist; the server holds the same. */
export const MAX_ADDED_STEPS = 12;

/**
 * What a person may save. The server checks it whole again, but an added
 * step without a name, or a document step without its document type, is
 * said next to the field before anything is sent.
 */
const editorItemSchema = z.object({
  key: z.string(),
  mode: checklistItemModeSchema,
  custom: z
    .object({
      label: z.string(),
      check: customCheckSchema,
      documentTypeId: z.string(),
      stepLabel: z.string(),
      prompt: z.string(),
    })
    .nullish(),
});

/**
 * The form holds steps as the API's parser hands them over, already filled
 * in, so its schema reads and writes the same shape and the form's values
 * and what it saves are one type.
 */
export const checklistEditorSchema = z.object({
  items: z.array(
    editorItemSchema.superRefine((item, ctx) => {
      const custom = item.custom;
      if (!custom) {
        return;
      }
      if (custom.label.trim() === "") {
        ctx.addIssue({ code: "custom", path: ["custom", "label"], message: "Name the step" });
      }
      if (custom.label.length > 80) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "label"],
          message: "A step's name can be at most 80 characters",
        });
      }
      if (custom.check === "Document" && !custom.documentTypeId) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "documentTypeId"],
          message: "Choose the document type that ticks it",
        });
      }
      if (custom.stepLabel.length > 40) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "stepLabel"],
          message: "A step's button can say at most 40 characters",
        });
      }
      if (custom.prompt.length > 500) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "prompt"],
          message: "What the step asks the agent can be at most 500 characters",
        });
      }
    }),
  ),
});

export type ChecklistEditorValues = z.infer<typeof checklistEditorSchema>;
