import { formulaCheck, type FormulaCheck, type FormulaExpect } from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type { KnownIdentifiers } from "./known-identifiers";

/** How many recent checks each knowledge keeps: the main formula and its breakdown lines. */
const RECENT_CHECKS = 24;

const recent = new WeakMap<KnownIdentifiers, Map<string, FormulaCheck>>();

/**
 * `formula.check` against what the editor knows. The studio's analysis and the
 * editor's linter ask for the same text against the same knowledge within a
 * keystroke of each other, so the answer is kept per knowledge for the last
 * few texts; new knowledge (a schema arriving, an input renamed) starts afresh.
 */
export function checkExpression(
  rules: RulesRuntime,
  source: string,
  known: KnownIdentifiers,
  expect: FormulaExpect = "amount",
): FormulaCheck {
  let checks = recent.get(known);
  if (!checks) {
    checks = new Map();
    recent.set(known, checks);
  }
  const key = `${expect}\u0000${source}`;
  const cached = checks.get(key);
  if (cached) {
    checks.delete(key);
    checks.set(key, cached);
    return cached;
  }
  const check = formulaCheck(rules, { source, expect, schema: known.schema });
  checks.set(key, check);
  if (checks.size > RECENT_CHECKS) {
    const oldest = checks.keys().next().value;
    if (oldest !== undefined) checks.delete(oldest);
  }
  return check;
}
