import type { TariffTestsState } from "@/components/tariff-studio/use-tariff-tests";
import { tariffCheckResultSchema, tariffDiagnosticSchema } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { proposalCheck } from "../proposal-check";

const finding = tariffDiagnosticSchema.parse({
  range: { from: 0, to: 3 },
  severity: "error",
  code: "unknown-code",
  message: { message: "{0} isn't in the accessorial charge catalog", args: ["`FSX`"] },
});
const clean = tariffCheckResultSchema.parse({ ok: true, diagnostics: [] });
const failing = tariffCheckResultSchema.parse({ ok: false, diagnostics: [finding] });
const passing: TariffTestsState = {
  status: "ran",
  run: { tests: [], error: null },
  passed: 2,
  failed: 0,
  stale: false,
};

describe("proposalCheck", () => {
  it("puts an error the browser's check finds against this Studio's context first", () => {
    const red: TariffTestsState = { ...passing, passed: 1, failed: 1 };
    expect(proposalCheck(failing, red)).toEqual({ kind: "problem", problem: finding });
  });

  it("then failing tests, and only once the run is current", () => {
    const red: TariffTestsState = { ...passing, passed: 1, failed: 1 };
    expect(proposalCheck(clean, red)).toEqual({ kind: "failing", failed: 1, total: 2 });
    expect(proposalCheck(clean, { ...red, stale: true })).toEqual({ kind: "ok" });
    expect(proposalCheck(clean, passing)).toEqual({ kind: "ok" });
  });

  it("is ok while the tests are blocked or absent, since the check is clean", () => {
    expect(proposalCheck(clean, { status: "empty" })).toEqual({ kind: "ok" });
  });
});
