import { describe, expect, it } from "vitest";
import { activeBucket, bucketCount, bucketFilter } from "../_components/ledger-filters";
import {
  backfillObjectTypes,
  backfillSchema,
  pauseSchema,
  skipSchema,
} from "../_components/ledger-schemas";

describe("ledger buckets", () => {
  it("round-trips each bucket through its filter", () => {
    for (const bucket of ["sent", "moving", "held", "attention"] as const) {
      expect(activeBucket(bucketFilter(bucket))).toBe(bucket);
    }
  });

  it("recognises a bucket whatever order its statuses come in", () => {
    expect(
      activeBucket([{ field: "status", operator: "in", value: ["DeadLettered", "Blocked"] }]),
    ).toBe("attention");
  });

  it("is no bucket for any other filter", () => {
    expect(activeBucket([])).toBeNull();
    expect(activeBucket([{ field: "status", operator: "eq", value: "Blocked" }])).toBeNull();
    expect(activeBucket([{ field: "status", operator: "in", value: ["Blocked"] }])).toBeNull();
    expect(activeBucket([{ field: "objectType", operator: "in", value: ["Synced"] }])).toBeNull();
    expect(
      activeBucket([
        ...bucketFilter("sent"),
        { field: "objectType", operator: "eq", value: "Invoice" },
      ]),
    ).toBeNull();
  });

  it("adds the counts in a bucket and ignores the rest", () => {
    const counts = [
      { status: "Queued" as const, count: 2 },
      { status: "InFlight" as const, count: 1 },
      { status: "Retrying" as const, count: 4 },
      { status: "Synced" as const, count: 9 },
    ];
    expect(bucketCount(counts, "moving")).toBe(7);
    expect(bucketCount(counts, "attention")).toBe(0);
  });
});

describe("ledger schemas", () => {
  it("requires a reason to skip, and trims it", () => {
    expect(skipSchema.safeParse({ reason: "   " }).success).toBe(false);
    expect(skipSchema.parse({ reason: "  by hand " })).toEqual({ reason: "by hand" });
    expect(skipSchema.safeParse({ reason: "x".repeat(501) }).success).toBe(false);
  });

  it("lets a pause go without a reason", () => {
    expect(pauseSchema.parse({ reason: "" })).toEqual({ reason: "" });
  });

  it("wants a backfill range that runs forwards and not into the future", () => {
    const schema = backfillSchema(1_000);
    expect(schema.safeParse({ rangeStart: 10, rangeEnd: 20, objectTypes: [] }).success).toBe(true);
    expect(schema.safeParse({ rangeStart: 20, rangeEnd: 10, objectTypes: [] }).success).toBe(false);
    expect(schema.safeParse({ rangeStart: 10, rangeEnd: 1_001, objectTypes: [] }).success).toBe(
      false,
    );
    expect(
      schema.safeParse({ rangeStart: 10, rangeEnd: 20, objectTypes: ["Timesheet"] }).success,
    ).toBe(false);
  });
});

describe("backfillObjectTypes", () => {
  it("never offers customers or vendors, which are not backfilled", () => {
    expect(backfillObjectTypes(true)).not.toContain("Customer");
    expect(backfillObjectTypes(true)).not.toContain("CarrierVendor");
    expect(backfillObjectTypes(true)).not.toContain("DriverVendor");
  });

  it("offers owner-operator settlements only while they are sent", () => {
    expect(backfillObjectTypes(false)).toEqual([
      "Invoice",
      "DebitMemo",
      "CreditMemo",
      "CustomerPayment",
      "CreditApplication",
      "CarrierBill",
      "CarrierBillPayment",
    ]);
    expect(backfillObjectTypes(true)).toEqual(
      expect.arrayContaining(["DriverBill", "DriverBillPayment"]),
    );
  });
});
