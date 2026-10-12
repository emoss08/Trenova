import type { AITuneUp } from "@/lib/graphql/ai-control";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { tuneUpCopy, type TuneUpLabels } from "../tune-up-copy";

const t = ((template: string, ...args: unknown[]) =>
  template.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

const NOW = 1_800_000_000;
const DAY = 86_400;

const labels: TuneUpLabels = {
  tool: (name) => (name === "assign_move" ? "Assign move" : name),
  task: (task) => (task === "Embedding" ? "Embedding" : `Task ${task}`),
  day: (unix) => `day-${unix}`,
  now: NOW,
};

function tuneUp(overrides: Partial<AITuneUp>): AITuneUp {
  return {
    id: "aitu_1",
    kind: "LeaveShadow",
    agent: { id: "agd_1", name: "Dispatch desk", icon: "truck", accent: "indigo" },
    provider: null,
    otherProvider: null,
    toolName: null,
    task: null,
    status: "Open",
    dismissedUntil: null,
    computedAt: NOW,
    version: 1,
    ...overrides,
    evidence: {
      fromTier: null,
      toTier: null,
      streak: 0,
      approvals: 0,
      approvalsPerWeek: 0,
      rejections: 0,
      calls: 0,
      failed: 0,
      rescued: 0,
      recorded: 0,
      matchRate: 0,
      wouldFail: 0,
      tasks: [],
      model: "",
      lastRunAt: null,
      idleSince: 0,
      tools: 0,
      ...overrides.evidence,
    },
  };
}

const titleText = (parts: { text: string }[]) => parts.map((part) => part.text).join("");
const strongText = (parts: { text: string; strong?: boolean }[]) =>
  parts.filter((part) => part.strong).map((part) => part.text);

describe("tuneUpCopy", () => {
  it("says a raised tool runs on its own only when it would reach automatic", () => {
    const automatic = tuneUpCopy(
      tuneUp({
        kind: "RaiseToolTier",
        toolName: "assign_move",
        evidence: { ...tuneUp({}).evidence, toTier: "AutoExecute", streak: 12, approvalsPerWeek: 6.4 },
      }),
      t,
      labels,
    );
    expect(titleText(automatic.title)).toBe("Let Assign move run on its own for Dispatch desk");
    expect(strongText(automatic.title)).toEqual(["Assign move"]);
    expect(automatic.evidence).toBe("Approved unchanged 12 times in a row · never rejected");
    expect(automatic.gain).toBe("About 6 approvals a week");
    expect(automatic.tone).toBe("ok");

    const askFirst = tuneUpCopy(
      tuneUp({
        kind: "RaiseToolTier",
        toolName: "assign_move",
        evidence: { ...tuneUp({}).evidence, toTier: "ActWithApproval", rejections: 3, approvalsPerWeek: 0.4 },
      }),
      t,
      labels,
    );
    expect(titleText(askFirst.title)).toBe("Move Assign move to Ask first for Dispatch desk");
    expect(askFirst.evidence).toContain("3 rejections before that");
    expect(askFirst.gain).toBe("Fewer approvals to make");
  });

  it("names both providers and the tasks a reorder helps", () => {
    const copy = tuneUpCopy(
      tuneUp({
        kind: "ReorderProviders",
        agent: null,
        provider: { id: "aip_o", name: "Local Ollama", kind: "Ollama" },
        otherProvider: { id: "aip_v", name: "Workstation vLLM", kind: "OpenAIChat" },
        evidence: { ...tuneUp({}).evidence, failed: 24, rescued: 22, tasks: ["General", "DailyBriefing"] },
      }),
      t,
      labels,
    );
    expect(titleText(copy.title)).toBe("Put Local Ollama ahead of Workstation vLLM");
    expect(strongText(copy.title)).toEqual(["Local Ollama"]);
    expect(copy.evidence).toBe(
      "Workstation vLLM failed 24 times in 30 days; Local Ollama answered 22 of them",
    );
    expect(copy.gain).toBe("Fewer retries on 2 tasks");

    const single = tuneUpCopy(
      tuneUp({
        kind: "ReorderProviders",
        agent: null,
        provider: { id: "aip_o", name: "Local Ollama", kind: "Ollama" },
        otherProvider: { id: "aip_v", name: "Workstation vLLM", kind: "OpenAIChat" },
        evidence: { ...tuneUp({}).evidence, tasks: ["General"] },
      }),
      t,
      labels,
    );
    expect(single.gain).toBe("Fewer retries on Task General");
  });

  it("gives the match rate as a whole percent and says when a shadow write would fail", () => {
    const clean = tuneUpCopy(
      tuneUp({ evidence: { ...tuneUp({}).evidence, recorded: 31, matchRate: 0.8696 } }),
      t,
      labels,
    );
    expect(titleText(clean.title)).toBe("Take Dispatch desk out of shadow");
    expect(clean.evidence).toBe(
      "31 recorded proposals · 87% match what people did · none would have failed",
    );
    expect(clean.tone).toBe("brand");

    const failing = tuneUpCopy(
      tuneUp({ evidence: { ...tuneUp({}).evidence, recorded: 40, matchRate: 0.9, wouldFail: 2 } }),
      t,
      labels,
    );
    expect(failing.evidence).toContain("2 would have failed");
  });

  it("names the task and the model that would take it", () => {
    const copy = tuneUpCopy(
      tuneUp({
        kind: "AssignTask",
        agent: null,
        task: "Embedding",
        provider: { id: "aip_o", name: "Local Ollama", kind: "Ollama" },
        evidence: { ...tuneUp({}).evidence, model: "nomic-embed-text" },
      }),
      t,
      labels,
    );
    expect(titleText(copy.title)).toBe("Give Embedding to Local Ollama");
    expect(strongText(copy.title)).toEqual(["Embedding"]);
    expect(copy.evidence).toBe(
      "Nothing handles it, so search matches words only. Local Ollama already serves nomic-embed-text",
    );
    expect(copy.gain).toBe("Search by meaning");
    expect(copy.applied).toBe("Embedding now goes to Local Ollama");

    const other = tuneUpCopy(
      tuneUp({
        kind: "AssignTask",
        agent: null,
        task: "DocumentExtraction",
        provider: { id: "aip_o", name: "Local Ollama", kind: "Ollama" },
      }),
      t,
      labels,
    );
    expect(other.evidence).toBe("Nothing handles it now. Local Ollama can take it");
    expect(other.gain).toBe("Task DocumentExtraction starts working");
  });

  it("counts idle days from the last run, and says never when there was none", () => {
    const idle = tuneUpCopy(
      tuneUp({
        kind: "TurnOffIdleAgent",
        evidence: { ...tuneUp({}).evidence, lastRunAt: NOW - 15 * DAY, idleSince: NOW - 15 * DAY, tools: 55 },
      }),
      t,
      labels,
    );
    expect(titleText(idle.title)).toBe("Dispatch desk hasn't run in 15 days");
    expect(idle.evidence).toBe(
      `Nobody has asked it anything since day-${NOW - 15 * DAY} · it holds 55 tools`,
    );
    expect(idle.tone).toBe("neutral");

    const never = tuneUpCopy(
      tuneUp({
        kind: "TurnOffIdleAgent",
        evidence: { ...tuneUp({}).evidence, idleSince: NOW - 90 * DAY, tools: 1 },
      }),
      t,
      labels,
    );
    expect(titleText(never.title)).toBe("Dispatch desk has never run");
    expect(never.evidence).toContain("it holds 1 tool");
  });
});
