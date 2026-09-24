import type { PricedScenario } from "@/types/page-draft";
import { describe, expect, it } from "vitest";
import { acceptableScenarios, scenarioToTestCaseInput } from "../proposed-scenarios";

/*
 * Scenarios arrive on a propose_formula draft edit, priced by the formula
 * engine on the server (pagedraft.PricedScenario). The amount is the engine's
 * decimal string, empty when the engine could not price it.
 */
const priced: PricedScenario = {
  name: "Short haul",
  description: "A 100 mile lane",
  variables: { baseRate: 2, totalDistance: 100 },
  amount: "200.00",
  valid: true,
  error: "",
};

const broken: PricedScenario = {
  name: "Broken",
  description: "",
  variables: {},
  amount: "",
  valid: false,
  error: "invalid operation",
};

describe("scenarioToTestCaseInput", () => {
  it("turns a priced scenario into a test case expecting the engine's amount, with a cent of tolerance", () => {
    expect(scenarioToTestCaseInput(priced)).toEqual({
      name: "Short haul",
      description: "A 100 mile lane",
      variables: { baseRate: 2, totalDistance: 100 },
      expectedAmount: 200,
      tolerance: 0.01,
    });
  });

  it("keeps a fractional amount exactly as the engine wrote it", () => {
    expect(scenarioToTestCaseInput({ ...priced, amount: "1234.57" }).expectedAmount).toBe(1234.57);
  });
});

describe("acceptableScenarios", () => {
  it("keeps only scenarios the engine priced, including a zero charge", () => {
    const zeroPriced: PricedScenario = { ...priced, name: "Free", amount: "0" };
    expect(acceptableScenarios([priced, broken, zeroPriced])).toEqual([priced, zeroPriced]);
  });

  it("drops a scenario marked valid that carries no amount or one that is not a number", () => {
    const unpriced: PricedScenario = { ...priced, name: "Unpriced", amount: "" };
    const garbled: PricedScenario = { ...priced, name: "Garbled", amount: "twelve" };
    const infinite: PricedScenario = { ...priced, name: "Infinite", amount: "Infinity" };
    expect(acceptableScenarios([unpriced, garbled, infinite])).toEqual([]);
  });
});
