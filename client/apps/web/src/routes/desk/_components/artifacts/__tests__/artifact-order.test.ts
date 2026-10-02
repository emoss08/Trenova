import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { filterArtifacts, orderArtifacts } from "../artifact-order";

function artifact(
  id: string,
  createdAt: number,
  overrides: Partial<AssistantArtifact> = {},
): AssistantArtifact {
  return {
    id,
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "document",
    status: "Ready",
    title: id,
    payload: {},
    sourceToolCallId: "",
    pinned: false,
    createdAt,
    updatedAt: createdAt,
    ...overrides,
  } as AssistantArtifact;
}

describe("the order a conversation's artifacts are listed in", () => {
  // The server hands the list back oldest first; a reader wants what was
  // just made, and what they kept, at the top.
  it("puts pinned artifacts first and the newest first within each group", () => {
    const ordered = orderArtifacts([
      artifact("old", 10),
      artifact("pinned-old", 5, { pinned: true }),
      artifact("new", 30),
      artifact("pinned-new", 20, { pinned: true }),
      artifact("mid", 20),
    ]);

    expect(ordered.map((entry) => entry.id)).toEqual([
      "pinned-new",
      "pinned-old",
      "new",
      "mid",
      "old",
    ]);
  });

  it("keeps the given order for artifacts made in the same second", () => {
    const ordered = orderArtifacts([artifact("a", 7), artifact("b", 7), artifact("c", 7)]);

    expect(ordered.map((entry) => entry.id)).toEqual(["a", "b", "c"]);
  });

  it("does not reorder the list it was given", () => {
    const given = [artifact("old", 1), artifact("new", 2)];
    orderArtifacts(given);

    expect(given.map((entry) => entry.id)).toEqual(["old", "new"]);
  });
});

describe("searching the list", () => {
  const list = [
    artifact("a", 1, { title: "Late loads", kind: "table_view" }),
    artifact("b", 2, { title: "Peak Distributing", kind: "entity_card" }),
    artifact("c", 3, { title: "Handover notes", kind: "document" }),
  ];

  it("matches the title regardless of case, and keeps the order", () => {
    expect(filterArtifacts(list, "LOAD").map((entry) => entry.id)).toEqual(["a"]);
    expect(filterArtifacts(list, "  ").map((entry) => entry.id)).toEqual(["a", "b", "c"]);
  });

  it("matches the kind's name, so 'table' finds every table", () => {
    expect(filterArtifacts(list, "table").map((entry) => entry.id)).toEqual(["a"]);
    expect(filterArtifacts(list, "record").map((entry) => entry.id)).toEqual(["b"]);
  });
});
