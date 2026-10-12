import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { dueSpan, dueText, groupByDue } from "../due";

const NOW = 1_000_000;

describe("due", () => {
  it("says how far off a target is, in the largest whole unit", () => {
    expect(dueText(translate, dueSpan(NOW + 38 * 60, NOW))).toBe("38m left");
    expect(dueText(translate, dueSpan(NOW + 3 * 3600 + 59 * 60, NOW))).toBe("3h left");
    expect(dueText(translate, dueSpan(NOW - 23 * 3600, NOW))).toBe("23h over");
    expect(dueText(translate, dueSpan(NOW - 26 * 3600, NOW))).toBe("1d over");
  });

  it("warns inside half an hour and is over once the target passes", () => {
    expect(dueSpan(NOW + 29 * 60, NOW).state).toBe("soon");
    expect(dueSpan(NOW + 30 * 60, NOW).state).toBe("later");
    expect(dueSpan(NOW - 1, NOW).state).toBe("over");
  });

  it("groups a due-first folder into over target, within the hour and later, keeping the order", () => {
    const rows = [
      { id: "over", dueAt: NOW - 60, waiting: true },
      { id: "soon", dueAt: NOW + 59 * 60, waiting: true },
      { id: "parked", dueAt: NOW - 60, waiting: false },
      { id: "later", dueAt: NOW + 2 * 3600, waiting: true },
      { id: "none", dueAt: null, waiting: true },
    ];
    expect(
      groupByDue(rows, NOW).map((group) => [group.group, group.items.map((row) => row.id)]),
    ).toEqual([
      ["over", ["over"]],
      ["hour", ["soon"]],
      ["later", ["parked", "later", "none"]],
    ]);
  });
});
