import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  composerDrafts,
  draftKey,
  DRAFT_STORAGE_KEY,
  DRAFT_WRITE_MS,
  MAX_DRAFT_LENGTH,
  MAX_DRAFTS,
  parseDrafts,
  recordSpace,
  serializeDrafts,
} from "../composer-draft-store";

const shipment = { type: "shipment", id: "shp_1", label: "PRO-1001" };

function stored(): Record<string, { text: string; mentions: unknown[] }> {
  const raw = localStorage.getItem(DRAFT_STORAGE_KEY);
  return raw === null ? {} : JSON.parse(raw).drafts;
}

beforeEach(() => {
  vi.useFakeTimers();
  composerDrafts.clear();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("composer drafts", () => {
  it("keeps a draft for each conversation and each space, apart", () => {
    composerDrafts.setText(draftKey.thread("thr_1"), "where is");
    composerDrafts.setText(draftKey.space("desk-today"), "who is free");
    composerDrafts.setText(draftKey.space(recordSpace("shipment", "shp_1")), "why is it late");

    expect(composerDrafts.text(draftKey.thread("thr_1"))).toBe("where is");
    expect(composerDrafts.text(draftKey.space("desk-today"))).toBe("who is free");
    expect(composerDrafts.text(draftKey.space("record:shipment:shp_1"))).toBe("why is it late");
    expect(composerDrafts.text(draftKey.thread("thr_2"))).toBe("");
  });

  it("is in memory at once and on disk only after typing pauses", () => {
    const key = draftKey.thread("thr_1");
    composerDrafts.setText(key, "w");
    composerDrafts.setText(key, "wh");

    expect(composerDrafts.text(key)).toBe("wh");
    expect(localStorage.getItem(DRAFT_STORAGE_KEY)).toBeNull();

    vi.advanceTimersByTime(DRAFT_WRITE_MS);

    expect(stored()[key]?.text).toBe("wh");
  });

  it("comes back from disk with the records it names", () => {
    const key = draftKey.thread("thr_1");
    composerDrafts.setText(key, "Where is @PRO-1001");
    composerDrafts.setMentions(key, [shipment]);
    composerDrafts.flush();

    composerDrafts.reload();

    expect(composerDrafts.text(key)).toBe("Where is @PRO-1001");
    expect(composerDrafts.mentions(key)).toEqual([shipment]);
  });

  it("forgets an emptied draft, and its records with it", () => {
    const key = draftKey.thread("thr_1");
    composerDrafts.setText(key, "Where is @PRO-1001");
    composerDrafts.setMentions(key, [shipment]);
    composerDrafts.setText(key, "");
    composerDrafts.flush();

    expect(composerDrafts.mentions(key)).toEqual([]);
    expect(localStorage.getItem(DRAFT_STORAGE_KEY)).toBeNull();
  });

  it("keeps no records for a draft with no words", () => {
    const key = draftKey.thread("thr_1");
    composerDrafts.setMentions(key, [shipment]);

    expect(composerDrafts.mentions(key)).toEqual([]);
  });

  it("tells only the listeners of the draft that changed", () => {
    const mine = vi.fn();
    const other = vi.fn();
    const stop = composerDrafts.subscribe(draftKey.thread("thr_1"), mine);
    composerDrafts.subscribe(draftKey.thread("thr_2"), other);

    composerDrafts.setText(draftKey.thread("thr_1"), "a");
    composerDrafts.setText(draftKey.thread("thr_1"), "a");
    stop();
    composerDrafts.setText(draftKey.thread("thr_1"), "ab");

    expect(mine).toHaveBeenCalledTimes(1);
    expect(other).not.toHaveBeenCalled();
  });

  it("drops the draft left alone longest once there are too many", () => {
    for (let index = 0; index <= MAX_DRAFTS; index += 1) {
      vi.setSystemTime(1_000 + index);
      composerDrafts.setText(draftKey.thread(`thr_${index}`), `draft ${index}`);
    }

    expect(composerDrafts.text(draftKey.thread("thr_0"))).toBe("");
    expect(composerDrafts.text(draftKey.thread("thr_1"))).toBe("draft 1");
    expect(composerDrafts.text(draftKey.thread(`thr_${MAX_DRAFTS}`))).toBe(`draft ${MAX_DRAFTS}`);
  });

  it("clears every draft from memory and disk", () => {
    composerDrafts.setText(draftKey.thread("thr_1"), "where is");
    composerDrafts.flush();

    composerDrafts.clear();

    expect(composerDrafts.text(draftKey.thread("thr_1"))).toBe("");
    expect(localStorage.getItem(DRAFT_STORAGE_KEY)).toBeNull();
  });
});

describe("what is written to disk", () => {
  it("leaves out a draft too long to keep", () => {
    const drafts = new Map([
      ["thread:short", { text: "short", mentions: [], at: 1 }],
      ["thread:long", { text: "x".repeat(MAX_DRAFT_LENGTH + 1), mentions: [], at: 2 }],
    ]);

    expect([...parseDrafts(serializeDrafts(drafts)).keys()]).toEqual(["thread:short"]);
  });

  it("reads nothing from a blob that is not its own, and drops a malformed draft or record", () => {
    expect(parseDrafts(null).size).toBe(0);
    expect(parseDrafts("not json").size).toBe(0);
    expect(parseDrafts(JSON.stringify({ v: 99, drafts: { a: { text: "x" } } })).size).toBe(0);

    const read = parseDrafts(
      JSON.stringify({
        v: 1,
        drafts: {
          good: { text: "Where is @PRO-1001", mentions: [shipment, { id: 4 }, "x"], at: 5 },
          empty: { text: "", mentions: [], at: 5 },
          wrong: "a string",
        },
      }),
    );

    expect([...read.keys()]).toEqual(["good"]);
    expect(read.get("good")?.mentions).toEqual([shipment]);
  });
});
