import { describe, expect, it } from "vitest";
import { rememberDraft } from "../assistant-store";

describe("rememberDraft", () => {
  it("keeps a draft per conversation and forgets an emptied one", () => {
    let drafts = rememberDraft({}, "thr_1", "where is");
    drafts = rememberDraft(drafts, "thr_2", "who is free");
    expect(drafts).toEqual({ thr_1: "where is", thr_2: "who is free" });

    expect(rememberDraft(drafts, "thr_1", "")).toEqual({ thr_2: "who is free" });
  });

  it("keeps only the most recently written conversations", () => {
    let drafts: Record<string, string> = {};
    for (let index = 0; index < 25; index += 1) {
      drafts = rememberDraft(drafts, `thr_${index}`, `draft ${index}`);
    }

    expect(Object.keys(drafts)).toHaveLength(20);
    expect(drafts.thr_0).toBeUndefined();
    expect(drafts.thr_24).toBe("draft 24");
  });

  it("moves a rewritten draft to the newest position", () => {
    let drafts = rememberDraft({}, "thr_a", "a");
    drafts = rememberDraft(drafts, "thr_b", "b");
    drafts = rememberDraft(drafts, "thr_a", "a2");

    expect(Object.keys(drafts)).toEqual(["thr_b", "thr_a"]);
  });
});
