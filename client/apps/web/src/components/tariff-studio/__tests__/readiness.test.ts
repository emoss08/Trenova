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
