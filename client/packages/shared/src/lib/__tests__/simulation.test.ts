import { describe, expect, it } from "vitest";
import {
  deltaDirection,
  isTerminal,
  measuredAnything,
  problemRules,
  ruleCoverageNote,
  ruleOutcomeLabel,
  runProgress,
  shouldPoll,
  summaryHeadline,
} from "../simulation";
import type { RateSimulation, RateSimulationSummary, RuleCoverage } from "../../types/rate";

function summary(overrides: Partial<RateSimulationSummary> = {}): RateSimulationSummary {
  return {
    shipmentCount: 100,
    evaluatedCount: 100,
    changedCount: 40,
    increasedCount: 30,
    decreasedCount: 10,
    errorCount: 0,
    beforeTotal: 200000,
    afterTotal: 208000,
    totalDelta: 8000,
    totalDeltaPct: 4,
    maxIncrease: 500,
    maxDecrease: -200,
    ...overrides,
  };
}

function simulation(overrides: Partial<RateSimulation> = {}): RateSimulation {
  return {
    rateAgreementId: "rag_1",
    name: "Q3 GRI",
    description: "",
    status: "Completed",
    partyType: "Customer",
    sampleFrom: 1,
    sampleTo: 2,
    sampleLimit: 0,
    summary: summary(),
    ruleCoverage: [],
    error: "",
    workflowId: "",
    ...overrides,
  } as RateSimulation;
}

function coverage(overrides: Partial<RuleCoverage> = {}): RuleCoverage {
  return {
    ruleId: "ragr_1",
    label: "chicago-atlanta",
    laneKey: "CS:chicago>CS:atlanta",
    outcome: "Won",
    wonCount: 10,
    lostCount: 0,
    lostToLabel: "",
    ...overrides,
  } as RuleCoverage;
}

describe("isTerminal", () => {
  it("is true once a run has stopped, however it stopped", () => {
    expect(isTerminal(simulation({ status: "Completed" }))).toBe(true);
    expect(isTerminal(simulation({ status: "Failed" }))).toBe(true);
    expect(isTerminal(simulation({ status: "Canceled" }))).toBe(true);
  });

  it("is false while a run is still going", () => {
    expect(isTerminal(simulation({ status: "Pending" }))).toBe(false);
    expect(isTerminal(simulation({ status: "Running" }))).toBe(false);
  });
});

describe("shouldPoll", () => {
  it("keeps asking while a run is going", () => {
    expect(shouldPoll(simulation({ status: "Running" }))).toBe(true);
  });

  // Polling a finished run forever is how a background job quietly becomes a
  // load on the database.
  it("stops once the run has finished", () => {
    expect(shouldPoll(simulation({ status: "Completed" }))).toBe(false);
  });

  it("does not poll for a simulation that does not exist yet", () => {
    expect(shouldPoll(undefined)).toBe(false);
  });
});

describe("measuredAnything", () => {
  // A completed run over zero shipments reports a zero delta, which reads as
  // "this change costs nothing" rather than "this measured nothing".
  it("is false for a run that priced no shipments", () => {
    expect(measuredAnything(simulation({ summary: summary({ evaluatedCount: 0 }) }))).toBe(false);
  });

  it("is true once anything was priced", () => {
    expect(measuredAnything(simulation())).toBe(true);
  });

  it("is false before a run has a summary at all", () => {
    expect(measuredAnything(simulation({ summary: null }))).toBe(false);
  });
});

describe("deltaDirection", () => {
  it("reads a positive total as an increase", () => {
    expect(deltaDirection(summary({ totalDelta: 8000 }))).toBe("up");
  });

  it("reads a negative total as a decrease", () => {
    expect(deltaDirection(summary({ totalDelta: -8000 }))).toBe("down");
  });

  it("reads no movement as flat rather than as a decrease", () => {
    expect(deltaDirection(summary({ totalDelta: 0 }))).toBe("flat");
  });
});

describe("runProgress", () => {
  it("counts both the priced and the failed as done", () => {
    expect(
      runProgress(
        simulation({
          summary: summary({ shipmentCount: 100, evaluatedCount: 40, errorCount: 10 }),
        }),
      ),
    ).toBe(0.5);
  });

  // An empty bar reads as "no progress", which is wrong for a job that has not
  // been told how much there is to do.
  it("reports nothing rather than zero before the total is known", () => {
    expect(runProgress(simulation({ summary: null }))).toBeNull();
    expect(runProgress(simulation({ summary: summary({ shipmentCount: 0 }) }))).toBeNull();
  });

  it("never exceeds one", () => {
    expect(
      runProgress(
        simulation({
          summary: summary({ shipmentCount: 10, evaluatedCount: 10, errorCount: 5 }),
        }),
      ),
    ).toBe(1);
  });
});

describe("ruleCoverageNote", () => {
  // The two failure states have different fixes, and one message for both would
  // send somebody looking in the wrong place.
  it("says a lane that never matched may be written for freight you do not move", () => {
    expect(ruleCoverageNote(coverage({ outcome: "NeverFired", wonCount: 0 }))).toContain(
      "do not move",
    );
  });

  it("names what shadows a lane that always loses", () => {
    const note = ruleCoverageNote(
      coverage({ outcome: "Lost", wonCount: 0, lostCount: 12, lostToLabel: "chicago-atlanta" }),
    );

    expect(note).toContain("chicago-atlanta");
    expect(note).toContain("12");
  });

  it("still explains a losing lane when nothing is named as beating it", () => {
    const note = ruleCoverageNote(coverage({ outcome: "Lost", wonCount: 0, lostCount: 12 }));

    expect(note).toContain("narrower");
  });

  it("mentions the losses of a rule that mostly wins", () => {
    expect(ruleCoverageNote(coverage({ outcome: "Won", wonCount: 8, lostCount: 3 }))).toContain(
      "outranked on 3",
    );
  });

  it("says nothing about losses for a rule that never lost", () => {
    expect(ruleCoverageNote(coverage({ outcome: "Won", wonCount: 8 }))).not.toContain("outranked");
  });
});

describe("ruleOutcomeLabel", () => {
  it("names each outcome in the words somebody reads", () => {
    expect(ruleOutcomeLabel("NeverFired")).toBe("Never matched");
    expect(ruleOutcomeLabel("Lost")).toBe("Always outranked");
    expect(ruleOutcomeLabel("Won")).toBe("Priced shipments");
  });
});

describe("problemRules", () => {
  it("keeps only the rules that did nothing", () => {
    const rows = problemRules([
      coverage({ ruleId: "a", outcome: "Won" }),
      coverage({ ruleId: "b", outcome: "Lost" }),
      coverage({ ruleId: "c", outcome: "NeverFired" }),
    ]);

    expect(rows.map((row) => row.ruleId)).toEqual(["b", "c"]);
  });

  it("has nothing to report before a run has coverage", () => {
    expect(problemRules(null)).toEqual([]);
  });
});

describe("summaryHeadline", () => {
  it("states the move in the terms somebody decides on", () => {
    const headline = summaryHeadline(simulation());

    expect(headline).toContain("100 shipments");
    expect(headline).toContain("4%");
    expect(headline).toContain("more");
  });

  it("reads a decrease as less rather than as a negative percentage", () => {
    const headline = summaryHeadline(
      simulation({ summary: summary({ totalDelta: -8000, totalDeltaPct: -4 }) }),
    );

    expect(headline).toContain("4%");
    expect(headline).toContain("less");
    expect(headline).not.toContain("-4");
  });

  // A run that priced nothing and a run that changed nothing look identical as
  // numbers and mean opposite things.
  it("distinguishes a run that measured nothing from one that changed nothing", () => {
    const measuredNothing = summaryHeadline(
      simulation({ summary: summary({ evaluatedCount: 0 }) }),
    );
    const changedNothing = summaryHeadline(
      simulation({ summary: summary({ totalDelta: 0, totalDeltaPct: 0 }) }),
    );

    expect(measuredNothing).toContain("nothing to compare");
    expect(changedNothing).toContain("exactly what they were billed");
  });

  it("shows the reason a failed run gives", () => {
    expect(
      summaryHeadline(simulation({ status: "Failed", error: "the agreement was deleted" })),
    ).toContain("the agreement was deleted");
  });

  it("says a run is still going rather than reporting a partial delta as final", () => {
    expect(summaryHeadline(simulation({ status: "Running" }))).toContain("Replaying");
  });
});
