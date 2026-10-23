import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { READINESS_KEYS, tariffReadiness, type TariffReadinessInput } from "../readiness";
import { withCoverageRegions, withLiveChecks } from "../readiness-view";

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

describe("withCoverageRegions", () => {
  // The server's readiness (`GET /formula-templates/:id/readiness`): a `coverage` check, and the
  // response's `coverage` with each unreached region placed on its line.
  const coverage = {
    reached: 10,
    total: 13,
    unreached: [
      {
        kind: "if",
        range: { from: 1001, to: 1050 },
        at: { line: 35, column: 1 },
        excerpt: "if shipment.hazmat {",
      },
      { kind: "someday", range: { from: 1100, to: 1120 }, at: null, excerpt: null },
    ],
    more: 4,
  };
  const checks = [
    { key: "tests", label: "Tests pass", status: "pass" as const },
    {
      key: "coverage",
      label: "Tests reach every region",
      status: "warn" as const,
      detail: "Tests reach 10 of 13 regions",
    },
  ];

  it("puts the response's unreached regions on the coverage check", () => {
    expect(withCoverageRegions(checks, coverage)).toEqual([
      checks[0],
      {
        ...checks[1],
        regions: [
          {
            range: { from: 1001, to: 1050 },
            kind: "if",
            line: 35,
            excerpt: "if shipment.hazmat {",
          },
          { range: { from: 1100, to: 1120 }, kind: "someday", line: null, excerpt: null },
        ],
        more: 4,
      },
    ]);
  });

  it("leaves the checks alone without coverage", () => {
    expect(withCoverageRegions(checks, null)).toEqual(checks);
  });
});

describe("withLiveChecks", () => {
  it("adds the Studio's row only where the server sends none of that key", () => {
    const server = [{ key: "coverage", label: "Server", status: "pass" as const }];
    const live = [
      { key: "coverage", label: "Live", status: "warn" as const },
      { key: "other", label: "Other", status: "warn" as const },
    ];
    expect(withLiveChecks(server, live).map((check) => check.label)).toEqual(["Server", "Other"]);
  });
});
