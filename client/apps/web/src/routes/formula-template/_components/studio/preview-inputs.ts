// preview-inputs.ts checks the Studio preview's sample input values the way a contract form checks
// a rule's: by the shared input rule (`tariffInputIssues`), so a value outside an input's type or
// `where` bounds is refused at its field as it is typed.

import { translate } from "@trenova/shared/i18n/runtime";
import { tariffInputIssues } from "@trenova/shared/lib/tariff-inputs";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import type { FieldErrors, ResolverResult } from "react-hook-form";

/**
 * The sample input values the preview rates with, held apart from the template itself. The
 * values are wire values; the form types them loosely because react-hook-form's path types do
 * not terminate on a recursive value type.
 */
export type PreviewInputsForm = { inputs: Record<string, unknown> };

/**
 * The preview's sample values against the tariff's declarations: every value must fit its type
 * and its `where` bounds. An input left unset takes its default, so nothing is required here.
 */
export function previewInputsResolution(
  values: PreviewInputsForm,
  declarations: readonly TariffInputDeclaration[],
): ResolverResult<PreviewInputsForm> {
  const issues = tariffInputIssues(declarations, values.inputs, translate);
  if (issues.length === 0) return { values, errors: {} };
  const errors: Record<string, unknown> = {};
  for (const issue of issues) {
    let node = errors;
    const path = ["inputs", ...issue.path.map(String)];
    path.slice(0, -1).forEach((part) => {
      node[part] = (node[part] as Record<string, unknown> | undefined) ?? {};
      node = node[part] as Record<string, unknown>;
    });
    node[path[path.length - 1] as string] = { type: "custom", message: issue.message };
  }
  return { values: {}, errors: errors as FieldErrors<PreviewInputsForm> };
}
