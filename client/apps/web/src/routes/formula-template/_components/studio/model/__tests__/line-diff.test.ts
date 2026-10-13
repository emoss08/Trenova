import { describe, expect, it } from "vitest";
import { lineDiff } from "../line-diff";

describe("lineDiff", () => {
  it("keeps shared lines and marks what a change removes and adds, in reading order", () => {
    expect(
      lineDiff(
        "let fuel = linehaul * 0.14;\nlinehaul + fuel",
        "let fuel = linehaul * 0.18;\nlinehaul + fuel",
      ),
    ).toEqual([
      { kind: "del", text: "let fuel = linehaul * 0.14;" },
      { kind: "add", text: "let fuel = linehaul * 0.18;" },
      { kind: "same", text: "linehaul + fuel" },
    ]);
  });

  it("reads an empty editor as a new formula, every line added", () => {
    expect(lineDiff("  ", "baseRate * totalDistance\n+ 50")).toEqual([
      { kind: "add", text: "baseRate * totalDistance" },
      { kind: "add", text: "+ 50" },
    ]);
  });

  it("marks a line appended at the end as the only addition", () => {
    expect(lineDiff("a\nb", "a\nb\nc")).toEqual([
      { kind: "same", text: "a" },
      { kind: "same", text: "b" },
      { kind: "add", text: "c" },
    ]);
  });
});
