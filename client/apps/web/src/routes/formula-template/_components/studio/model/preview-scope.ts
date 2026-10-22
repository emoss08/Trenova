import { EXPRESSION_SCOPE } from "@/lib/formula-preview";
import type {
  BreakdownDefinitionInput,
  TestExpressionRequest,
} from "@trenova/shared/types/formula-template";

const BREAKDOWN_SCOPE = /^breakdowns\[(\d+)\]\.expression$/;

/** Where in the form a server finding belongs, and the text the server checked there. */
export type ScopeTarget = {
  /** The form field path: `expression` or `breakdownDefinitions.<n>.expression`. */
  path: "expression" | `breakdownDefinitions.${number}.expression`;
  /** The expression as the preview request sent it. */
  checked: string;
};

/**
 * Maps a preview finding's scope (`expression`, `breakdowns[1].expression`,
 * indexed into the request the preview sent) onto the form. The request
 * leaves out blank breakdown lines, so a line is found by its key, not its
 * position. Null when the scope names nothing the form still holds.
 */
export function scopeTarget(
  scope: string,
  request: TestExpressionRequest | null,
  breakdownDefinitions: readonly BreakdownDefinitionInput[],
): ScopeTarget | null {
  if (!request) return null;
  if (scope === EXPRESSION_SCOPE) return { path: "expression", checked: request.expression };
  const match = BREAKDOWN_SCOPE.exec(scope);
  const sent = match ? request.breakdowns?.[Number(match[1])] : undefined;
  if (!sent) return null;
  const index = breakdownDefinitions.findIndex((line) => line.name === sent.name);
  if (index === -1) return null;
  return { path: `breakdownDefinitions.${index}.expression`, checked: sent.expression };
}
