import { describe, expect, it } from "vitest";

import {
  filterMentionOptions,
  getMentionState,
  resolveEntityAlias,
  resolveMentionCommit,
  searchEntityOptions,
  stripMentionToken,
  type AvailableScopes,
} from "./search-entity-filter";

/** Every scope, for a person who may read every kind of record. */
const EVERY: AvailableScopes = new Set(searchEntityOptions.map((option) => option.key));

/** A person who may not read invoices. */
const NO_INVOICES: AvailableScopes = new Set(
  searchEntityOptions.map((option) => option.key).filter((key) => key !== "invoice"),
);

describe("search entity filter helpers", () => {
  it("detects an exact worker mention", () => {
    expect(getMentionState("@workers", EVERY)).toEqual({
      activeFilter: "worker",
      mentionOpen: true,
      mentionText: "workers",
    });
  });

  it("does not keep mention mode open after free text begins", () => {
    expect(getMentionState("@workers sam", EVERY)).toEqual({
      activeFilter: null,
      mentionOpen: false,
      mentionText: "",
    });
  });

  it("opens a mention only at the start of a word, so an email address is a search", () => {
    expect(getMentionState("dispatch@acme", EVERY).mentionOpen).toBe(false);
    expect(getMentionState("late @lo", EVERY)).toEqual({
      activeFilter: null,
      mentionOpen: true,
      mentionText: "lo",
    });
  });

  it("reaches pages, commands and the loads and drivers people call them", () => {
    expect(resolveEntityAlias("load", EVERY)).toBe("shipment");
    expect(resolveEntityAlias("driver", EVERY)).toBe("worker");
    expect(resolveEntityAlias("doc", EVERY)).toBe("document");
    expect(resolveEntityAlias("pages", EVERY)).toBe("pages");
    expect(resolveEntityAlias("cmd", EVERY)).toBe("commands");
    expect(filterMentionOptions("p", EVERY).map((option) => option.key)).toEqual(["pages"]);
  });

  it("filters mention options by aliases", () => {
    expect(filterMentionOptions("cust", EVERY).map((option) => option.key)).toEqual(["customer"]);
  });

  it("strips the trailing mention token from the query", () => {
    expect(stripMentionToken("sam @workers")).toBe("sam");
  });

  it("resolves a committed alias to a canonical entity type", () => {
    expect(resolveEntityAlias("customers", EVERY)).toBe("customer");
  });

  it("commits a mention to the only type its prefix can still mean", () => {
    expect(resolveMentionCommit("work", EVERY)).toBe("worker");
    expect(resolveMentionCommit("cust", EVERY)).toBe("customer");
    expect(resolveMentionCommit("docu", EVERY)).toBe("document");
  });

  it("commits nothing for an empty, unknown or ambiguous prefix", () => {
    expect(resolveMentionCommit("", EVERY)).toBeNull();
    expect(resolveMentionCommit("zzz", EVERY)).toBeNull();
  });

  it("reaches invoices by what billers call them", () => {
    expect(resolveEntityAlias("inv", EVERY)).toBe("invoice");
    expect(resolveEntityAlias("bills", EVERY)).toBe("invoice");
    expect(resolveMentionCommit("invo", EVERY)).toBe("invoice");
  });

  it("never offers or settles on a kind of record the person may not read", () => {
    expect(filterMentionOptions("", NO_INVOICES).map((option) => option.key)).not.toContain(
      "invoice",
    );
    expect(filterMentionOptions("inv", NO_INVOICES)).toEqual([]);
    expect(resolveEntityAlias("invoices", NO_INVOICES)).toBeNull();
    expect(resolveMentionCommit("inv", NO_INVOICES)).toBeNull();
    expect(getMentionState("@invoices", NO_INVOICES).activeFilter).toBeNull();
  });
});
