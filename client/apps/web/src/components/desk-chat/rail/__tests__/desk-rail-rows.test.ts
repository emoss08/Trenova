import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  deskRailRowKey,
  deskRailRows,
  type DeskRailRow,
  type DeskRailShelf,
} from "../desk-rail-rows";
import type { DeskShelfKey } from "../desk-threads";

function thread(id: string): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agtd_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: id,
    status: "Active",
    lastMessageAt: 100,
    version: 0,
    createdAt: 50,
    updatedAt: 100,
  };
}

const keys = (rows: readonly DeskRailRow[]) => rows.map(deskRailRowKey);

function shelf(
  key: DeskShelfKey,
  threads: AssistantThread[],
  overrides: Partial<Omit<DeskRailShelf, "key" | "threads">> = {},
): DeskRailShelf {
  return {
    key,
    threads,
    count: threads.length,
    paging: { loading: false, hasMore: false, failed: false },
    ...overrides,
  };
}

/**
 * The rail draws its shelves as one flat list so only the rows in view are
 * mounted. Each shelf is read on its own, so the place the rest of a shelf
 * arrives sits at its own end.
 */
describe("deskRailRows", () => {
  const read: DeskRailShelf[] = [
    shelf("pinned", [thread("p")]),
    shelf("today", [thread("a"), thread("b")]),
  ];

  it("lays out each shelf's heading above its conversations", () => {
    expect(keys(deskRailRows(read))).toEqual(["s:pinned", "c:p", "s:today", "c:a", "c:b"]);
  });

  it("ends a shelf with the place the rest of it arrives while more remains", () => {
    const rows = deskRailRows([
      shelf("today", [thread("a")], {
        paging: { loading: false, hasMore: true, failed: false },
      }),
      shelf("week", [thread("w")]),
    ]);

    expect(keys(rows)).toEqual(["s:today", "c:a", "more:today", "s:week", "c:w"]);
    expect(rows[2]).toEqual({ kind: "more", shelf: "today", loading: false, failed: false });
  });

  it("shows a shelf's first page arriving under its counted heading", () => {
    const rows = deskRailRows([
      shelf("month", [], { count: 7, paging: { loading: true, hasMore: false, failed: false } }),
    ]);

    expect(rows).toEqual([
      { kind: "shelf", key: "month" },
      { kind: "more", shelf: "month", loading: true, failed: false },
    ]);
  });

  it("keeps the place of a page that could not be read, so it can be asked for again", () => {
    const rows = deskRailRows([
      shelf("older", [], { count: 3, paging: { loading: false, hasMore: false, failed: true } }),
    ]);

    expect(rows.at(-1)).toEqual({ kind: "more", shelf: "older", loading: false, failed: true });
  });

  it("leaves out a shelf counted empty with nothing read", () => {
    expect(keys(deskRailRows([shelf("week", [], { count: 0 })]))).toEqual([]);
  });

  it("draws what was read even when the count says otherwise", () => {
    expect(keys(deskRailRows([shelf("week", [thread("w")], { count: 0 })]))).toEqual([
      "s:week",
      "c:w",
    ]);
  });

  it("waits for a shelf's count before drawing it with nothing read", () => {
    const rows = deskRailRows([
      shelf("today", [], {
        count: "counting",
        paging: { loading: true, hasMore: false, failed: false },
      }),
    ]);

    expect(rows).toEqual([]);
  });

  it("keeps the heading of a shelf that could not be counted", () => {
    expect(keys(deskRailRows([shelf("older", [], { count: "unknown" })]))).toEqual(["s:older"]);
  });

  it("is empty when there is nothing on any shelf", () => {
    expect(deskRailRows([])).toEqual([]);
  });
});
