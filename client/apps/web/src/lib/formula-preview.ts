import type {
  FormulaMessage,
  ScopedFormulaDiagnostic,
  TestExpressionResponse,
} from "@trenova/shared/types/formula-template";

/** The scope of the template's main expression in the preview's findings. */
export const EXPRESSION_SCOPE = "expression";

type PreviewFindings = Pick<TestExpressionResponse, "diagnostics" | "error">;

/**
 * Why a preview did not price, in the engine's words: the evaluation error
 * when the formula ran and failed, otherwise the first error the check found
 * on the expression or a breakdown line. Null for a preview that priced.
 */
export function previewFailure(result: PreviewFindings | undefined): FormulaMessage | null {
  if (!result) return null;
  if (result.error) return result.error;
  return result.diagnostics.find((diagnostic) => diagnostic.severity === "error") ?? null;
}

/** The warnings the server found, on every expression it checked or only on one scope. */
export function previewWarnings(
  result: PreviewFindings | undefined,
  scope?: string,
): ScopedFormulaDiagnostic[] {
  return (result?.diagnostics ?? []).filter(
    (diagnostic) =>
      diagnostic.severity === "warning" && (scope === undefined || diagnostic.scope === scope),
  );
}

/** The fields some expression reads without a guard though they can be empty, by name. */
export function unguardedFields(result: PreviewFindings | undefined): string[] {
  const names = new Set<string>();
  for (const diagnostic of result?.diagnostics ?? []) {
    if (diagnostic.code === "nullable" && diagnostic.name) names.add(diagnostic.name);
  }
  return [...names];
}
