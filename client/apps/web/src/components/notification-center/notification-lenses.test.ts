import { describe, expect, it } from "vitest";
import { lensCount } from "./notification-lenses";

const counts = { forYou: 9, forYouUnread: 6, team: 26, teamUnread: 20, later: 2 };

describe("lensCount", () => {
  it("shows each counted part's own number", () => {
    expect(lensCount(counts, "forYou")).toBe("9");
    expect(lensCount(counts, "team")).toBe("26");
    expect(lensCount(counts, "later")).toBe("2");
  });

  it("shows a zero rather than hiding it, so an empty part reads as empty", () => {
    expect(lensCount({ ...counts, later: 0 }, "later")).toBe("0");
  });

  it("does not count Done, which only grows", () => {
    expect(lensCount(counts, "done")).toBeNull();
  });

  it("shows nothing until the counts have loaded", () => {
    expect(lensCount(undefined, "forYou")).toBeNull();
  });

  it("shows 999+ once a count reaches the 1,000 the server stops counting at", () => {
    expect(lensCount({ ...counts, team: 999 }, "team")).toBe("999");
    expect(lensCount({ ...counts, team: 1000 }, "team")).toBe("999+");
  });
});
