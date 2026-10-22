import { translateMessage } from "@trenova/shared/i18n/message";
import {
  matrixCoverage,
  type MatrixCellInput,
  type MatrixDimensionInput,
} from "@trenova/shared/rules/matrix";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { corpusCase, expectCorpusOutput } from "@trenova/shared/rules/__tests__/corpus";
import { describe, expect, it } from "vitest";

type CoverageInput = { dimensions: MatrixDimensionInput[]; cells: MatrixCellInput[] };

function coverageCase(name: string) {
  const testCase = corpusCase("matrix", name);
  const input = testCase.input as CoverageInput;
  expectCorpusOutput(testCase, { issues: matrixCoverage(rulesSync(), input) });
}

describe("matrixCoverage", () => {
  it("reports the hole between two bands", () => {
    coverageCase("coverage (TS): the hole between two bands");
  });

  it("checks each zone's weight breaks on their own and lists where a finding repeats", () => {
    coverageCase("coverage: each zone's weight breaks are checked on their own");
  });

  it("finds both overlaps of a wide band over two narrow ones", () => {
    coverageCase(
      "coverage: a wide band covering two narrow ones overlaps both and leaves no false gap",
    );
  });

  it("reports a duplicate coordinate once however often it repeats", () => {
    coverageCase(
      "coverage (TS): two rates at one coordinate, reported once however often it repeats",
    );
  });

  it("names an overlap's bands as messages of their own, which translate", () => {
    const testCase = corpusCase(
      "matrix",
      "coverage: an unbounded band in the middle overlaps everything above it",
    );
    const [issue] = matrixCoverage(rulesSync(), testCase.input as CoverageInput);
    expect(translateMessage(issue.message)).toBe(
      "Weight bands 0 and over and 1,000 – under 2,000 overlap, so the same shipment matches both.",
    );
  });

  it("leaves an axis without a valid position out instead of refusing the matrix", () => {
    const issues = matrixCoverage(rulesSync(), {
      dimensions: [
        { position: 0, kind: "WeightBreak", matchMode: "Range", label: "Weight" },
        { position: 7, kind: "Zone", matchMode: "Exact", label: "Zone" },
      ],
      cells: [{ d0Min: 0, d0Max: "1000" }, { d0Min: "5000" }],
    });
    expect(issues.map((issue) => translateMessage(issue.message))).toEqual([
      "Weight jumps from 1,000 to 5,000 with nothing in between.",
    ]);
  });
});
