import { groupThread, type ThreadEntry } from "@/components/assistant/thread-view";
import type { AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { shareItemsByKey, shareMapValues } from "../keyed-sharing";

type Row = { id: string; text: string; tags: string[] };

const byId = (row: Row) => row.id;

let sequence = 0;
function message(overrides: Partial<AssistantMessage>): AssistantMessage {
  sequence += 1;
  return {
    id: `msg_${sequence}`,
    threadId: "t1",
    sequence,
    role: "User",
    content: "",
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 1_700_000_000 + sequence,
    kind: "Message",
    ...overrides,
  };
}

/**
 * A transcript reading an older page puts it in front of what is drawn. Kept
 * by key, every row still equal to what it was stays the same object, so its
 * memoized component is skipped rather than drawn again.
 */
describe("shareItemsByKey", () => {
  it("keeps every unchanged item when items arrive in front", () => {
    const previous: Row[] = [
      { id: "c", text: "three", tags: ["x"] },
      { id: "d", text: "four", tags: [] },
    ];
    const next: Row[] = [
      { id: "a", text: "one", tags: [] },
      { id: "b", text: "two", tags: [] },
      { id: "c", text: "three", tags: ["x"] },
      { id: "d", text: "four", tags: [] },
    ];

    const shared = shareItemsByKey(previous, next, byId);

    expect(shared).toHaveLength(4);
    expect(shared[2]).toBe(previous[0]);
    expect(shared[3]).toBe(previous[1]);
    expect(shared[0]).toBe(next[0]);
  });

  it("takes the new item where it changed, keeping the parts that did not", () => {
    const tags = ["x"];
    const previous: Row[] = [{ id: "c", text: "three", tags }];
    const next: Row[] = [{ id: "c", text: "three, edited", tags: ["x"] }];

    const [row] = shareItemsByKey(previous, next, byId);

    expect(row).not.toBe(previous[0]);
    expect(row.text).toBe("three, edited");
    expect(row.tags).toBe(tags);
  });

  it("hands back the previous list itself when nothing changed", () => {
    const previous: Row[] = [{ id: "a", text: "one", tags: [] }];

    expect(shareItemsByKey(previous, [{ id: "a", text: "one", tags: [] }], byId)).toBe(previous);
    expect(shareItemsByKey(undefined, previous, byId)).toBe(previous);
  });

  it("is a new list when an item went, even if the rest are kept", () => {
    const previous: Row[] = [
      { id: "a", text: "one", tags: [] },
      { id: "b", text: "two", tags: [] },
    ];

    const shared = shareItemsByKey(previous, [{ id: "b", text: "two", tags: [] }], byId);

    expect(shared).not.toBe(previous);
    expect(shared[0]).toBe(previous[1]);
  });

  it("keeps a transcript's entries when an older page is grouped in front of them", () => {
    const older = [
      message({ role: "User", content: "Where is S0?" }),
      message({ role: "Assistant", content: "S0 delivered." }),
    ];
    const newer = [
      message({ role: "User", content: "Where are S1 and S2?" }),
      message({
        role: "Assistant",
        content: "Checking both.",
        toolCalls: [{ id: "c1", name: "get_shipment", arguments: { proNumber: "S1" } }],
      }),
      message({ role: "Tool", toolCallId: "c1", toolName: "get_shipment", content: "one" }),
      message({ role: "Assistant", content: "S1 is in transit." }),
    ];
    const entryKey = (entry: ThreadEntry) => entry.message.id;

    const before = groupThread(newer);
    const after = shareItemsByKey(before, groupThread([...older, ...newer]), entryKey);

    expect(after).toHaveLength(5);
    expect(after.slice(2)).toEqual(before);
    after.slice(2).forEach((entry, index) => expect(entry).toBe(before[index]));
  });
});

describe("shareMapValues", () => {
  it("keeps each value still equal to the one under its key", () => {
    const steps = [{ name: "lookup" }];
    const previous = new Map([["m2", steps]]);
    const next = new Map([
      ["m1", [{ name: "older" }]],
      ["m2", [{ name: "lookup" }]],
    ]);

    const shared = shareMapValues(previous, next);

    expect(shared.get("m2")).toBe(steps);
    expect(shared.get("m1")).toBe(next.get("m1"));
  });

  it("hands back the previous map itself when nothing changed", () => {
    const previous = new Map([["m1", [{ name: "lookup" }]]]);

    expect(shareMapValues(previous, new Map([["m1", [{ name: "lookup" }]]]))).toBe(previous);
  });
});
