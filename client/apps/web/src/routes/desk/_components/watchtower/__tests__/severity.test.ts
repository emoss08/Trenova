import { isUnseen, SEVERITY_TONE, toggleFilter } from "../severity";
import { describe, expect, it } from "vitest";

describe("SEVERITY_TONE", () => {
  // Severity is an ordering, so it is a tone rather than a categorical
  // accent: a reader scanning the feed is sorting by loudness whether they
  // mean to or not.
  it("maps each severity to a tone with the same ordering", () => {
    expect(SEVERITY_TONE.Info).toBe("info");
    expect(SEVERITY_TONE.Warning).toBe("warning");
    expect(SEVERITY_TONE.Critical).toBe("danger");
  });
});

describe("toggleFilter", () => {
  it("turns a filter on", () => {
    expect(toggleFilter([], "Critical")).toEqual(["Critical"]);
  });

  it("turns one off without disturbing the rest", () => {
    expect(toggleFilter(["Critical", "Warning"], "Critical")).toEqual(["Warning"]);
  });

  // An empty set means "everything" to the server, so turning the last chip
  // off asks for the whole feed rather than for nothing. A reader who
  // deselects their way back to the start should not be staring at a blank
  // list.
  it("comes back to everything when the last one is turned off", () => {
    expect(toggleFilter(["Critical"], "Critical")).toEqual([]);
  });

  it("does not mutate the set it was given", () => {
    const before = ["Critical"];
    toggleFilter(before, "Warning");

    expect(before).toEqual(["Critical"]);
  });
});

describe("isUnseen", () => {
  it("counts what arrived after the reader's cursor", () => {
    expect(isUnseen(200, 100)).toBe(true);
    expect(isUnseen(100, 100)).toBe(false);
    expect(isUnseen(50, 100)).toBe(false);
  });

  // A reader who has never opened the feed has a cursor of zero, so
  // everything is new to them rather than nothing.
  it("treats everything as new for a reader who has never looked", () => {
    expect(isUnseen(1, 0)).toBe(true);
  });
});
