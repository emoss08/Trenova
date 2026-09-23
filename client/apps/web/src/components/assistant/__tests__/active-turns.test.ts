import { assistantLiveTurnListSchema, type AssistantLiveTurn } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { liveReplyCount, liveThreadIds } from "../active-turns";

function turn(overrides: Partial<AssistantLiveTurn> = {}): AssistantLiveTurn {
  return {
    turnId: "atrn_1",
    threadId: "athr_1",
    threadTitle: "Where is SEED-SHP-001?",
    origin: "Person",
    startedAt: 1_700_000_000,
    ...overrides,
  };
}

/*
Fixtures follow GET /assistant/turns/active/: { items: [{ turnId, threadId,
threadTitle, origin, startedAt }] }. The server writes an empty Go slice as
null, and a conversation can for a moment show a closing turn and the next.
*/
describe("assistantLiveTurnListSchema", () => {
  it("reads the list the server sends", () => {
    const parsed = assistantLiveTurnListSchema.parse({
      items: [
        {
          turnId: "atrn_1",
          threadId: "athr_1",
          threadTitle: "Blocked invoices",
          origin: "DecisionFollowUp",
          startedAt: 1_700_000_000,
        },
      ],
    });

    expect(parsed.items).toEqual([
      {
        turnId: "atrn_1",
        threadId: "athr_1",
        threadTitle: "Blocked invoices",
        origin: "DecisionFollowUp",
        startedAt: 1_700_000_000,
      },
    ]);
  });

  it("reads a null list as no replies", () => {
    expect(assistantLiveTurnListSchema.parse({ items: null }).items).toEqual([]);
  });

  it("reads an untitled conversation's null title as empty", () => {
    const parsed = assistantLiveTurnListSchema.parse({
      items: [{ turnId: "t", threadId: "h", threadTitle: null, origin: "Person", startedAt: 1 }],
    });

    expect(parsed.items[0].threadTitle).toBe("");
  });
});

describe("liveThreadIds", () => {
  it("is empty before the list is known and when nothing is being written", () => {
    expect(liveThreadIds(undefined).size).toBe(0);
    expect(liveThreadIds({ items: [] }).size).toBe(0);
  });

  it("names every conversation with a reply being written", () => {
    const ids = liveThreadIds({
      items: [turn(), turn({ turnId: "atrn_2", threadId: "athr_2" })],
    });

    expect([...ids].sort()).toEqual(["athr_1", "athr_2"]);
  });
});

describe("liveReplyCount", () => {
  it("counts replies across conversations", () => {
    expect(
      liveReplyCount({
        items: [
          turn(),
          turn({ turnId: "atrn_2", threadId: "athr_2" }),
          turn({ turnId: "atrn_3", threadId: "athr_3" }),
        ],
      }),
    ).toBe(3);
  });

  // A conversation writes one reply at a time; two of its turns in the list
  // together are one closing and the next starting, not two replies.
  it("counts a conversation once when two of its turns overlap", () => {
    expect(
      liveReplyCount({ items: [turn(), turn({ turnId: "atrn_2", startedAt: 1_700_000_100 })] }),
    ).toBe(1);
  });
});
