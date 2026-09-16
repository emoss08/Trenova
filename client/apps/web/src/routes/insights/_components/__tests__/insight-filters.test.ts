import { describe, expect, it } from "vitest";
import {
  DEFAULT_FILTERS,
  hasActiveFilters,
  pageCount,
  parseFilters,
  serializeFilters,
  setStatusFilter,
  statusesFor,
  toggleFilter,
  type InsightFilterState,
} from "../insight-filters";

function filters(overrides: Partial<InsightFilterState> = {}): InsightFilterState {
  return { ...DEFAULT_FILTERS, ...overrides };
}

describe("statusesFor", () => {
  it("asks for what the reader chose", () => {
    expect(statusesFor("active")).toEqual(["Active"]);
    expect(statusesFor("dismissed")).toEqual(["Dismissed"]);
  });

  // Resolved and Superseded are different machinery — the condition went away
  // versus fresher numbers replaced it — but from the reader's side both mean
  // nobody is being asked to act any more.
  it("treats both ways a finding stops being made as closed", () => {
    expect(statusesFor("closed")).toEqual(["Resolved", "Superseded"]);
  });
});

describe("parseFilters", () => {
  it("reads a filtered view out of the URL", () => {
    const parsed = parseFilters(
      new URLSearchParams("status=dismissed&category=CashFlow&severity=Critical&page=3"),
    );

    expect(parsed).toEqual({
      status: "dismissed",
      categories: ["CashFlow"],
      severities: ["Critical"],
      page: 3,
      selected: null,
    });
  });

  it("reads several values for one filter", () => {
    const parsed = parseFilters(new URLSearchParams("category=CashFlow&category=Compliance"));

    expect(parsed.categories).toEqual(["CashFlow", "Compliance"]);
  });

  it("falls back to the defaults for an empty URL", () => {
    expect(parseFilters(new URLSearchParams())).toEqual(DEFAULT_FILTERS);
  });

  // A hand-edited or stale link should open the page showing something rather
  // than erroring, and a value this build has not heard of narrows nothing.
  it("drops values it does not recognise rather than failing", () => {
    const parsed = parseFilters(
      new URLSearchParams("status=nonsense&category=Wibble&category=CashFlow&severity=Loud"),
    );

    expect(parsed.status).toBe("active");
    expect(parsed.categories).toEqual(["CashFlow"]);
    expect(parsed.severities).toEqual([]);
  });

  // A repeated value in a hand-edited URL must not produce a duplicated filter
  // chip or send the same value twice to the API.
  it("keeps a repeated value once", () => {
    const parsed = parseFilters(new URLSearchParams("category=CashFlow&category=CashFlow"));

    expect(parsed.categories).toEqual(["CashFlow"]);
  });

  it("refuses a page number that is not a positive whole number", () => {
    expect(parseFilters(new URLSearchParams("page=0")).page).toBe(1);
    expect(parseFilters(new URLSearchParams("page=-4")).page).toBe(1);
    expect(parseFilters(new URLSearchParams("page=2.5")).page).toBe(1);
    expect(parseFilters(new URLSearchParams("page=banana")).page).toBe(1);
  });
});

describe("parseFilters selection", () => {
  it("reads which finding is open", () => {
    expect(parseFilters(new URLSearchParams("selected=inst_1")).selected).toBe("inst_1");
  });

  it("has nothing open by default", () => {
    expect(parseFilters(new URLSearchParams()).selected).toBeNull();
  });
});

describe("serializeFilters", () => {
  it("round-trips a filtered view", () => {
    const original = filters({
      status: "closed",
      categories: ["CashFlow", "Compliance"],
      severities: ["Warning"],
      page: 2,
    });

    expect(parseFilters(serializeFilters(original))).toEqual(original);
  });

  // A URL spelling out every default is unreadable and makes an unfiltered page
  // look filtered.
  it("writes nothing for a view nobody has narrowed", () => {
    expect(serializeFilters(DEFAULT_FILTERS).toString()).toBe("");
  });

  // An opened card is part of the view, so a link to it reopens it.
  it("round-trips an opened finding", () => {
    const opened = filters({ selected: "inst_1" });

    expect(parseFilters(serializeFilters(opened)).selected).toBe("inst_1");
  });

  it("omits the first page", () => {
    expect(serializeFilters(filters({ page: 1 })).has("page")).toBe(false);
    expect(serializeFilters(filters({ page: 2 })).get("page")).toBe("2");
  });
});

describe("toggleFilter", () => {
  it("adds a value that was not selected", () => {
    const next = toggleFilter(DEFAULT_FILTERS, "categories", "CashFlow");

    expect(next.categories).toEqual(["CashFlow"]);
  });

  it("removes a value that was selected", () => {
    const next = toggleFilter(
      filters({ categories: ["CashFlow", "Compliance"] }),
      "categories",
      "CashFlow",
    );

    expect(next.categories).toEqual(["Compliance"]);
  });

  // Staying on page four while narrowing the results is how someone lands on an
  // empty page and concludes there is nothing to see.
  it("returns to the first page", () => {
    const next = toggleFilter(filters({ page: 4 }), "severities", "Critical");

    expect(next.page).toBe(1);
  });

  it("does not mutate the state it was given", () => {
    const original = filters({ categories: ["CashFlow"] });

    toggleFilter(original, "categories", "Compliance");

    expect(original.categories).toEqual(["CashFlow"]);
  });

  // The open card may not survive the narrowing, and leaving a panel showing a
  // finding that is no longer in the list behind it is disorienting.
  it("closes whatever was open", () => {
    const next = toggleFilter(filters({ selected: "inst_1" }), "categories", "CashFlow");

    expect(next.selected).toBeNull();
  });
});

describe("setStatusFilter", () => {
  it("changes the lifecycle and returns to the first page", () => {
    const next = setStatusFilter(filters({ page: 3 }), "dismissed");

    expect(next.status).toBe("dismissed");
    expect(next.page).toBe(1);
  });

  it("leaves the other filters alone", () => {
    const next = setStatusFilter(filters({ categories: ["CashFlow"] }), "closed");

    expect(next.categories).toEqual(["CashFlow"]);
  });

  it("closes whatever was open", () => {
    expect(setStatusFilter(filters({ selected: "inst_1" }), "closed").selected).toBeNull();
  });
});

describe("hasActiveFilters", () => {
  it("is false for the view as it opens", () => {
    expect(hasActiveFilters(DEFAULT_FILTERS)).toBe(false);
  });

  it("is true once anything has been narrowed", () => {
    expect(hasActiveFilters(filters({ status: "dismissed" }))).toBe(true);
    expect(hasActiveFilters(filters({ categories: ["CashFlow"] }))).toBe(true);
    expect(hasActiveFilters(filters({ severities: ["Info"] }))).toBe(true);
  });

  // Paging is not filtering. Offering "clear filters" on page two of an
  // unfiltered list would suggest something is hidden when nothing is.
  it("is not triggered by paging alone", () => {
    expect(hasActiveFilters(filters({ page: 5 }))).toBe(false);
  });

  // Opening a card narrows nothing, so it must not offer to clear filters.
  it("is not triggered by opening a finding", () => {
    expect(hasActiveFilters(filters({ selected: "inst_1" }))).toBe(false);
  });
});

describe("pageCount", () => {
  it("counts the pages a total spans", () => {
    expect(pageCount(50, 25)).toBe(2);
    expect(pageCount(51, 25)).toBe(3);
  });

  // An empty list is one empty page, not zero pages: the pager would otherwise
  // render "page 1 of 0".
  it("is one page when there is nothing to show", () => {
    expect(pageCount(0, 25)).toBe(1);
  });

  it("does not divide by a page size of zero", () => {
    expect(pageCount(10, 0)).toBe(1);
  });
});
