import { messageRow } from "@/lib/triage/__tests__/message-fixture";
import { rangeBetween } from "@/stores/triage-store";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { buildListItems } from "../list/list-items";
import { swipeOutcome } from "../list/triage-row";

const NOW = 1_000_000;

function shape(items: ReturnType<typeof buildListItems>): string[] {
  return items.map((item) => (item.type === "heading" ? `# ${item.label}` : item.message.id));
}

describe("buildListItems", () => {
  it("heads a due-first folder with over target, due within the hour and later", () => {
    const items = buildListItems(
      translate,
      [
        messageRow({ id: "a", dueAt: NOW - 10 }),
        messageRow({ id: "b", dueAt: NOW + 600 }),
        messageRow({ id: "c", dueAt: NOW + 7200 }),
      ],
      "Due",
      NOW,
    );
    expect(shape(items)).toEqual(["# Over target", "a", "# Due within the hour", "b", "# Later", "c"]);
    expect(items[0]).toMatchObject({ tone: "danger" });
  });

  it("heads a newest-first folder with the days the mail arrived", () => {
    const day = (seconds: number) => Math.floor(seconds / 86_400) * 86_400;
    const items = buildListItems(
      translate,
      [messageRow({ id: "today", receivedAt: NOW - 60 }), messageRow({ id: "yesterday", receivedAt: NOW - 86_400 })],
      "Newest",
      NOW,
      day,
    );
    expect(shape(items)).toEqual(["# Today", "today", "# Yesterday", "yesterday"]);
  });
});

describe("selecting a range", () => {
  it("selects from the last row picked to the one shift-clicked, either way", () => {
    const ids = ["a", "b", "c", "d"];
    expect(rangeBetween(ids, "b", "d")).toEqual(["b", "c", "d"]);
    expect(rangeBetween(ids, "d", "a")).toEqual(["a", "b", "c", "d"]);
    expect(rangeBetween(ids, null, "c")).toEqual(["c"]);
    expect(rangeBetween(ids, "gone", "c")).toEqual(["c"]);
  });
});

describe("swiping a row", () => {
  it("marks handled far enough left, snoozes far enough right, and does nothing in between", () => {
    expect(swipeOutcome(-90)).toBe("handle");
    expect(swipeOutcome(-89)).toBeNull();
    expect(swipeOutcome(120)).toBe("snooze");
    expect(swipeOutcome(0)).toBeNull();
  });
});
