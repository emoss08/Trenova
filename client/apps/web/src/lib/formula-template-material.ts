import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";

/**
 * Mirrors the server's material-change rule: a change to what the template bills is a change
 * to its source (the imports are derived from it), and returns an approved template to Draft.
 * Name and description edits are not material.
 */
export function hasMaterialChange(
  template: Pick<FormulaTemplate, "source">,
  values: Pick<FormulaTemplateFormValues, "source">,
): boolean {
  return template.source !== (values.source ?? "");
}

/** Whether saving would take this template out of production (or out of its review). */
export function saveDemotesToDraft(
  template: Pick<FormulaTemplate, "source" | "status"> | null,
  values: Pick<FormulaTemplateFormValues, "source">,
): boolean {
  if (!template) return false;
  if (template.status !== "Active" && template.status !== "InReview") return false;
  return hasMaterialChange(template, values);
}
