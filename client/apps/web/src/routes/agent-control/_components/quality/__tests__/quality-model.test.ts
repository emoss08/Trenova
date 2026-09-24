import type { AgentQualityControl } from "@/lib/graphql/agent-quality";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { describe, expect, it } from "vitest";
import {
  SUITE_RUN_STATUS,
  centsToDecimal,
  decimalToCents,
  formatDelta,
  formatShare,
  qualityControlSchema,
  sparklineValues,
  toFormValues,
  toUpdateInput,
} from "../quality-model";

const control: AgentQualityControl = {
  id: null,
  enabled: true,
  runHourLocal: 2,
  timezone: "",
  maxCasesPerAgent: 50,
  nightlyBudgetUsd: "5.00",
  monthlyBudgetUsd: "50.00",
  judgeEnabled: false,
  judgeSampleRate: 0.2,
  regressionThreshold: 0.1,
  minCases: 10,
  forceRerunDays: 7,
  version: 0,
  updatedAt: 0,
};

describe("suite run status", () => {
  // The phase decides the tone, so a new status cannot pick a colour.
  it("maps each run status onto its lifecycle phase", () => {
    expect(SUITE_RUN_STATUS.Completed.phase).toBe("complete");
    expect(SUITE_RUN_STATUS.Skipped.phase).toBe("closed");
    expect(SUITE_RUN_STATUS.BudgetStopped.phase).toBe("attention");
    expect(SUITE_RUN_STATUS.Failed.phase).toBe("failed");
    expect(SUITE_RUN_STATUS.Running.phase).toBe("active");
    expect(phaseTone(SUITE_RUN_STATUS.Failed.phase)).toBe("danger");
  });
});

describe("formatting", () => {
  it("shows a share as a whole percentage, and nothing as a dash", () => {
    expect(formatShare(0.875)).toBe("88%");
    expect(formatShare(0)).toBe("0%");
    expect(formatShare(null)).toBe("—");
    expect(formatShare(undefined)).toBe("—");
    expect(formatShare(Number.NaN)).toBe("—");
  });

  it("signs a change in satisfaction and gives it a tone", () => {
    expect(formatDelta(0.05)).toEqual({ text: "+5 pts", tone: "success" });
    expect(formatDelta(-0.12)).toEqual({ text: "-12 pts", tone: "danger" });
    expect(formatDelta(0.001)).toEqual({ text: "±0 pts", tone: "muted" });
    expect(formatDelta(null)).toEqual({ text: "—", tone: "muted" });
  });

  it("draws the quality line in percentage points, oldest first", () => {
    expect(sparklineValues([{ qualityScore: 0.9 }, { qualityScore: 0.8125 }])).toEqual([90, 81.3]);
    expect(sparklineValues([])).toEqual([]);
  });

  it("moves money between cents and the Decimal scalar without losing a cent", () => {
    expect(centsToDecimal(500)).toBe("5.00");
    expect(centsToDecimal(1)).toBe("0.01");
    expect(decimalToCents("50.00")).toBe(5000);
    expect(decimalToCents("0.105")).toBe(11);
    expect(decimalToCents("not money")).toBe(0);
  });
});

describe("quality settings form", () => {
  it("opens on the saved controls in the units a person edits", () => {
    expect(toFormValues(control)).toEqual({
      enabled: true,
      runHour: "2",
      timezone: "",
      maxCasesPerAgent: 50,
      nightlyBudgetCents: 500,
      monthlyBudgetCents: 5000,
      judgeEnabled: false,
      judgeSamplePercent: 20,
      regressionThresholdPoints: 10,
      minCases: 10,
      forceRerunDays: 7,
    });
  });

  // An empty timezone is left out so the server keeps the organization's
  // own; the version read is sent so a newer save is not overwritten.
  it("saves at the version it opened on, in the server's units", () => {
    const input = toUpdateInput(
      { ...toFormValues(control), runHour: "23", timezone: "  ", judgeSamplePercent: 35 },
      4,
    );

    expect(input).toEqual({
      version: 4,
      enabled: true,
      runHourLocal: 23,
      maxCasesPerAgent: 50,
      nightlyBudgetUsd: "5.00",
      monthlyBudgetUsd: "50.00",
      judgeEnabled: false,
      judgeSampleRate: 0.35,
      regressionThreshold: 0.1,
      minCases: 10,
      forceRerunDays: 7,
    });
    expect("timezone" in input).toBe(false);
    expect(
      toUpdateInput({ ...toFormValues(control), timezone: " America/Chicago " }, 0).timezone,
    ).toBe("America/Chicago");
  });

  it("refuses a monthly budget below one night's", () => {
    const result = qualityControlSchema.safeParse({
      ...toFormValues(control),
      nightlyBudgetCents: 6000,
      monthlyBudgetCents: 5000,
    });

    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["monthlyBudgetCents"]);
  });

  it("refuses an hour that is not on the clock", () => {
    expect(
      qualityControlSchema.safeParse({ ...toFormValues(control), runHour: "24" }).success,
    ).toBe(false);
    expect(qualityControlSchema.safeParse({ ...toFormValues(control), runHour: "0" }).success).toBe(
      true,
    );
  });
});
