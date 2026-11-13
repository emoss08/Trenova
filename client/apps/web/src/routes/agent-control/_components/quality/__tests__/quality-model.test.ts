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
  trialProviders,
  versionBefore,
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
  autoActivate: true,
  autoCasesPerAgent: 100,
  autoCaseDays: 30,
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
      autoActivate: true,
      autoCasesPerAgent: 100,
      version: control.version,
    });
  });

  // An empty timezone is left out so the server keeps the organization's
  // own; the version read is sent so a newer save is not overwritten.
  it("saves at the version it opened on, in the server's units", () => {
    const input = toUpdateInput({
      ...toFormValues(control),
      runHour: "23",
      timezone: "  ",
      judgeSamplePercent: 35,
      version: 4,
    });

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
      autoActivate: true,
      autoCasesPerAgent: 100,
    });
    expect("timezone" in input).toBe(false);
    expect(
      toUpdateInput({ ...toFormValues(control), timezone: " America/Chicago " }).timezone,
    ).toBe("America/Chicago");
  });

  // The days an automatic case lives are the server's to say, so they are read
  // and never sent back.
  it("saves whether the golden set keeps itself and how many cases an agent keeps", () => {
    const input = toUpdateInput({
      ...toFormValues(control),
      autoActivate: false,
      autoCasesPerAgent: 200,
    });

    expect(input.autoActivate).toBe(false);
    expect(input.autoCasesPerAgent).toBe(200);
    expect("autoCaseDays" in input).toBe(false);
  });

  it("refuses to keep no automatic cases or more than five hundred", () => {
    const values = toFormValues(control);

    expect(qualityControlSchema.safeParse({ ...values, autoCasesPerAgent: 0 }).success).toBe(false);
    expect(qualityControlSchema.safeParse({ ...values, autoCasesPerAgent: 501 }).success).toBe(
      false,
    );
    expect(qualityControlSchema.safeParse({ ...values, autoCasesPerAgent: 500 }).success).toBe(
      true,
    );
  });

  it("refuses a monthly budget below one night's", () => {
    expect(qualityControlSchema.parse(toFormValues(control)).version).toBe(control.version);
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

describe("the version before a change", () => {
  it("is the version the agent was saved from", () => {
    expect(
      versionBefore({
        fingerprintChanges: [
          { field: "model", from: "claude-haiku-5-5", to: "claude-sonnet-5-5" },
          { field: "definitionVersion", from: "4", to: "6" },
        ],
      }),
    ).toBe(4);
  });

  // A model or provider that changed under the agent leaves no earlier
  // version of the agent to go back to.
  it("is nothing when the agent itself was not saved", () => {
    expect(
      versionBefore({
        fingerprintChanges: [{ field: "provider", from: "", to: "" }],
      }),
    ).toBeNull();
    expect(versionBefore({ fingerprintChanges: [] })).toBeNull();
    expect(versionBefore(null)).toBeNull();
  });

  it("is nothing when the version on record is not a version number", () => {
    expect(
      versionBefore({ fingerprintChanges: [{ field: "definitionVersion", from: "", to: "2" }] }),
    ).toBeNull();
    expect(
      versionBefore({ fingerprintChanges: [{ field: "definitionVersion", from: "0", to: "1" }] }),
    ).toBeNull();
  });
});

describe("the providers a golden set can be tried on", () => {
  const provider = (id: string, enabled: boolean, tasks: string[]) => ({
    id,
    name: `Provider ${id}`,
    model: `model-${id}`,
    enabled,
    tasks,
  });

  it("are the ones turned on that serve conversations, in routing order", () => {
    expect(
      trialProviders([
        provider("b", true, ["AssistantChat", "Embedding"]),
        provider("off", false, ["AssistantChat"]),
        provider("embed", true, ["Embedding"]),
        provider("a", true, ["AssistantChat"]),
      ]),
    ).toEqual([
      { id: "b", name: "Provider b", model: "model-b" },
      { id: "a", name: "Provider a", model: "model-a" },
    ]);
  });

  it("are none when no provider serves conversations", () => {
    expect(trialProviders([])).toEqual([]);
    expect(trialProviders([provider("embed", true, ["Embedding"])])).toEqual([]);
  });
});
