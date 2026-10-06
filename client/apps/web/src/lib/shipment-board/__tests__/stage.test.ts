import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { STAGE_META, stageGroups, stageRankLookup } from "../stage";

const t = ((message: string | null | undefined) => message ?? "") as TranslateFn;

const SUMMARY = [
  { stage: "Moving" as const, rank: 3, count: 30, revenue: "41200.5" },
  { stage: "Late" as const, rank: 1, count: 2, revenue: "3000" },
  { stage: "Scheduled" as const, rank: 4, count: 0, revenue: "0" },
];

describe("stage", () => {
  it("describes every server stage", () => {
    expect(Object.keys(STAGE_META).sort()).toEqual(
      ["Canceled", "Delivered", "Late", "Moving", "NeedsCoverage", "Scheduled"].sort(),
    );
  });

  it("builds groups from the server summary, ordered and keyed by the server's rank", () => {
    const groups = stageGroups(SUMMARY, t);
    expect(groups.map((g) => [g.key, g.label, g.count])).toEqual([
      [1, "Needs attention", 2],
      [3, "Moving", 30],
    ]);
    expect(groups[1].aggregate).toBe("$41,201");
    expect(groups[0].swatchClassName).toBe("bg-danger");
  });

  it("looks a row's group up by its stage, using the server's ranks", () => {
    const rankOf = stageRankLookup(SUMMARY);
    expect(rankOf("Late")).toBe(1);
    expect(rankOf("Moving")).toBe(3);
    expect(rankOf("Delivered")).toBe("Delivered");
    expect(rankOf(undefined)).toBe("unknown");
  });
});
