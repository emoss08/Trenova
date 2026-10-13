import { describe, expect, it } from "vitest";
import { promisePaidShare } from "../collection-labels";

describe("promisePaidShare", () => {
  it("is the share of the promise paid since it was made", () => {
    expect(promisePaidShare({ amountMinor: 100_000, paidSinceMinor: 25_000 })).toBe(25);
  });

  it("rounds down so a promise is never shown as fully paid early", () => {
    expect(promisePaidShare({ amountMinor: 100_000, paidSinceMinor: 99_999 })).toBe(99);
  });

  it("stops at 100 when more than the promise was paid", () => {
    expect(promisePaidShare({ amountMinor: 100_000, paidSinceMinor: 150_000 })).toBe(100);
  });

  it("is zero for a promise with no amount", () => {
    expect(promisePaidShare({ amountMinor: 0, paidSinceMinor: 10 })).toBe(0);
  });
});
