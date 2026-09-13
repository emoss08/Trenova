import { selectOptionsQueryFilter } from "@/lib/select-options-cache";
import { describe, expect, it } from "vitest";

const matchesPtoPolicy = selectOptionsQueryFilter("PTO_POLICY");

function matches(queryKey: unknown[]): boolean {
  return matchesPtoPolicy.predicate({ queryKey });
}

describe("selectOptionsQueryFilter", () => {
  // The three queries that read a resource's options put the resource in three
  // different positions, so all three have to be caught by one filter.
  it("matches the autocomplete's option search", () => {
    expect(
      matches(["autocomplete-search", undefined, "", undefined, { resource: "PTO_POLICY" }, 20]),
    ).toBe(true);
  });

  it("matches the autocomplete's selected-value lookup", () => {
    expect(
      matches([
        "autocomplete-option",
        undefined,
        undefined,
        "ptop_1",
        { resource: "PTO_POLICY" },
        "PTO_POLICY",
        undefined,
      ]),
    ).toBe(true);
  });

  it("matches a single option read back by id", () => {
    expect(matches(["select-option", "PTO_POLICY", "ptop_1", null])).toBe(true);
  });

  it("leaves another resource's options alone", () => {
    expect(
      matches(["autocomplete-search", undefined, "", undefined, { resource: "WORKER" }, 20]),
    ).toBe(false);
    expect(matches(["select-option", "WORKER", "wrk_1", null])).toBe(false);
  });

  // A resource name that turns up in an unrelated query's key is not a reason
  // to refetch it; only the three select-option scopes are ours to invalidate.
  it("leaves unrelated queries alone even when they name the resource", () => {
    expect(matches(["pto-policy-list", "PTO_POLICY"])).toBe(false);
    expect(matches([])).toBe(false);
  });

  it("matches a filtered picker, where the filters ride alongside the resource", () => {
    expect(
      matches([
        "autocomplete-search",
        undefined,
        "std",
        undefined,
        { resource: "PTO_POLICY", filters: { membersOnly: true } },
        20,
      ]),
    ).toBe(true);
  });
});
