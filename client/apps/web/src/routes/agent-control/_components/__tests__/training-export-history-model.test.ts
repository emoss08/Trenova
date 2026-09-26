import { describe, expect, it } from "vitest";
import { trainingExportTotals } from "../training-export-history-model";

const entry = (examples: number, exportedAt: number) => ({
  exportId: `aitx_${exportedAt}`,
  examples,
  trainExamples: examples,
  validationExamples: 0,
  consentGrantedAt: 100,
  exportedAt,
});

describe("trainingExportTotals", () => {
  it("is zero with no exports", () => {
    expect(trainingExportTotals([])).toEqual({ exports: 0, corrections: 0, lastExportedAt: null });
  });

  it("adds corrections across exports and finds the latest, whatever the order", () => {
    expect(trainingExportTotals([entry(4, 200), entry(9, 500), entry(1, 300)])).toEqual({
      exports: 3,
      corrections: 14,
      lastExportedAt: 500,
    });
  });
});
