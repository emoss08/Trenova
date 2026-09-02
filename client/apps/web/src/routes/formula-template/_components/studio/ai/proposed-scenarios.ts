import type {
  FormulaTestCaseInput,
  ProposedScenario,
} from "@trenova/shared/types/formula-template";

/** A cent: the engine priced the scenario, so the saved case should match it exactly. */
export const PROPOSED_SCENARIO_TOLERANCE = 0.01;

export function acceptableScenarios(scenarios: ProposedScenario[]): ProposedScenario[] {
  return scenarios.filter(
    (scenario) => scenario.valid && typeof scenario.expectedAmount === "number",
  );
}

export function scenarioToTestCaseInput(scenario: ProposedScenario): FormulaTestCaseInput {
  return {
    name: scenario.name,
    description: scenario.description,
    variables: scenario.variables,
    expectedAmount: scenario.expectedAmount ?? 0,
    tolerance: PROPOSED_SCENARIO_TOLERANCE,
  };
}
