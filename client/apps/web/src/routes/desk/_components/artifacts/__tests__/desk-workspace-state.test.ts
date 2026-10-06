import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { groupLineages } from "../desk-lineage";
import {
  commandJ,
  olderNote,
  pushRecent,
  stackLineages,
  stepLineage,
} from "../desk-workspace-state";

function artifact(id: string, createdAt: number, extra: Partial<AssistantArtifact> = {}) {
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
    createdAt,
    updatedAt: createdAt,
    ...extra,
  } satisfies AssistantArtifact;
}

describe("commandJ", () => {
  it("opens the pane on every artifact from anywhere but there", () => {
    expect(commandJ({ open: false, browsing: false })).toEqual({ open: true, browsing: true });
    expect(commandJ({ open: true, browsing: false })).toEqual({ open: true, browsing: true });
  });

  it("closes the pane when every artifact is already showing", () => {
    expect(commandJ({ open: true, browsing: true })).toEqual({ open: false, browsing: false });
  });
});

describe("stackLineages", () => {
  const lineages = groupLineages([
    artifact("a", 10),
    artifact("b", 20),
    artifact("c", 30),
    artifact("d", 40),
    artifact("e", 50),
    artifact("f", 5, { pinned: true }),
    ...Array.from({ length: 10 }, (_, i) => artifact(`old${i}`, i)),
  ]);
  const byId = (id: string) => lineages.find((lineage) => lineage.id === id)!;

  it("holds the open one, the three newest, the pinned and the recent, once each", () => {
    const ids = stackLineages(lineages, byId("a"), ["old3", "e"]).map((lineage) => lineage.id);
    expect(ids).toEqual(["a", "e", "d", "c", "f", "old3"]);
  });

  it("stops at eight cards", () => {
    const recent = ["old1", "old2", "old3", "old4", "old5", "old6"];
    expect(stackLineages(lineages, byId("a"), recent)).toHaveLength(8);
  });

  it("keeps an open lineage the page did not hold", () => {
    const [outside] = groupLineages([artifact("far", 1)]);
    expect(stackLineages(lineages, outside, [])[0].id).toBe("far");
  });
});

describe("pushRecent", () => {
  it("moves the opened one to the front and keeps six", () => {
    let recent: string[] = [];
    for (const id of ["a", "b", "c", "d", "e", "f", "g", "b"]) {
      recent = pushRecent(recent, id);
    }
    expect(recent).toEqual(["b", "g", "f", "e", "d", "c"]);
  });
});

describe("olderNote", () => {
  const now = new Date(2026, 9, 3, 15, 0);

  it("says nothing of today's artifacts", () => {
    expect(olderNote(new Date(2026, 9, 3, 9).getTime() / 1000, "Read the board", now)).toBeNull();
  });

  it("names yesterday by name and an older day by date", () => {
    expect(olderNote(new Date(2026, 9, 2, 9).getTime() / 1000, "Read the board", now)).toEqual({
      day: "yesterday",
      turn: "Read the board",
    });
    expect(olderNote(new Date(2026, 8, 28, 9).getTime() / 1000, "Ran reports", now)?.day).not.toBe(
      "yesterday",
    );
  });
});

describe("stepLineage", () => {
  it("wraps at either end", () => {
    const lineages = groupLineages([artifact("a", 3), artifact("b", 2), artifact("c", 1)]);
    expect(stepLineage(lineages, "a", -1)).toBe("c");
    expect(stepLineage(lineages, "c", 1)).toBe("a");
    expect(stepLineage(lineages, "b", 1)).toBe("c");
  });
});
