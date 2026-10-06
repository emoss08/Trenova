import {
  compactionReducer,
  IDLE,
  kfmt,
  meterView,
  type CompactionState,
} from "@/components/assistant/compaction";
import type { AssistantCompactionEvent, ContextUsage } from "@/types/assistant";
import { describe, expect, it } from "vitest";

function usage(overrides: Partial<ContextUsage> = {}): ContextUsage {
  return {
    instructions: 6200,
    messages: 41_800,
    toolResults: 79_500,
    files: 12_300,
    compactable: 110_000,
    window: 200_000,
    model: "claude-sonnet-4-5",
    measuredAt: 0,
    ...overrides,
  };
}

function started(overrides: Partial<AssistantCompactionEvent> = {}): AssistantCompactionEvent {
  return {
    turnId: "atrn_1",
    threadId: "athr_1",
    auto: false,
    before: 139_800,
    after: 31_000,
    autoCompactOff: false,
    ...overrides,
  };
}

describe("meterView", () => {
  it("adds up the parts against the model's window", () => {
    const view = meterView(usage());

    expect(view.used).toBe(139_800);
    expect(view.window).toBe(200_000);
    expect(view.pct).toBe(70);
    expect(view.tone).toBe("");
    expect(view.showPct).toBe(false);
    expect(view.parts.map((part) => [part.key, part.tokens])).toEqual([
      ["conv", 41_800],
      ["tools", 79_500],
      ["files", 12_300],
      ["sys", 6200],
    ]);
  });

  it("turns amber from three quarters full and red from nine tenths, with the figure shown", () => {
    const warm = meterView(usage({ toolResults: 100_000 }));
    expect(warm.tone).toBe("warm");
    expect(warm.showPct).toBe(true);

    const hot = meterView(usage({ toolResults: 125_000 }));
    expect(hot.tone).toBe("hot");
    expect(hot.pct).toBe(93);
  });

  it("offers to compact only what frees enough, less the summary that replaces it", () => {
    const view = meterView(usage());
    expect(view.frees).toBe(110_000 - 1200);
    expect(view.canCompact).toBe(true);

    const small = meterView(usage({ compactable: 5000 }));
    expect(small.frees).toBe(4000);
    expect(small.canCompact).toBe(false);
  });

  it("shows an empty ring for a conversation not yet measured", () => {
    const view = meterView(null);

    expect(view.used).toBe(0);
    expect(view.window).toBe(200_000);
    expect(view.canCompact).toBe(false);
  });
});

describe("kfmt", () => {
  it("writes tokens the way the meter does", () => {
    expect(kfmt(950)).toBe("950");
    expect(kfmt(6200)).toBe("6.2k");
    expect(kfmt(41_000)).toBe("41k");
    expect(kfmt(139_800)).toBe("140k");
    expect(kfmt(200_000)).toBe("200k");
  });
});

describe("compactionReducer", () => {
  it("locks the composer from the request, then takes the turn and figures the server names", () => {
    const requested = compactionReducer(IDLE, { type: "request", before: 139_800, after: 30_000 });
    expect(requested).toEqual({
      phase: "compacting",
      turnId: null,
      auto: false,
      before: 139_800,
      after: 30_000,
    });

    const named = compactionReducer(requested, { type: "started", event: started() });
    expect(named).toMatchObject({ phase: "compacting", turnId: "atrn_1", after: 31_000 });
  });

  it("keeps what it shows when a stream names the turn without figures", () => {
    const requested = compactionReducer(IDLE, { type: "request", before: 139_800, after: 30_000 });
    const named = compactionReducer(requested, {
      type: "started",
      event: started({ before: 0, after: 0 }),
    });

    expect(named).toMatchObject({ before: 139_800, after: 30_000 });
  });

  it("follows a compaction a turn set off on its own", () => {
    const state = compactionReducer(IDLE, { type: "started", event: started({ auto: true }) });

    expect(state).toMatchObject({ phase: "compacting", auto: true, turnId: "atrn_1" });
  });

  it("frees the composer when the compaction finishes or is cancelled on the server", () => {
    const compacting = compactionReducer(IDLE, { type: "started", event: started() });

    expect(compactionReducer(compacting, { type: "finished", turnId: "atrn_1" })).toEqual(IDLE);
    expect(compactionReducer(compacting, { type: "cancelled", turnId: "atrn_1" })).toEqual(IDLE);
  });

  it("ignores the ending of another turn than the one shown", () => {
    const compacting = compactionReducer(IDLE, { type: "started", event: started() });

    expect(compactionReducer(compacting, { type: "finished", turnId: "atrn_2" })).toBe(compacting);
    expect(
      compactionReducer(compacting, { type: "started", event: started({ turnId: "atrn_2" }) }),
    ).toBe(compacting);
  });

  it("frees the composer at once on Cancel and ignores the cancelled turn's late events", () => {
    const compacting = compactionReducer(IDLE, { type: "started", event: started() });
    const cancelled = compactionReducer(compacting, { type: "cancel" });

    expect(cancelled).toEqual({ phase: "idle", ignored: "atrn_1", error: null });
    expect(compactionReducer(cancelled, { type: "started", event: started() })).toBe(cancelled);
    expect(
      compactionReducer(cancelled, {
        type: "rejoined",
        turnId: "atrn_1",
        before: 1,
        after: 1,
      }),
    ).toBe(cancelled);
  });

  it("says why a compaction failed and frees the composer", () => {
    const compacting = compactionReducer(IDLE, { type: "started", event: started() });
    const failed = compactionReducer(compacting, {
      type: "failed",
      message: "There is nothing to compact yet.",
    });

    expect(failed).toEqual({
      phase: "idle",
      ignored: null,
      error: "There is nothing to compact yet.",
    });
  });

  it("picks up a compaction under way when the conversation is reopened", () => {
    const rejoined = compactionReducer(IDLE, {
      type: "rejoined",
      turnId: "atrn_9",
      before: 172_000,
      after: 31_000,
    });

    expect(rejoined).toMatchObject({ phase: "compacting", turnId: "atrn_9" });
  });

  it("does not start a second compaction over one under way", () => {
    const compacting: CompactionState = compactionReducer(IDLE, {
      type: "started",
      event: started(),
    });

    expect(compactionReducer(compacting, { type: "request", before: 1, after: 1 })).toBe(
      compacting,
    );
  });
});
