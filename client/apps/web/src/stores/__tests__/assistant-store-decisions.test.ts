import { describe, expect, it } from "vitest";
import {
  MAX_DEFERRED_DECISIONS,
  deferKeys,
  mergePersisted,
  resumeKeys,
  useAssistantStore,
} from "../assistant-store";

const current = () => useAssistantStore.getInitialState();

describe("decisions put off for later", () => {
  it("remembers each decision once, newest last, and forgets the oldest past the cap", () => {
    let deferred = deferKeys([], ["proposal:a", "plan:b"]);
    deferred = deferKeys(deferred, ["proposal:a"]);
    expect(deferred).toEqual(["plan:b", "proposal:a"]);

    const many = Array.from(
      { length: MAX_DEFERRED_DECISIONS + 5 },
      (_, index) => `proposal:${index}`,
    );
    const capped = deferKeys([], many);
    expect(capped).toHaveLength(MAX_DEFERRED_DECISIONS);
    expect(capped[0]).toBe("proposal:5");
  });

  it("reopens exactly the decisions named", () => {
    expect(
      resumeKeys(["proposal:a", "plan:b", "proposal:c"], ["proposal:a", "proposal:c"]),
    ).toEqual(["plan:b"]);
  });

  // The list is read back from the browser; anything that is not a key is
  // dropped rather than trusted, and a focus never survives a reload.
  it("restores only keys, and never a focus", () => {
    const restored = mergePersisted(
      { deferredDecisions: ["proposal:a", 7, null, "plan:b"], decisionFocus: { athr_1: {} } },
      current(),
    );

    expect(restored.deferredDecisions).toEqual(["proposal:a", "plan:b"]);
    expect(restored.decisionFocus).toEqual({});
    expect(
      mergePersisted({ deferredDecisions: "proposal:a" }, current()).deferredDecisions,
    ).toEqual([]);
  });

  it("opens a focused decision even when it was put off", () => {
    useAssistantStore.setState({
      deferredDecisions: ["proposal:a", "proposal:b"],
      decisionFocus: {},
    });

    useAssistantStore
      .getState()
      .focusDecision("athr_1", { proposalIds: ["a"], planId: "" }, ["proposal:a"]);

    expect(useAssistantStore.getState().deferredDecisions).toEqual(["proposal:b"]);
    expect(useAssistantStore.getState().decisionFocus).toEqual({
      athr_1: { proposalIds: ["a"], planId: "" },
    });

    useAssistantStore.getState().clearDecisionFocus("athr_1");
    expect(useAssistantStore.getState().decisionFocus).toEqual({});
  });
});
