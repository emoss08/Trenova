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
  it("puts the server's refusal first", () => {
    expect(
      proposalCheck({ check: { ok: false, diagnostics: [finding] } }, clean, passing),
    ).toEqual({ kind: "server", problem: finding });
  });

  it("then the browser's check against this Studio's context", () => {
    expect(proposalCheck({ check: { ok: true, diagnostics: [] } }, failing, passing)).toEqual({
      kind: "problem",
      problem: finding,
    });
  });

  it("then failing tests, and only once the run is current", () => {
    const red: TariffTestsState = { ...passing, passed: 1, failed: 1 };
    expect(proposalCheck({ check: { ok: true, diagnostics: [] } }, clean, red)).toEqual({
      kind: "failing",
      failed: 1,
      total: 2,
    });
    expect(
      proposalCheck({ check: { ok: true, diagnostics: [] } }, clean, { ...red, stale: true }),
    ).toEqual({ kind: "ok" });
    expect(proposalCheck({ check: { ok: true, diagnostics: [] } }, clean, passing)).toEqual({
      kind: "ok",
    });
  });
});
