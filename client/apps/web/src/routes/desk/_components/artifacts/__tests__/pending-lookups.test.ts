import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { NO_PENDING_LOOKUPS, pendingLookupIds, withoutPendingLookups } from "../pending-lookups";

function artifact(id: string, extra: Partial<AssistantArtifact> = {}): AssistantArtifact {
  return {
    id,
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: id,
    payload: {},
    sourceToolCallId: "",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: id,
    turn: "",
    createdAt: 1,
    updatedAt: 1,
    ...extra,
  };
}

const counts = (all: number, families: Record<string, number>, pinned = 0) => ({
  all,
  pinned,
  families,
});

describe("pendingLookupIds", () => {
  it("collects only the lookups a turn saved", () => {
    const pending = pendingLookupIds(
      [
        { id: "art_1", kind: "table_view" },
        { id: "art_2", kind: "document" },
        { id: "art_3", kind: "entity_card" },
      ],
      NO_PENDING_LOOKUPS,
    );
    expect([...pending]).toEqual(["art_1", "art_3"]);
  });

  it("keeps the same set when nothing new arrived", () => {
    const pending = new Set(["art_1"]);
    expect(pendingLookupIds([{ id: "art_1", kind: "table_view" }], pending)).toBe(pending);
    expect(pendingLookupIds([{ id: "art_2", kind: "plan" }], pending)).toBe(pending);
  });

  it("adds to what is already waiting, so a lookup the turn folded away stays hidden", () => {
    const pending = pendingLookupIds([{ id: "art_2", kind: "table_view" }], new Set(["art_1"]));
    expect([...pending].sort()).toEqual(["art_1", "art_2"]);
  });
});

describe("withoutPendingLookups", () => {
  it("returns the page untouched when nothing is waiting", () => {
    const results = [artifact("art_1")];
    const page = counts(1, { table: 1 });
    const out = withoutPendingLookups(results, page, NO_PENDING_LOOKUPS);
    expect(out.results).toBe(results);
    expect(out.counts).toBe(page);
    expect(out.hidden).toBe(0);
  });

  it("hides the running turn's lookups and takes them off the counts", () => {
    const results = [
      artifact("art_old"),
      artifact("art_new", { kind: "entity_card" }),
      artifact("art_doc", { kind: "document" }),
    ];
    const out = withoutPendingLookups(
      results,
      counts(3, { table: 1, record: 1, doc: 1 }),
      new Set(["art_new"]),
    );
    expect(out.results.map((item) => item.id)).toEqual(["art_old", "art_doc"]);
    expect(out.counts).toEqual(counts(2, { table: 1, record: 0, doc: 1 }));
    expect(out.hidden).toBe(1);
  });

  it("keeps lookups from earlier turns", () => {
    const results = [artifact("art_earlier"), artifact("art_now")];
    const out = withoutPendingLookups(results, counts(2, { table: 2 }), new Set(["art_now"]));
    expect(out.results.map((item) => item.id)).toEqual(["art_earlier"]);
    expect(out.counts?.all).toBe(1);
  });

  it("hides only the new version of a lineage an earlier turn started", () => {
    const results = [
      artifact("art_v1"),
      artifact("art_v2", { lineageId: "art_v1", lineageSeq: 2 }),
    ];
    const out = withoutPendingLookups(results, counts(1, { table: 1 }), new Set(["art_v2"]));
    expect(out.results.map((item) => item.id)).toEqual(["art_v1"]);
    expect(out.counts).toEqual(counts(1, { table: 1 }));
    expect(out.hidden).toBe(0);
  });

  it("never hides what is not a lookup, whatever the set says", () => {
    const results = [artifact("art_doc", { kind: "document" })];
    const out = withoutPendingLookups(results, counts(1, { doc: 1 }), new Set(["art_doc"]));
    expect(out.results).toBe(results);
  });

  it("does not count below zero", () => {
    const out = withoutPendingLookups(
      [artifact("art_1", { pinned: true })],
      counts(0, {}, 0),
      new Set(["art_1"]),
    );
    expect(out.counts).toEqual(counts(0, {}, 0));
  });
});
