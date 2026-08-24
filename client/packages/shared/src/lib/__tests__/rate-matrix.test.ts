import { describe, expect, it } from "vitest";
import {
  axisLabel,
  bandLabel,
  buildMatrixGrid,
  bucketKeyOf,
  cellBoundsAt,
  coordinateKey,
  findMatrixCoverageIssues,
  sliceBuckets,
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
    ...overrides,
  } as RateMatrixDimension;
}

function cell(overrides: Partial<RateMatrixCell> = {}): RateMatrixCell {
  return {
    d0Key: "",
    d1Key: "",
    d2Key: "",
    d3Key: "",
    value: 0,
    deficitEligible: true,
    ...overrides,
  } as RateMatrixCell;
}

describe("axisLabel", () => {
  it("prefers what the author typed", () => {
    expect(axisLabel(dimension(0, { label: "Origin market" }))).toBe("Origin market");
  });

  it("falls back to the kind so an unnamed axis still reads as something", () => {
    expect(axisLabel(dimension(0, { kind: "WeightBreak" }))).toBe("Weight break");
  });

  it("treats a whitespace label as unnamed", () => {
    expect(axisLabel(dimension(0, { kind: "FreightClass", label: "   " }))).toBe("Freight class");
  });
});

describe("bandLabel", () => {
  // The engine's upper bound is exclusive. Printing "1,000 – 2,000" would read
  // as inclusive and mislead somebody checking where 2,000 lb prices.
  it("names the upper bound as the point the next band takes over", () => {
    expect(bandLabel(1000, 2000)).toBe("1,000 – under 2,000");
  });

  it("says a top band with no ceiling catches everything above it", () => {
    expect(bandLabel(20000, null)).toBe("20,000 and over");
  });

  it("says a bottom band with no floor catches everything below it", () => {
    expect(bandLabel(null, 500)).toBe("under 500");
  });
});

describe("bucketKeyOf", () => {
  it("identifies an exact axis by its key", () => {
    expect(bucketKeyOf(cell({ d0Key: "SE" }), dimension(0))).toBe("SE");
  });

  // Two bands starting at the same weight but ending at different ones are
  // different buckets. Collapsing them onto the lower bound would hide one row
  // of the tariff entirely.
  it("identifies a range axis by both bounds, not just the floor", () => {
    const banded = dimension(1, { matchMode: "Range", kind: "WeightBreak" });

    const first = bucketKeyOf(cell({ d1Min: 1000, d1Max: 2000 }), banded);
    const second = bucketKeyOf(cell({ d1Min: 1000, d1Max: 5000 }), banded);

    expect(first).not.toBe(second);
  });
});

describe("cellBoundsAt", () => {
  it("reads decimals the schema has already parsed to numbers", () => {
    expect(cellBoundsAt(cell({ d2Min: 1000.5, d2Max: 2000 }), 2)).toEqual({
      min: 1000.5,
      max: 2000,
    });
  });

  it("treats an absent bound as open rather than as zero", () => {
    expect(cellBoundsAt(cell({ d0Min: 500 }), 0)).toEqual({ min: 500, max: null });
  });
});

describe("buildMatrixGrid", () => {
  const origin = dimension(0, { label: "Origin" });
  const destination = dimension(1, { label: "Destination" });

  const cells = [
    cell({ d0Key: "SE", d1Key: "MW", value: 2.1 }),
    cell({ d0Key: "SE", d1Key: "NE", value: 2.45 }),
    cell({ d0Key: "MW", d1Key: "MW", value: 1.9 }),
  ];

  it("lays the flat cell list out as a sheet", () => {
    const grid = buildMatrixGrid({
      dimensions: [origin, destination],
      cells,
      rowPosition: 0,
      columnPosition: 1,
    });

    expect(grid.rows.map((row) => row.key)).toEqual(["MW", "SE"]);
    expect(grid.columns.map((column) => column.key)).toEqual(["MW", "NE"]);
    expect(grid.cells.get(coordinateKey("SE", "NE"))?.value).toBe(2.45);
  });

  // A coordinate the sheet draws but nothing prices is the failure somebody
  // actually hits: the lane resolves, the matrix returns nothing, and the
  // shipment rates at zero.
  it("names the coordinates it draws but nothing prices", () => {
    const grid = buildMatrixGrid({
      dimensions: [origin, destination],
      cells,
      rowPosition: 0,
      columnPosition: 1,
    });

    expect(grid.blanks).toEqual([coordinateKey("MW", "NE")]);
  });

  it("orders banded rows by their floor, not alphabetically", () => {
    const weight = dimension(0, { matchMode: "Range", kind: "WeightBreak" });

    const grid = buildMatrixGrid({
      dimensions: [weight],
      cells: [
        cell({ d0Min: 20000, value: 1.1 }),
        cell({ d0Min: 0, d0Max: 1000, value: 3.0 }),
        cell({ d0Min: 1000, d0Max: 20000, value: 1.8 }),
      ],
      rowPosition: 0,
      columnPosition: null,
    });

    expect(grid.rows.map((row) => row.min)).toEqual([0, 1000, 20000]);
  });

  // A three-axis matrix is a stack of sheets. A cell from another sheet folded
  // into this one would show a rate at a coordinate that does not price it.
  it("keeps cells from other slices off the sheet", () => {
    const weight = dimension(2, { matchMode: "Range", kind: "WeightBreak" });

    const grid = buildMatrixGrid({
      dimensions: [origin, destination, weight],
      cells: [
        cell({ d0Key: "SE", d1Key: "MW", d2Min: 0, d2Max: 1000, value: 3.0 }),
        cell({ d0Key: "SE", d1Key: "MW", d2Min: 1000, value: 1.8 }),
      ],
      rowPosition: 0,
      columnPosition: 1,
      slice: { 2: "1000.." },
    });

    expect(grid.cells.get(coordinateKey("SE", "MW"))?.value).toBe(1.8);
    expect(grid.cells.size).toBe(1);
  });

  it("reports a coordinate carrying two rates", () => {
    const grid = buildMatrixGrid({
      dimensions: [origin, destination],
      cells: [
        cell({ d0Key: "SE", d1Key: "MW", value: 2.1 }),
        cell({ d0Key: "SE", d1Key: "MW", value: 9.99 }),
      ],
      rowPosition: 0,
      columnPosition: 1,
    });

    expect(grid.duplicates).toEqual([coordinateKey("SE", "MW")]);
  });

  it("draws a single-axis matrix as one column", () => {
    const grid = buildMatrixGrid({
      dimensions: [origin],
      cells: [cell({ d0Key: "SE", value: 2.1 })],
      rowPosition: 0,
      columnPosition: null,
    });

    expect(grid.columns).toHaveLength(1);
    expect(grid.cells.get(coordinateKey("SE", ""))?.value).toBe(2.1);
  });

  it("returns nothing rather than throwing when the row axis is not declared", () => {
    const grid = buildMatrixGrid({
      dimensions: [origin],
      cells: [cell({ d0Key: "SE" })],
      rowPosition: 3,
      columnPosition: null,
    });

    expect(grid.rows).toHaveLength(0);
  });
});

describe("sliceBuckets", () => {
  it("lists each distinct band once, in order", () => {
    const weight = dimension(2, { matchMode: "Range", kind: "WeightBreak" });

    const buckets = sliceBuckets(weight, [
      cell({ d2Min: 1000, d2Max: 20000 }),
      cell({ d2Min: 0, d2Max: 1000 }),
      cell({ d2Min: 1000, d2Max: 20000 }),
    ]);

    expect(buckets.map((bucket) => bucket.min)).toEqual([0, 1000]);
  });
});

describe("findMatrixCoverageIssues", () => {
  const weight = dimension(0, { matchMode: "Range", kind: "WeightBreak", label: "Weight" });

  it("says so when a matrix declares no axes", () => {
    const issues = findMatrixCoverageIssues([], [cell()]);

    expect(issues).toHaveLength(1);
    expect(issues[0]?.kind).toBe("empty");
  });

  it("says so when a matrix has axes but no rates", () => {
    const issues = findMatrixCoverageIssues([weight], []);

    expect(issues[0]?.kind).toBe("empty");
  });

  // Bands are half open, so one ending exactly where the next begins is
  // contiguous. Treating that as an overlap would flag every correctly written
  // tariff in the product.
  it("accepts bands that meet exactly", () => {
    const issues = findMatrixCoverageIssues(
      [weight],
      [cell({ d0Min: 0, d0Max: 1000 }), cell({ d0Min: 1000 })],
    );

    expect(issues).toEqual([]);
  });

  it("finds the hole between two bands", () => {
    const issues = findMatrixCoverageIssues(
      [weight],
      [cell({ d0Min: 0, d0Max: 1000 }), cell({ d0Min: 5000 })],
    );

    expect(issues.map((issue) => issue.kind)).toContain("gap");
    expect(issues[0]?.message).toContain("1,000");
    expect(issues[0]?.message).toContain("5,000");
  });

  it("finds two bands claiming the same weight", () => {
    const issues = findMatrixCoverageIssues(
      [weight],
      [cell({ d0Min: 0, d0Max: 2000 }), cell({ d0Min: 1000 })],
    );

    expect(issues.map((issue) => issue.kind)).toContain("overlap");
  });

  // A top band with a ceiling is the classic tariff mistake: the sheet looks
  // complete, and the one shipment heavier than anybody planned for rates at
  // nothing.
  it("finds a top band that leaves everything above it unpriced", () => {
    const issues = findMatrixCoverageIssues(
      [weight],
      [cell({ d0Min: 0, d0Max: 1000 }), cell({ d0Min: 1000, d0Max: 20000 })],
    );

    expect(issues.map((issue) => issue.kind)).toContain("unbounded");
  });

  it("finds a bottom band that leaves light freight unpriced", () => {
    const issues = findMatrixCoverageIssues([weight], [cell({ d0Min: 500 })]);

    expect(issues.map((issue) => issue.kind)).toContain("gap");
  });

  it("finds two rates at the same coordinate", () => {
    const zone = dimension(0);

    const issues = findMatrixCoverageIssues(
      [zone],
      [cell({ d0Key: "SE", value: 2.1 }), cell({ d0Key: "SE", value: 9.99 })],
    );

    expect(issues.map((issue) => issue.kind)).toEqual(["duplicate"]);
  });

  it("reports a repeated coordinate once, however many times it repeats", () => {
    const zone = dimension(0);

    const issues = findMatrixCoverageIssues(
      [zone],
      [cell({ d0Key: "SE" }), cell({ d0Key: "SE" }), cell({ d0Key: "SE" })],
    );

    expect(issues).toHaveLength(1);
  });

  it("passes a complete exact grid", () => {
    const origin = dimension(0);
    const destination = dimension(1);

    const issues = findMatrixCoverageIssues(
      [origin, destination],
      [cell({ d0Key: "SE", d1Key: "MW" }), cell({ d0Key: "SE", d1Key: "NE" })],
    );

    expect(issues).toEqual([]);
  });
});
