import { describe, expect, it } from "vitest";
import { diffHunks, lineDiff } from "../line-diff";

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

describe("diffHunks", () => {
  const same = (text: string) => ({ kind: "same" as const, text });

  it("keeps two lines either side of a change and folds the runs beyond into counted gaps", () => {
    const before = Array.from({ length: 12 }, (_, index) => `line ${index}`).join("\n");
    const after = before.replace("line 6", "line six");
    expect(diffHunks(lineDiff(before, after))).toEqual([
      { kind: "gap", count: 4 },
      same("line 4"),
      same("line 5"),
      { kind: "del", text: "line 6" },
      { kind: "add", text: "line six" },
      same("line 7"),
      same("line 8"),
      { kind: "gap", count: 3 },
    ]);
  });

  it("joins two changes whose context touches into one hunk", () => {
    const lines = lineDiff("a\nb\nc\nd\ne\nf", "A\nb\nc\nd\ne\nF");
    expect(diffHunks(lines, 2)).toEqual([
      { kind: "del", text: "a" },
      { kind: "add", text: "A" },
      same("b"),
      same("c"),
      same("d"),
      same("e"),
      { kind: "del", text: "f" },
      { kind: "add", text: "F" },
    ]);
  });

  it("returns a diff with no change whole", () => {
    expect(diffHunks(lineDiff("a\nb", "a\nb"))).toEqual([same("a"), same("b")]);
  });
});
