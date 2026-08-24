import { describe, expect, it } from "vitest";
import { queries } from "@/lib/queries";
import { permitViewQueryKeys } from "@/lib/queries/shipment";

const SHIPMENT_ID = "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV";

/** TanStack Query matches an invalidation key against the head of a cache key. */
function isPrefixOf(prefix: readonly unknown[], full: readonly unknown[]) {
  return prefix.length <= full.length && prefix.every((part, i) => Object.is(part, full[i]));
}

// createQueryKeys prepends ["shipment", <methodName>] to each declared array, so
// a hand-written invalidation key silently matches nothing when it guesses that
// convention wrong. These assertions compare against the real keys, so a rename
// on either side fails here instead of leaving a panel quietly stale.
describe("permitViewQueryKeys", () => {
  const keys = permitViewQueryKeys(SHIPMENT_ID);

  it("invalidates the recorded permit list", () => {
    const full = queries.shipment.permits(SHIPMENT_ID).queryKey;
    expect(keys.some((key) => isPrefixOf(key, full))).toBe(true);
  });

  it("invalidates the persisted requirement list", () => {
    const full = queries.shipment.permitRequirements(SHIPMENT_ID).queryKey;
    expect(keys.some((key) => isPrefixOf(key, full))).toBe(true);
  });

  // The bug this test exists for: recording a permit re-syncs server-side and
  // can release the dispatch hold, but the panel kept rendering the assessment
  // it had before the write.
  it("invalidates the assessment regardless of which version is cached", () => {
    for (const version of [undefined, 1, 7]) {
      const full = queries.shipment.permitAssessment(SHIPMENT_ID, version).queryKey;
      expect(
        keys.some((key) => isPrefixOf(key, full)),
        `no invalidation key matched the assessment at version ${String(version)}`,
      ).toBe(true);
    }
  });

  it("does not invalidate an unrelated shipment's permits", () => {
    const other = queries.shipment.permits("shp_someone_else").queryKey;
    expect(keys.some((key) => isPrefixOf(key, other))).toBe(false);
  });

  it("does not invalidate unrelated shipment queries", () => {
    const billing = queries.shipment.billingReadiness(SHIPMENT_ID).queryKey;
    expect(keys.some((key) => isPrefixOf(key, billing))).toBe(false);
  });
});
