import { describe, expect, it } from "vitest";
import { projectScore, scoreLine } from "../training-score";

const t = (message: string | null | undefined, ...args: unknown[]) =>
  (message ?? "").replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]));

describe("projectScore", () => {
  it("passes at the mark and fails below it", () => {
    expect(projectScore("80", "80.00")).toEqual({ kind: "pass", passing: 80, score: 80 });
    expect(projectScore("79.5", "80.00")).toEqual({ kind: "fail", passing: 80, score: 79.5 });
  });

  it("waits for a score on a scored course and says the mark", () => {
    expect(projectScore("", "80.00")).toEqual({ kind: "waiting", passing: 80 });
    expect(scoreLine(projectScore(null, "80.00"), t)).toBe("Passes at 80%.");
  });

  it("treats a course with no mark as unscored, whatever is typed", () => {
    expect(projectScore("55", null)).toEqual({ kind: "unscored" });
    expect(projectScore(null, "")).toEqual({ kind: "unscored" });
  });

  it("refuses a score outside 0 to 100 or not a number", () => {
    expect(projectScore("101", "80")).toEqual({ kind: "invalid" });
    expect(projectScore("-1", "80")).toEqual({ kind: "invalid" });
    expect(projectScore("abc", "80")).toEqual({ kind: "invalid" });
  });

  it("words the live line for a pass and a fail", () => {
    expect(scoreLine(projectScore("92", "80"), t)).toBe("Passes at 80%. This would pass.");
    expect(scoreLine(projectScore("70", "80"), t)).toBe("Passes at 80%. This would fail.");
  });
});
