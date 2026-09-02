import type { ProposedScenario } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { acceptableScenarios, scenarioToTestCaseInput } from "../proposed-scenarios";

const priced: ProposedScenario = {
  name: "Short haul",
  description: "A 100 mile lane",
  variables: { baseRate: 2, totalDistance: 100 },
  expectedAmount: 200,
  valid: true,
  error: null,
};

const broken: ProposedScenario = {
  name: "Broken",
  description: "",
  variables: {},
  expectedAmount: null,
  valid: false,
  error: "invalid operation",
};

describe("scenarioToTestCaseInput", () => {
  it("turns a priced scenario into a test case with a cent of tolerance", () => {
    expect(scenarioToTestCaseInput(priced)).toEqual({
      name: "Short haul",
      description: "A 100 mile lane",
      variables: { baseRate: 2, totalDistance: 100 },
      expectedAmount: 200,
      tolerance: 0.01,
    });
  });
});

describe("acceptableScenarios", () => {
  it("keeps only scenarios the engine could price", () => {
    const zeroPriced: ProposedScenario = { ...priced, name: "Free", expectedAmount: 0 };
    expect(acceptableScenarios([priced, broken, zeroPriced])).toEqual([priced, zeroPriced]);
  });
});
