import { leaveControlSchema } from "@trenova/shared/types/worker-leave";
import { describe, expect, it } from "vitest";
import { leaveControlTabs } from "../leave-control-tabs";

const RECORD_KEYS = new Set(["id", "version"]);

describe("leave control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = leaveControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(leaveControlSchema.shape).filter((key) => !RECORD_KEYS.has(key));

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  // The page has one job, so it has one tab and the shell draws no tab row.
  it("has the one leave policy tab", () => {
    expect(leaveControlTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["policy", "Leave policy"],
    ]);
  });
});
