import { describe, expect, it } from "vitest";
import {
  addPostalRange,
  axisLabel,
  buildMatrixGrid,
  comparePostal,
  normalizePostal,
  postalRangeLabel,
  postalRangeProblem,
  postalRangesOf,
  rerangePostalAxis,
  withoutPostalRange,
  type MatrixAxisPosition,
} from "../rate-matrix";
import type { RateMatrixCell, RateMatrixDimension } from "../../types/rate";

function dimension(
  position: MatrixAxisPosition,
  overrides: Partial<RateMatrixDimension> = {},
): RateMatrixDimension {
  return {
    position,
    kind: "Zone",
    matchMode: "Exact",
    label: "",
    keyNormalization: "None",
    rangeOverflow: "Error",
    ...overrides,
  } as RateMatrixDimension;
}

const POSTAL = dimension(0, { kind: "ZipRange", matchMode: "Range" });
const WEIGHT = dimension(1, { kind: "WeightBreak", matchMode: "Range" });

function cell(overrides: Partial<RateMatrixCell> = {}): RateMatrixCell {
  return {
    d0Key: "",
    d1Key: "",
    d2Key: "",
    d3Key: "",
    d0KeyTo: "",
    d1KeyTo: "",
    d2KeyTo: "",
    d3KeyTo: "",
    value: 0,
    deficitEligible: true,
    ...overrides,
  } as RateMatrixCell;
}

describe("postal range axes", () => {
  it("names the kind a postal range", () => {
    expect(axisLabel(POSTAL)).toBe("Postal range");
  });

  it("labels a range by both ends, and a one-code range by its code", () => {
    expect(postalRangeLabel({ from: "100", to: "149" })).toBe("100 through 149");
    expect(postalRangeLabel({ from: "300", to: "300" })).toBe("300");
  });

  it("normalises a typed code the way the engine matches it", () => {
    expect(normalizePostal("  k1a ")).toBe("K1A");
  });

  it("orders codes character by character in Unicode order, not numerically", () => {
    expect(comparePostal("9", "10")).toBeGreaterThan(0);
    expect(comparePostal("A0", "9Z")).toBeGreaterThan(0);
    expect(comparePostal("100", "100")).toBe(0);
  });

  it("lays each range out as a row, in code order, with its cells across the columns", () => {
    const cells = [
      cell({ d0Key: "150", d0KeyTo: "199", d1Min: 0, d1Max: 500, value: 4 }),
      cell({ d0Key: "100", d0KeyTo: "149", d1Min: 0, d1Max: 500, value: 2 }),
      cell({ d0Key: "100", d0KeyTo: "149", d1Min: 500, value: 3 }),
    ];

    const grid = buildMatrixGrid({
      dimensions: [POSTAL, WEIGHT],
      cells,
      rowPosition: 0,
      columnPosition: 1,
    });

    expect(grid.rows.map((row) => row.label)).toEqual(["100 through 149", "150 through 199"]);
    expect(grid.rows[0]?.postal).toEqual({ from: "100", to: "149" });
    expect(grid.columns.map((column) => column.key)).toEqual(["0..500", "500.."]);
    expect(grid.blanks).toEqual([`${grid.rows[1]?.key}\u0000500..`]);
  });

  it("lists an axis's ranges once each, in order", () => {
    const cells = [
      cell({ d0Key: "150", d0KeyTo: "199", d1Min: 0 }),
      cell({ d0Key: "100", d0KeyTo: "149", d1Min: 0 }),
      cell({ d0Key: "100", d0KeyTo: "149", d1Min: 500 }),
    ];
    expect(postalRangesOf(POSTAL, cells)).toEqual([
      { from: "100", to: "149" },
      { from: "150", to: "199" },
    ]);
  });
});

describe("postalRangeProblem", () => {
  const existing = [
    { from: "100", to: "149" },
    { from: "200", to: "249" },
  ];

  it("accepts a range that fits beside the others", () => {
    expect(postalRangeProblem({ from: "150", to: "199" }, existing)).toBeNull();
  });

  it("needs both ends", () => {
    expect(postalRangeProblem({ from: "150", to: "" }, existing)).toBe(
      "A postal range needs its first and its last postal code.",
    );
  });

  it("refuses ends of different lengths", () => {
    expect(postalRangeProblem({ from: "15", to: "199" }, existing)).toBe(
      "Both ends of a postal range have the same number of characters.",
    );
  });

  it("refuses a range that runs backwards", () => {
    expect(postalRangeProblem({ from: "199", to: "150" }, existing)).toBe(
      "199 comes after 150; a range starts at its lower end.",
    );
  });

  it("refuses a length the axis's other ranges do not have", () => {
    expect(postalRangeProblem({ from: "15000", to: "19999" }, existing)).toBe(
      "Every range on this axis has 3 characters.",
    );
  });

  it("refuses a range sharing a code with another, even at one end", () => {
    expect(postalRangeProblem({ from: "149", to: "160" }, existing)).toBe(
      "This range overlaps 100 through 149, so a postal code in both would match twice.",
    );
  });
});

describe("editing postal ranges", () => {
  const cells = [
    cell({ id: "c1", d0Key: "100", d0KeyTo: "149", d1Min: 0, d1Max: 500, value: 2 }),
    cell({ id: "c2", d0Key: "100", d0KeyTo: "149", d1Min: 500, value: 3 }),
    cell({ id: "c3", d0Key: "150", d0KeyTo: "199", d1Min: 0, d1Max: 500, value: 4 }),
  ];

  it("adds a range as a blank cell for every band the other axis already prices", () => {
    const added = addPostalRange({
      dimensions: [POSTAL, WEIGHT],
      cells,
      position: 0,
      range: { from: "200", to: "249" },
    });

    expect(added).toEqual([
      expect.objectContaining({
        id: undefined,
        d0Key: "200",
        d0KeyTo: "249",
        d1Min: 0,
        d1Max: 500,
        value: 0,
      }),
      expect.objectContaining({
        id: undefined,
        d0Key: "200",
        d0KeyTo: "249",
        d1Min: 500,
        value: 0,
      }),
    ]);
  });

  it("adds one cell to a matrix whose only axis is the range", () => {
    expect(
      addPostalRange({
        dimensions: [POSTAL],
        cells: [],
        position: 0,
        range: { from: "1", to: "4" },
      }),
    ).toEqual([expect.objectContaining({ d0Key: "1", d0KeyTo: "4", value: 0 })]);
  });

  it("adds nothing when the other axes price nothing yet", () => {
    expect(
      addPostalRange({
        dimensions: [POSTAL, WEIGHT],
        cells: [],
        position: 0,
        range: { from: "1", to: "4" },
      }),
    ).toEqual([]);
  });

  it("moves every cell of a range, keeping their values", () => {
    const moved = rerangePostalAxis(
      cells,
      POSTAL,
      { from: "100", to: "149" },
      { from: "100", to: "139" },
    );
    expect(moved.map((entry) => [entry.id, entry.d0Key, entry.d0KeyTo, entry.value])).toEqual([
      ["c1", "100", "139", 2],
      ["c2", "100", "139", 3],
      ["c3", "150", "199", 4],
    ]);
  });

  it("removes every cell of a range and nothing else", () => {
    expect(
      withoutPostalRange(cells, POSTAL, { from: "100", to: "149" }).map((entry) => entry.id),
    ).toEqual(["c3"]);
  });
});
