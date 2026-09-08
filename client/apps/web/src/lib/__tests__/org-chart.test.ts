import { describe, expect, it } from "vitest";
import {
  ancestorIds,
  buildPositionTree,
  canReportTo,
  coverSummary,
  descendantIds,
  findNode,
  flattenTree,
  headcountSummary,
  largestGroup,
  matchesPositionSearch,
  pruneTree,
  vacantPositions,
} from "../org-chart";

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;

function position(
  over: Partial<{
    id: string;
    code: string;
    title: string;
    status: string;
    department: string;
    isDrivingPosition: boolean;
    reportsToPositionId: string | null;
  }> = {},
) {
  return {
    id: "pos_1",
    code: "OTR",
    title: "Over-the-Road Driver",
    status: "Active",
    department: "Operations",
    isDrivingPosition: true,
    reportsToPositionId: null,
    ...over,
  };
}

function count(
  key: string,
  workers: number,
  over: Partial<{ drivers: number; terminated: number; staff: number }> = {},
) {
  return { key, workers, drivers: workers, terminated: 0, staff: 0, ...over };
}

const ORG = [
  position({
    id: "ceo",
    code: "CEO",
    title: "Chief Executive",
    isDrivingPosition: false,
    department: "Executive",
  }),
  position({
    id: "ops",
    code: "OPS",
    title: "Operations Manager",
    isDrivingPosition: false,
    reportsToPositionId: "ceo",
  }),
  position({ id: "otr", code: "OTR", title: "Over-the-Road Driver", reportsToPositionId: "ops" }),
  position({ id: "local", code: "LOC", title: "Local Driver", reportsToPositionId: "ops" }),
  position({
    id: "safety",
    code: "SAF",
    title: "Safety Director",
    isDrivingPosition: false,
    department: "Safety",
    reportsToPositionId: "ceo",
  }),
];

describe("buildPositionTree", () => {
  it("hangs positions from the one they report to and rolls the headcount up", () => {
    const tree = buildPositionTree(ORG, [
      count("ceo", 1),
      count("ops", 2),
      count("otr", 30),
      count("local", 10),
      count("", 3),
    ]);
    expect(tree.unplaced).toBe(3);
    expect(tree.roots.map((node) => node.position.id)).toEqual(["ceo"]);
    const ceo = tree.roots[0];
    expect(ceo).toMatchObject({ workers: 1, rolledUp: 43, depth: 0 });
    expect(ceo.children.map((node) => node.position.id)).toEqual(["ops", "safety"]);
    const ops = ceo.children[0];
    expect(ops).toMatchObject({ workers: 2, rolledUp: 42, depth: 1 });
    expect(ops.children.map((node) => [node.position.id, node.rolledUp, node.depth])).toEqual([
      ["otr", 30, 2],
      ["local", 10, 2],
    ]);
  });

  // A chart that silently lost a branch would be worse than one with a loop,
  // so a position whose chain comes back to itself is hung from the top.
  it("hangs a looping position from the top instead of dropping it", () => {
    const tree = buildPositionTree(
      [
        position({ id: "a", title: "A", reportsToPositionId: "b" }),
        position({ id: "b", title: "B", reportsToPositionId: "a" }),
        position({ id: "c", title: "C", reportsToPositionId: "missing" }),
      ],
      [],
    );
    expect(tree.roots.map((node) => node.position.id).sort()).toEqual(["a", "b", "c"]);
    expect(flattenTree(tree.roots)).toHaveLength(3);
  });

  it("puts archived positions after active ones at the same level", () => {
    const tree = buildPositionTree(
      [
        position({ id: "old", title: "Old", status: "Inactive" }),
        position({ id: "small", title: "Small" }),
        position({ id: "big", title: "Big" }),
      ],
      [count("old", 50), count("big", 5), count("small", 1)],
    );
    expect(tree.roots.map((node) => node.position.id)).toEqual(["big", "small", "old"]);
  });

  it("reads top to bottom when flattened", () => {
    const tree = buildPositionTree(ORG, [count("otr", 30), count("local", 10)]);
    expect(flattenTree(tree.roots).map((node) => node.position.id)).toEqual([
      "ceo",
      "ops",
      "otr",
      "local",
      "safety",
    ]);
  });
});

describe("pruneTree", () => {
  it("keeps a match and the positions above it, dropping the rest", () => {
    const tree = buildPositionTree(ORG, []);
    const pruned = flattenTree(pruneTree(tree.roots, "local"));
    expect(pruned.map((node) => node.position.id)).toEqual(["ceo", "ops", "local"]);
    expect(flattenTree(pruneTree(tree.roots, "  "))).toHaveLength(5);
  });

  it("matches title, code or department", () => {
    const row = position();
    expect(matchesPositionSearch(row, "road")).toBe(true);
    expect(matchesPositionSearch(row, "otr")).toBe(true);
    expect(matchesPositionSearch(row, "operations")).toBe(true);
    expect(matchesPositionSearch(row, "safety")).toBe(false);
  });
});

describe("vacantPositions", () => {
  it("lists open positions nobody holds, by title", () => {
    const vacant = vacantPositions(ORG, [count("otr", 30), count("ceo", 1)]);
    expect(vacant.map((row) => row.id)).toEqual(["local", "ops", "safety"]);
  });

  it("ignores archived positions", () => {
    expect(vacantPositions([position({ status: "Inactive" })], [])).toEqual([]);
  });

  // A dispatch desk is held by somebody who logs in, not by a worker; a
  // title only staff hold is filled, not vacant.
  it("counts a title held only by staff as filled", () => {
    const desk = position({ id: "desk", title: "Load Planner", isDrivingPosition: false });
    expect(vacantPositions([desk], [count("desk", 0, { staff: 2 })])).toEqual([]);
    const tree = buildPositionTree([desk], [count("desk", 0, { staff: 2 })]);
    expect(tree.roots[0]).toMatchObject({ workers: 0, staff: 2, people: 2, rolledUp: 2 });
  });
});

describe("coverSummary", () => {
  const delegation = (
    over: Partial<{ startsAt: number; endsAt: number | null; revokedAt: number | null }> = {},
  ) => ({
    startsAt: NOW - 5 * DAY,
    endsAt: NOW + 30 * DAY,
    revokedAt: null,
    ...over,
  });

  it("counts what is in force, what starts later, what ends this week and what is open-ended", () => {
    expect(
      coverSummary(
        [
          delegation(),
          delegation({ endsAt: NOW + 3 * DAY }),
          delegation({ endsAt: null }),
          delegation({ startsAt: NOW + DAY }),
          delegation({ revokedAt: NOW - DAY }),
          delegation({ endsAt: NOW - DAY }),
        ],
        NOW,
      ),
    ).toEqual({ active: 3, scheduled: 1, endingSoon: 1, openEnded: 1 });
  });
});

describe("headcountSummary", () => {
  it("splits driving from non-driving and never goes negative", () => {
    expect(
      headcountSummary({ activeTotal: 40, driverTotal: 30, terminated: 2, staffTotal: 6 }),
    ).toEqual({
      active: 40,
      drivers: 30,
      nonDriving: 10,
      terminated: 2,
      staff: 6,
      people: 46,
      driverShare: 0.75,
    });
    expect(headcountSummary({ activeTotal: 3, driverTotal: 3, terminated: 0 }).people).toBe(3);
    expect(headcountSummary({ activeTotal: 0, driverTotal: 0, terminated: 0 }).driverShare).toBe(0);
  });

  it("finds the largest group", () => {
    expect(largestGroup([{ workers: 2 }, { workers: 9 }, { workers: 4 }])).toEqual({ workers: 9 });
    expect(largestGroup([])).toBeNull();
  });
});

describe("moving positions", () => {
  it("refuses to hang a position from itself or anything beneath it", () => {
    const tree = buildPositionTree(ORG, []);
    expect(canReportTo(tree.roots, "ops", "ceo")).toBe(true);
    expect(canReportTo(tree.roots, "ops", null)).toBe(true);
    expect(canReportTo(tree.roots, "ops", "ops")).toBe(false);
    expect(canReportTo(tree.roots, "ops", "otr")).toBe(false);
    expect(canReportTo(tree.roots, "ceo", "local")).toBe(false);
    expect(canReportTo(tree.roots, "safety", "ops")).toBe(true);
  });

  it("knows everything under a node and everything above one", () => {
    const tree = buildPositionTree(ORG, []);
    const ceo = findNode(tree.roots, "ceo");
    expect(ceo && Array.from(descendantIds(ceo)).sort()).toEqual(["local", "ops", "otr", "safety"]);
    expect(findNode(tree.roots, "missing")).toBeNull();
    expect(ancestorIds(ORG, "local")).toEqual(["ops", "ceo"]);
    expect(ancestorIds(ORG, "ceo")).toEqual([]);
  });

  it("stops climbing a looping chain", () => {
    expect(
      ancestorIds(
        [
          position({ id: "a", reportsToPositionId: "b" }),
          position({ id: "b", reportsToPositionId: "a" }),
        ],
        "a",
      ),
    ).toEqual(["b"]);
  });
});
