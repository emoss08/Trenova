import { describe, expect, it } from "vitest";
import { acknowledgeableEventIds } from "../use-event-actions";
import { buildInboxQueryVariables, hasInboxFilters, type InboxFilterState } from "../inbox-filters";

const BASE: InboxFilterState = {
  scope: "attention",
  q: "",
  severity: [],
  category: [],
  source: [],
  carrier: null,
};

describe("buildInboxQueryVariables", () => {
  it("maps each scope to the statuses it covers", () => {
    expect(buildInboxQueryVariables(BASE).filter).toEqual({ statuses: ["Open"] });
    expect(buildInboxQueryVariables({ ...BASE, scope: "acknowledged" }).filter).toEqual({
      statuses: ["Acknowledged"],
    });
    expect(buildInboxQueryVariables({ ...BASE, scope: "resolved" }).filter).toEqual({
      statuses: ["Resolved", "Dismissed"],
    });
    expect(buildInboxQueryVariables({ ...BASE, scope: "all" }).filter).toEqual({});
  });

  it("sends severity, category and carrier through the event filter and source as a field filter", () => {
    expect(
      buildInboxQueryVariables({
        ...BASE,
        q: "  blue ridge ",
        severity: ["Critical", "High"],
        category: ["Insurance"],
        source: ["NativeChangeFeed"],
        carrier: "car_1",
      }),
    ).toEqual({
      filter: {
        statuses: ["Open"],
        severities: ["Critical", "High"],
        categories: ["Insurance"],
        carrierId: "car_1",
      },
      query: "blue ridge",
      fieldFilters: [{ field: "source", operator: "in", value: ["NativeChangeFeed"] }],
    });
  });

  it("sends no search and no field filters when nothing narrows the list", () => {
    expect(buildInboxQueryVariables({ ...BASE, q: "   " })).toEqual({
      filter: { statuses: ["Open"] },
      query: null,
      fieldFilters: [],
    });
  });

  it("does not count the scope as a filter", () => {
    expect(hasInboxFilters({ ...BASE, scope: "all" })).toBe(false);
    expect(hasInboxFilters({ ...BASE, source: ["Override"] })).toBe(true);
    expect(hasInboxFilters({ ...BASE, q: "x" })).toBe(true);
  });
});

describe("acknowledgeableEventIds", () => {
  it("keeps only open events", () => {
    expect(
      acknowledgeableEventIds([
        { id: "1", status: "Open" },
        { id: "2", status: "Acknowledged" },
        { id: "3", status: "Resolved" },
        { id: "4", status: "Open" },
        { id: "5", status: "Dismissed" },
      ]),
    ).toEqual(["1", "4"]);
  });
});
