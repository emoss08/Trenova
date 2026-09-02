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
    currentAmount: 100,
    candidateAmount: 150,
    delta: 50,
    deltaPct: 50,
    guardrailApplied: true,
  },
  {
    shipmentId: "shp_2",
    proNumber: "",
    currentAmount: 0,
    candidateAmount: 0,
    delta: 0,
    deltaPct: 0,
    currentError: "no rating detail",
    candidateError: 'bad "lookup"',
    guardrailApplied: false,
  },
];

describe("backtestCsv", () => {
  it("writes one row per shipment with amounts, delta, clamp, and errors", () => {
    const lines = backtestCsv(rows).split("\r\n");
    expect(lines[0]).toBe(
      "Shipment ID,Pro #,Current,Candidate,Delta,Delta %,Guardrail Clamped,Current Error,Candidate Error",
    );
    expect(lines[1]).toBe("shp_1,PRO-1,100,150,50,50,Yes,,");
    expect(lines[2]).toBe('shp_2,,0,0,0,0,No,no rating detail,"bad ""lookup"""');
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
