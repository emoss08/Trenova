import { describe, expect, it } from "vitest";

import {
  filterMentionOptions,
  getMentionState,
  resolveEntityAlias,
  stripMentionToken,
} from "./search-entity-filter";

describe("search entity filter helpers", () => {
  it("detects an exact worker mention", () => {
    expect(getMentionState("@workers")).toEqual({
      activeFilter: "worker",
      mentionOpen: true,
      mentionText: "workers",
    });
  });

  it("does not keep mention mode open after free text begins", () => {
    expect(getMentionState("@workers sam")).toEqual({
      activeFilter: null,
      mentionOpen: false,
      mentionText: "",
    });
  });

  it("filters mention options by aliases", () => {
    expect(filterMentionOptions("cust").map((option) => option.key)).toEqual(["customer"]);
  });

  it("strips the trailing mention token from the query", () => {
    expect(stripMentionToken("sam @workers")).toBe("sam");
  });

  it("resolves a committed alias to a canonical entity type", () => {
    expect(resolveEntityAlias("customers")).toBe("customer");
  });
});
