import { activeTableBucket, tableBucketFilters } from "@/lib/table-buckets";
import { describe, expect, it } from "vitest";

const BUCKETS = {
  open: { status: ["Open"] },
  money: { status: ["Open"], kind: ["AmountMismatch", "CustomerBalanceMismatch"] },
  settled: { status: ["Resolved", "Dismissed"] },
} as const;

describe("table buckets", () => {
  it("round-trips every bucket, including one over two fields", () => {
    for (const bucket of Object.keys(BUCKETS) as (keyof typeof BUCKETS)[]) {
      expect(activeTableBucket(tableBucketFilters(BUCKETS[bucket]), BUCKETS)).toBe(bucket);
    }
  });

  it("matches values in any order and an eq for a single value", () => {
    expect(
      activeTableBucket(
        [{ field: "status", operator: "in", value: ["Dismissed", "Resolved"] }],
        BUCKETS,
      ),
    ).toBe("settled");
    expect(activeTableBucket([{ field: "status", operator: "eq", value: "Open" }], BUCKETS)).toBe(
      "open",
    );
  });

  it("matches nothing when a field, a value or an extra filter differs", () => {
    expect(activeTableBucket([], BUCKETS)).toBeNull();
    expect(
      activeTableBucket(
        [
          { field: "status", operator: "in", value: ["Open"] },
          { field: "kind", operator: "in", value: ["AmountMismatch"] },
        ],
        BUCKETS,
      ),
    ).toBeNull();
    expect(
      activeTableBucket(
        [
          { field: "status", operator: "in", value: ["Open"] },
          { field: "status", operator: "in", value: ["Open"] },
        ],
        BUCKETS,
      ),
    ).toBeNull();
    expect(
      activeTableBucket([{ field: "status", operator: "contains", value: "Open" }], BUCKETS),
    ).toBeNull();
    expect(
      activeTableBucket(
        [
          ...tableBucketFilters(BUCKETS.open),
          { field: "objectType", operator: "in", value: ["Invoice"] },
        ],
        BUCKETS,
      ),
    ).toBeNull();
  });
});
