import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { deskRailRowKey, deskRailRows, type DeskRailRow } from "../desk-rail-rows";
import type { DeskThreadShelf } from "../desk-threads";

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

/**
 * The rail draws its shelves as one flat list so only the rows in view are
 * mounted. The server pages by date, so while more remains the next page
 * arrives at the end of the dated shelves, above the parked cases.
 */
describe("deskRailRows", () => {
  const dated: DeskThreadShelf[] = [
    { key: "pinned", threads: [thread("p")] },
    { key: "today", threads: [thread("a"), thread("b")] },
  ];

  it("lays out each shelf's heading above its conversations", () => {
    expect(keys(deskRailRows(dated, false))).toEqual(["s:pinned", "c:p", "s:today", "c:a", "c:b"]);
  });

  it("ends with the place the next page arrives while more remains", () => {
    expect(keys(deskRailRows(dated, true)).at(-1)).toBe("more");
  });

  it("puts the next page above the snoozed and settled cases", () => {
    const shelves: DeskThreadShelf[] = [
      ...dated,
      { key: "snoozed", threads: [thread("z")] },
      { key: "settled", threads: [thread("s")] },
    ];

    expect(keys(deskRailRows(shelves, true))).toEqual([
      "s:pinned",
      "c:p",
      "s:today",
      "c:a",
      "c:b",
      "more",
      "s:snoozed",
      "c:z",
      "s:settled",
      "c:s",
    ]);
  });

  it("places it once even when only parked shelves were read", () => {
    const shelves: DeskThreadShelf[] = [{ key: "settled", threads: [thread("s")] }];

    expect(keys(deskRailRows(shelves, true))).toEqual(["more", "s:settled", "c:s"]);
  });

  it("is empty when there is nothing and nothing more", () => {
    expect(deskRailRows([], false)).toEqual([]);
  });
});
