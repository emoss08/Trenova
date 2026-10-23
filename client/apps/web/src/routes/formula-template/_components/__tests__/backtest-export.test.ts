import type {
  BacktestResult,
  FormulaTemplateVersion,
} from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { backtestCsv, describeVersionOption } from "../backtest-export";

const rows: BacktestResult[] = [
  {
    shipmentId: "shp_1",
    proNumber: "PRO-1",
    currentSource: "Billed",
    currentAmount: 100,
    candidateAmount: 150,
    delta: 50,
    deltaPct: 50,
  },
  {
    shipmentId: "shp_2",
    proNumber: "",
    currentSource: "Recomputed",
    currentAmount: 0,
    candidateAmount: 0,
    delta: 0,
    deltaPct: 0,
    currentError: "no rating detail",
    candidateError: 'bad "lookup"',
    candidateRefusal: "No rate for TX to OK",
  },
];

describe("backtestCsv", () => {
  it("writes one row per shipment with where the current amount came from, amounts, delta, errors and refusals", () => {
    const lines = backtestCsv(rows).split("\r\n");
    expect(lines[0]).toBe(
      "Shipment ID,Pro #,Current source,Current,Candidate,Delta,Delta %,Current error,Candidate error,Candidate refused",
    );
    expect(lines[1]).toBe("shp_1,PRO-1,Billed,100,150,50,50,,,");
    expect(lines[2]).toBe(
      'shp_2,,Recomputed,0,0,0,0,no rating detail,"bad ""lookup""",No rate for TX to OK',
    );
  });
});

describe("describeVersionOption", () => {
  it("names the version with its tags and date", () => {
    const version = {
      versionNumber: 3,
      tags: ["Stable", "Production"],
      createdAt: 1_700_000_000,
    } as FormulaTemplateVersion;
    expect(describeVersionOption(version, () => "Nov 14, 2023")).toBe(
      "v3 · Stable, Production · Nov 14, 2023",
    );
  });

  it("leaves out the tag segment when a version has none", () => {
    const version = {
      versionNumber: 1,
      tags: [],
      createdAt: 1,
    } as unknown as FormulaTemplateVersion;
    expect(describeVersionOption(version, () => "Jan 1, 1970")).toBe("v1 · Jan 1, 1970");
  });
});
