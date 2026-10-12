import { beforeEach, describe, expect, it } from "vitest";
import { rememberActiveArtifact, useDeskStore, withLiveHandoff } from "../desk-store";

/**
 * The rail is the Desk's table of contents, and whether a person keeps it
 * open is a habit rather than a choice made per visit, so it is remembered
 * beside the workspace pane and comes back the way it was left.
 */
describe("the Desk rail", () => {
  beforeEach(() => {
    useDeskStore.setState({ rail: "open", pane: "open" });
  });

  it("starts open", () => {
    expect(useDeskStore.getState().rail).toBe("open");
  });

  it("folds and unfolds on a toggle", () => {
    useDeskStore.getState().toggleRail();
    expect(useDeskStore.getState().rail).toBe("closed");

    useDeskStore.getState().toggleRail();
    expect(useDeskStore.getState().rail).toBe("open");
  });

  it("can be set outright", () => {
    useDeskStore.getState().setRail("closed");
    expect(useDeskStore.getState().rail).toBe("closed");
  });

  it("leaves the workspace pane alone", () => {
    useDeskStore.getState().setPane("closed");
    useDeskStore.getState().toggleRail();

    expect(useDeskStore.getState()).toMatchObject({ rail: "closed", pane: "closed" });
  });

  it("is remembered between visits, beside the pane and the open artifacts", () => {
    useDeskStore.getState().setRail("closed");
    useDeskStore.getState().setActiveArtifact("thr_1", "art_1");

    const partialize = useDeskStore.persist.getOptions().partialize;
    expect(partialize).toBeDefined();
    expect(partialize?.(useDeskStore.getState())).toMatchObject({
      rail: "closed",
      pane: "open",
      activeArtifactByThread: { thr_1: "art_1" },
      sharePage: true,
    });
  });
});

describe("rememberActiveArtifact", () => {
  it("keeps an artifact per conversation and forgets a cleared one", () => {
    let remembered = rememberActiveArtifact({}, "thr_1", "art_1");
    remembered = rememberActiveArtifact(remembered, "thr_2", "art_2");
    expect(remembered).toEqual({ thr_1: "art_1", thr_2: "art_2" });

    expect(rememberActiveArtifact(remembered, "thr_1", null)).toEqual({ thr_2: "art_2" });
  });

  it("keeps only the most recently visited conversations", () => {
    let remembered: Record<string, string> = {};
    for (let index = 0; index < 45; index += 1) {
      remembered = rememberActiveArtifact(remembered, `thr_${index}`, `art_${index}`);
    }

    expect(Object.keys(remembered)).toHaveLength(40);
    expect(remembered.thr_0).toBeUndefined();
    expect(remembered.thr_44).toBe("art_44");
  });
});

/**
 * What the reply being written says about a hand-off, per conversation, for
 * the top bar's menu. A reader is woken by a new record only, so an unchanged
 * report must leave the record as it was.
 */
describe("the live hand-off", () => {
  const billing: readonly string[] = ["agdef_billing"];

  beforeEach(() => {
    useDeskStore.setState({ liveHandoff: {} });
  });

  it("keeps what each conversation's reply said, and forgets it once the reply is gone", () => {
    const { setLiveHandoff } = useDeskStore.getState();
    setLiveHandoff("thr_1", { asked: true, agentIds: billing });
    setLiveHandoff("thr_2", { asked: false, agentIds: [] });

    expect(useDeskStore.getState().liveHandoff).toEqual({
      thr_1: { asked: true, agentIds: ["agdef_billing"] },
      thr_2: { asked: false, agentIds: [] },
    });

    setLiveHandoff("thr_1", null);
    expect(useDeskStore.getState().liveHandoff).toEqual({
      thr_2: { asked: false, agentIds: [] },
    });
  });

  it("leaves the record alone when nothing changed", () => {
    const current = withLiveHandoff({}, "thr_1", { asked: true, agentIds: billing });

    expect(withLiveHandoff(current, "thr_1", { asked: true, agentIds: billing })).toBe(current);
    expect(withLiveHandoff(current, "thr_9", null)).toBe(current);
    expect(withLiveHandoff(current, "thr_1", { asked: false, agentIds: billing })).not.toBe(
      current,
    );
  });

  it("is not remembered between visits", () => {
    useDeskStore.getState().setLiveHandoff("thr_1", { asked: true, agentIds: billing });

    const partialize = useDeskStore.persist.getOptions().partialize;
    expect(partialize?.(useDeskStore.getState())).not.toHaveProperty("liveHandoff");
  });
});
