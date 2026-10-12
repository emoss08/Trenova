import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  joinSteps,
  memorySource,
  memoryWhy,
  savedLine,
  scopeLabel,
  scopeWord,
  splitSteps,
  usageLine,
} from "../memory-format";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d)\}/g, (_match, index: string) => String(args[Number(index)]))) as TranslateFn;

// 2026-10-03 15:00 UTC
const NOW = 1_791_039_600;
const DAY = 86_400;

describe("memory lines", () => {
  it("says where a memory came from", () => {
    expect(
      savedLine(
        {
          source: "Agent",
          sourceTitle: "Acme rate review",
          scope: "Role",
          createdAt: NOW - 15 * DAY,
        },
        "UTC",
        t,
        NOW,
      ),
    ).toBe("Saved Sep 18 from “Acme rate review”");
    expect(memorySource({ source: "User", sourceTitle: "", scope: "User" }, t)).toBe(
      "Added by you",
    );
    expect(
      savedLine({ source: "User", sourceTitle: "", scope: "User", createdAt: NOW }, "UTC", t, NOW),
    ).toBe("Saved Today from “Added by you”");
  });

  it("says how much it has been used, or that it is paused", () => {
    expect(usageLine({ status: "Active", useCount: 14, lastUsedAt: NOW - 60 }, "UTC", t, NOW)).toBe(
      "Used 14× · last today",
    );
    expect(usageLine({ status: "Active", useCount: 9, lastUsedAt: NOW - DAY }, "UTC", t, NOW)).toBe(
      "Used 9× · last yesterday",
    );
    expect(
      usageLine({ status: "Active", useCount: 11, lastUsedAt: NOW - 2 * DAY }, "UTC", t, NOW),
    ).toBe("Used 11× · last Oct 1");
    expect(usageLine({ status: "Paused", useCount: 11, lastUsedAt: NOW }, "UTC", t, NOW)).toBe(
      "Paused",
    );
    expect(usageLine({ status: "Active", useCount: 0, lastUsedAt: null }, "UTC", t, NOW)).toBe(
      "Not used yet",
    );
  });

  it("names a role scope by the role", () => {
    expect(scopeLabel({ scope: "Role", roleName: "Billing" }, t)).toBe("Billing");
    expect(scopeLabel({ scope: "User" }, t)).toBe("Just you");
    expect(scopeLabel({ scope: "Organization" }, t)).toBe("Organization");
  });
});

describe("learned memories", () => {
  it("names a lesson an agent kept and who it is kept for", () => {
    expect(memorySource({ source: "Reflection", sourceTitle: "", scope: "Agent" }, t)).toBe(
      "An agent looking back over its work",
    );
    expect(scopeLabel({ scope: "Agent" }, t)).toBe("Everyone using this agent");
  });
});

/*
The server sends reason, quotes, replaces and replacedBy on DeskMemory
(services/tms/internal/api/graphql/schema/desk_memory.graphqls). A memory a
person wrote has an empty reason, no quotes and no links, and shows nothing.
*/
describe("memoryWhy", () => {
  it("shows nothing for a memory a person wrote down", () => {
    expect(memoryWhy({ reason: "", quotes: [], replaces: null, replacedBy: null }, t)).toEqual([]);
    expect(memoryWhy({}, t)).toEqual([]);
    expect(memoryWhy({ reason: "   ", quotes: ["  "] }, t)).toEqual([]);
  });

  it("gives the reason, a little of what was said, and what it replaced", () => {
    expect(
      memoryWhy(
        {
          reason: " Billing asked to be copied as well. ",
          quotes: ["Billing needs these too", "And the rep", "A third quote"],
          replaces: { content: "Copy dispatch on rate confirmations." },
          replacedBy: { content: "Copy dispatch, billing and the rep." },
        },
        t,
      ),
    ).toEqual([
      { key: "reason", label: "Why", text: "Billing asked to be copied as well." },
      { key: "quote", label: "Said", text: "“Billing needs these too”" },
      { key: "quote", label: "Said", text: "“And the rep”" },
      { key: "replaces", label: "Replaces", text: "“Copy dispatch on rate confirmations.”" },
      { key: "replacedBy", label: "Replaced by", text: "“Copy dispatch, billing and the rep.”" },
    ]);
  });
});

describe("scopeWord", () => {
  it("names who a memory is for in a word, capitalised at the start of a line", () => {
    expect(scopeWord({ scope: "User" }, t)).toBe("just you");
    expect(scopeWord({ scope: "User" }, t, true)).toBe("Just you");
    expect(scopeWord({ scope: "Agent" }, t)).toBe("everyone");
    expect(scopeWord({ scope: "Agent" }, t, true)).toBe("Everyone");
    expect(scopeWord({ scope: "Role", roleName: "Dispatch" }, t)).toBe("Dispatch");
    expect(scopeWord({ scope: "Role", roleName: "Dispatch" }, t, true)).toBe("Dispatch");
    expect(scopeWord({ scope: "Role", roleName: null }, t)).toBe("your team");
    expect(scopeWord({ scope: "Organization" }, t)).toBe("the whole organization");
  });
});

describe("splitSteps", () => {
  it("splits numbered steps written on one line", () => {
    expect(
      splitSteps("1. If `search_shipments` finds nothing, keep looking. 2. Call `list_shipments`."),
    ).toEqual(["If `search_shipments` finds nothing, keep looking.", "Call `list_shipments`."]);
  });

  it("splits on lines when the memory is written on lines, dropping their numbers", () => {
    expect(splitSteps("1. Read the move\n\n2. Assign it\nConfirm")).toEqual([
      "Read the move",
      "Assign it",
      "Confirm",
    ]);
  });

  it("returns text that does not split as one entry, and nothing for empty text", () => {
    expect(splitSteps("Acme pays on the 15th.")).toEqual(["Acme pays on the 15th."]);
    expect(splitSteps("   ")).toEqual([]);
  });
});

describe("joinSteps", () => {
  it("writes edited steps back in the memory's own form", () => {
    expect(joinSteps(["Read it", "Assign it"], "1. Read. 2. Assign.")).toBe(
      "1. Read it 2. Assign it",
    );
    expect(joinSteps(["Read it", " ", "Assign it"], "Read.\nAssign.")).toBe("Read it\nAssign it");
    expect(joinSteps(["Acme pays on the 15th."], "Acme pays late.")).toBe("Acme pays on the 15th.");
  });
});
