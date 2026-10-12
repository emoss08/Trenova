import { describe, expect, it } from "vitest";

import {
  filterMentionOptions,
  getMentionState,
  resolveEntityAlias,
  resolveMentionCommit,
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

  it("opens a mention only at the start of a word, so an email address is a search", () => {
    expect(getMentionState("dispatch@acme").mentionOpen).toBe(false);
    expect(getMentionState("late @lo")).toEqual({
      activeFilter: null,
      mentionOpen: true,
      mentionText: "lo",
    });
  });

  it("reaches pages, commands and the loads and drivers people call them", () => {
    expect(resolveEntityAlias("load")).toBe("shipment");
    expect(resolveEntityAlias("driver")).toBe("worker");
    expect(resolveEntityAlias("doc")).toBe("document");
    expect(resolveEntityAlias("pages")).toBe("pages");
    expect(resolveEntityAlias("cmd")).toBe("commands");
    expect(filterMentionOptions("p").map((option) => option.key)).toEqual(["pages"]);
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

  it("commits a mention to the only type its prefix can still mean", () => {
    expect(resolveMentionCommit("work")).toBe("worker");
    expect(resolveMentionCommit("cust")).toBe("customer");
    expect(resolveMentionCommit("docu")).toBe("document");
  });

  it("commits nothing for an empty, unknown or ambiguous prefix", () => {
    expect(resolveMentionCommit("")).toBeNull();
    expect(resolveMentionCommit("zzz")).toBeNull();
  });
});
