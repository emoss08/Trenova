import { tariffSetName } from "@trenova/shared/rules/tariff";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";

type Content = Pick<FormulaTemplate, "name" | "source">;

/**
 * Mirrors the server's material-change rule: a change to what the template bills is a change
 * to its source (the imports are derived from it), and returns an approved template to Draft.
 * The name is the header's, so a change to the header alone is a rename, not a material change;
 * nor is a description edit.
 */
export function hasMaterialChange(
  rules: RulesRuntime,
  template: Content,
  values: Pick<FormulaTemplateFormValues, "name" | "source">,
): boolean {
  const source = values.source ?? "";
  if (template.source === source) return false;
  const name = values.name ?? "";
  if (template.name === name) return true;
  return tariffSetName(rules, template.source, name).source !== source;
}

/** Whether saving would take this template out of production (or out of its review). */
export function saveDemotesToDraft(
  rules: RulesRuntime,
  template: (Content & Pick<FormulaTemplate, "status">) | null,
  values: Pick<FormulaTemplateFormValues, "name" | "source">,
): boolean {
  if (!template) return false;
  if (template.status !== "Active" && template.status !== "InReview") return false;
  return hasMaterialChange(rules, template, values);
}
