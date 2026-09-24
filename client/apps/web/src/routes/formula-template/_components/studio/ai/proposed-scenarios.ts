import type { PricedScenario } from "@/types/page-draft";
import type { FormulaTestCaseInput } from "@trenova/shared/types/formula-template";

/** A cent: the engine priced the scenario, so the saved case should match it exactly. */
export const PROPOSED_SCENARIO_TOLERANCE = 0.01;

/** The engine's amount as a number, or null when it priced nothing usable. */
export function scenarioAmount(scenario: PricedScenario): number | null {
  const raw = scenario.amount.trim();
  if (!scenario.valid || raw === "") {
    return null;
  }
  const amount = Number(raw);
  return Number.isFinite(amount) ? amount : null;
}

export function acceptableScenarios(scenarios: readonly PricedScenario[]): PricedScenario[] {
  return scenarios.filter((scenario) => scenarioAmount(scenario) !== null);
}

export function scenarioToTestCaseInput(scenario: PricedScenario): FormulaTestCaseInput {
  return {
    name: scenario.name,
    description: scenario.description,
    variables: scenario.variables,
    expectedAmount: scenarioAmount(scenario) ?? 0,
    tolerance: PROPOSED_SCENARIO_TOLERANCE,
  };
}
