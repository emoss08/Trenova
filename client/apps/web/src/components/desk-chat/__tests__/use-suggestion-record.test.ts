import { beforeEach, describe, expect, it, vi } from "vitest";
import { createSuggestionRecord } from "../use-suggestion-record";

const calls = vi.hoisted(() => ({
  recorded: [] as { suggestedAgentId: string; outcome: string }[],
  settled: [] as { id: string; outcome: string }[],
  failRecord: false,
}));

vi.mock("@/lib/graphql/agent-suggestion", () => ({
  recordAgentSuggestion: (input: { suggestedAgentId: string; outcome: string }) => {
    calls.recorded.push({ suggestedAgentId: input.suggestedAgentId, outcome: input.outcome });
    if (calls.failRecord) {
      return Promise.reject(new Error("offline"));
    }

    return Promise.resolve(`agsg_${calls.recorded.length}`);
  },
  settleAgentSuggestion: (id: string, outcome: string) => {
    calls.settled.push({ id, outcome });

    return Promise.resolve();
  },
}));

const flush = () => new Promise((settle) => setTimeout(settle, 0));

beforeEach(() => {
  calls.recorded = [];
  calls.settled = [];
  calls.failRecord = false;
});

describe("createSuggestionRecord", () => {
  it("records an agent once while it stays named", () => {
    const record = createSuggestionRecord("Home");

    record.shown("agdef_billing", "agdef_dispatch");
    record.shown("agdef_billing", "agdef_dispatch");

    expect(calls.recorded).toEqual([{ suggestedAgentId: "agdef_billing", outcome: "Shown" }]);
  });

  it("passes over the agent named before when another is named in its place", async () => {
    const record = createSuggestionRecord("Conversation");

    record.shown("agdef_billing", "agdef_dispatch");
    record.shown("agdef_service", "agdef_dispatch");
    await flush();

    expect(calls.settled).toEqual([{ id: "agsg_1", outcome: "PassedOver" }]);
    expect(calls.recorded.map((call) => call.suggestedAgentId)).toEqual([
      "agdef_billing",
      "agdef_service",
    ]);
  });

  it("settles a suggestion taken for a send even after the box was cleared", async () => {
    const record = createSuggestionRecord("Home");

    record.shown("agdef_billing", "agdef_dispatch");
    const settle = record.take();
    record.forget();
    settle("SentThrough");
    await flush();

    expect(calls.settled).toEqual([{ id: "agsg_1", outcome: "SentThrough" }]);
  });

  it("settles nothing once the suggestion was settled or forgotten", async () => {
    const record = createSuggestionRecord("Home");

    record.settle("Kept");
    record.shown("agdef_billing", "agdef_dispatch");
    record.settle("Switched");
    record.settle("Kept");
    record.shown("agdef_service", "agdef_dispatch");
    record.forget();
    record.settle("Kept");
    await flush();

    expect(calls.settled).toEqual([{ id: "agsg_1", outcome: "Switched" }]);
  });

  it("does not settle a suggestion that could not be recorded", async () => {
    calls.failRecord = true;
    const record = createSuggestionRecord("Home");

    record.shown("agdef_billing", "agdef_dispatch");
    record.settle("Switched");
    await flush();

    expect(calls.settled).toEqual([]);
  });
});
