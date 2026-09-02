const CENT = 0.005;

type BreakdownLine = {
  name: string;
  label?: string;
  amount: number;
  error?: string;
};

type BreakdownDefinitionLike = { name: string };

export type BreakdownReconciliation = {
  /** Sum of the lines that evaluated. */
  sum: number;
  /** What the total holds that no line accounts for; negative when lines exceed it. */
  residual: number;
  /** The lines explain the total to the cent. */
  balanced: boolean;
  /** A guardrail moved the total but the lines still add up to the raw amount. */
  clampMismatch: boolean;
  failedCount: number;
};

function roundCents(value: number): number {
  return Math.round(value * 100) / 100;
}

/**
 * Compares itemized lines with the amount actually charged. The lines exist
 * so an invoice can explain the total; a residual is a line the author forgot,
 * and a clamp mismatch is a floor or ceiling the lines never heard about.
 */
export function reconcileBreakdown({
  total,
  rawAmount,
  guardrailApplied = false,
  lines,
}: {
  total: number;
  rawAmount: number;
  guardrailApplied?: boolean;
  lines: BreakdownLine[];
}): BreakdownReconciliation {
  const evaluated = lines.filter((line) => !line.error);
  const sum = roundCents(evaluated.reduce((acc, line) => acc + line.amount, 0));
  const residual = roundCents(total - sum);
  const balanced = Math.abs(total - sum) < CENT;
  const clampMismatch =
    guardrailApplied && Math.abs(rawAmount - sum) < CENT && Math.abs(total - sum) >= CENT;

  return {
    sum,
    residual: balanced ? 0 : residual,
    balanced,
    clampMismatch,
    failedCount: lines.length - evaluated.length,
  };
}

/**
 * Maps failed breakdown lines back to the form path of the definition that
 * produced them, so the error lands under the line's own editor.
 */
export function breakdownErrorsByPath(
  items: BreakdownLine[],
  definitions: BreakdownDefinitionLike[],
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const item of items) {
    if (!item.error) continue;
    const index = definitions.findIndex((definition) => definition.name === item.name);
    if (index === -1) continue;
    errors[`breakdownDefinitions.${index}.expression`] = item.error;
  }
  return errors;
}
