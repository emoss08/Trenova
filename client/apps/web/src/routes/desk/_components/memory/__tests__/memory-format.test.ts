import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { memorySource, memoryWhy, savedLine, scopeLabel, usageLine } from "../memory-format";

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
