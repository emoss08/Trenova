import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { READINESS_KEYS, tariffReadiness, type TariffReadinessInput } from "../readiness";

const base: TariffReadinessInput = {
  mode: "edit",
  name: "Per mile",
  description: "Rates by the mile",
  sourceEmpty: false,
  errors: 0,
  warnings: 0,
  tests: { status: "empty" },
  dirty: false,
};

function testsRow(input: TariffReadinessInput) {
  return tariffReadiness(input, translate).find((check) => check.key === READINESS_KEYS.tests);
}

describe("tariffReadiness tests row", () => {
  it("warns when a tariff has no tests", () => {
    expect(testsRow(base)?.status).toBe("warn");
  });

  it("blocks when a library has no tests, as the server's readiness does", () => {
    expect(testsRow({ ...base, testsRequired: true })).toEqual({
      key: READINESS_KEYS.tests,
      label: "Tests pass",
      status: "fail",
      detail: "Add at least one test",
    });
  });

  it("blocks on failing tests and passes when all pass, for either kind", () => {
    const ran = (passed: number, failed: number): TariffReadinessInput["tests"] => ({
      status: "ran",
      run: { tests: [], needs: [], error: null },
      passed,
      failed,
      stale: false,
      cells: "ready",
      retryCells: () => {},
    });
    for (const testsRequired of [false, true]) {
      expect(testsRow({ ...base, testsRequired, tests: ran(1, 1) })?.status).toBe("fail");
      expect(testsRow({ ...base, testsRequired, tests: ran(2, 0) })).toMatchObject({
        status: "pass",
        detail: "2 of 2",
      });
    }
  });

  it("blocks until the errors are fixed", () => {
    expect(testsRow({ ...base, errors: 2, tests: { status: "blocked", errors: 2 } })).toMatchObject(
      { status: "fail", detail: "Fix the errors to run the tests" },
    );
  });
});

describe("tariffReadiness coverage row", () => {
  const ran: TariffReadinessInput["tests"] = {
    status: "ran",
    run: { tests: [], needs: [], error: null },
    passed: 2,
    failed: 0,
    stale: false,
    cells: "ready",
    retryCells: () => {},
  };
  const coverage = {
    reached: 2,
    total: 4,
    regions: [
      { range: { from: 10, to: 40 }, kind: "if" as const, hits: 3 },
      { range: { from: 46, to: 90 }, kind: "else" as const, hits: 0 },
      { range: { from: 50, to: 80 }, kind: "charge" as const, hits: 0 },
      { range: { from: 95, to: 120 }, kind: "charge" as const, hits: 1 },
    ],
  };
  const row = (input: TariffReadinessInput) =>
    tariffReadiness(input, translate).find((check) => check.key === READINESS_KEYS.coverage);

  it("warns, never blocks, with each outermost region no test reaches", () => {
    expect(row({ ...base, tests: ran, coverage })).toEqual({
      key: READINESS_KEYS.coverage,
      label: "Tests reach every region",
      status: "warn",
      detail: "Tests reach 2 of 4 regions",
      regions: [{ range: { from: 46, to: 90 }, kind: "else" }],
    });
  });

  it("passes when every region is reached", () => {
    const all = {
      ...coverage,
      reached: 4,
      regions: coverage.regions.map((region) => ({ ...region, hits: 1 })),
    };
    expect(row({ ...base, tests: ran, coverage: all })).toMatchObject({ status: "pass" });
  });

  it("says nothing without coverage, or while a run is on its way", () => {
    expect(row({ ...base, tests: ran, coverage: null })).toBeUndefined();
    expect(row({ ...base, tests: { ...ran, stale: true }, coverage })).toBeUndefined();
    expect(row({ ...base, tests: { status: "empty" }, coverage })).toBeUndefined();
  });
});
