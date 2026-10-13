import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { studioReadiness, type StudioReadinessInput } from "../readiness";

const t = ((message: string, ...args: unknown[]) =>
  message.replace(
    /\{(\d+)(?:,[^}]*\{#\s*([^}]*)\}[^}]*\{#\s*([^}]*)\}\})?\}/g,
    (_m, index, one, other) => {
      const value = args[Number(index)];
      if (one !== undefined) return `${String(value)} ${value === 1 ? one : other}`;
      return String(value);
    },
  )) as TranslateFn;

const ready: StudioReadinessInput = {
  mode: "edit",
  name: "Linehaul",
  description: "Per mile",
  expressionEmpty: false,
  expressionErrors: 0,
  previewValid: true,
  unguardedFields: [],
  scenarioTotal: 3,
  scenarioPassed: 3,
  breakdownLines: 0,
  reconciliation: null,
  dirty: false,
};

const statuses = (input: StudioReadinessInput) =>
  Object.fromEntries(studioReadiness(input, t).map((check) => [check.key, check.status]));

describe("studioReadiness", () => {
  it("uses the server's check keys and passes a clean, saved, tested template", () => {
    expect(statuses(ready)).toEqual({
      details: "pass",
      expression: "pass",
      nullableFields: "pass",
      scenarios: "pass",
    });
  });

  it("blocks on a missing description, an expression that fails on the preview load, and no scenarios", () => {
    const checks = studioReadiness(
      { ...ready, description: " ", previewValid: false, scenarioTotal: 0, scenarioPassed: 0 },
      t,
    );
    expect(checks.map((check) => [check.key, check.status, check.detail])).toEqual([
      ["details", "fail", "Description is missing"],
      ["expression", "fail", "Fails on the preview load"],
      ["nullableFields", "pass", undefined],
      ["scenarios", "fail", "Add at least one"],
    ]);
  });

  it("only warns for unguarded fields and a breakdown that does not reconcile", () => {
    expect(
      statuses({
        ...ready,
        unguardedFields: ["weight", "pieces"],
        breakdownLines: 2,
        reconciliation: {
          sum: 10,
          residual: 5,
          balanced: false,
          clampMismatch: false,
          failedCount: 0,
        },
      }),
    ).toEqual({
      details: "pass",
      expression: "pass",
      nullableFields: "warn",
      scenarios: "pass",
      breakdown: "warn",
    });
  });

  it("leaves scenarios out in create mode and asks an edited template to be saved first", () => {
    expect(Object.keys(statuses({ ...ready, mode: "create", dirty: true }))).toEqual([
      "details",
      "expression",
      "nullableFields",
    ]);
    expect(statuses({ ...ready, dirty: true }).saved).toBe("fail");
  });

  it("fails scenarios when any of them fail", () => {
    expect(statuses({ ...ready, scenarioPassed: 2 }).scenarios).toBe("fail");
  });
});
