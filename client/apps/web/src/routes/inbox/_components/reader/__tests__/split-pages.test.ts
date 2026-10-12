import { describe, expect, it } from "vitest";
import { splitRanges } from "../split-pages";

describe("splitRanges", () => {
  it("files each run of pages of one type as one document", () => {
    expect(splitRanges([1, 2, 3, 4], ["rc", "rc", "bol", "rc"])).toEqual([
      { fromPage: 1, toPage: 2, documentTypeId: "rc" },
      { fromPage: 3, toPage: 3, documentTypeId: "bol" },
      { fromPage: 4, toPage: 4, documentTypeId: "rc" },
    ]);
  });

  it("leaves out pages marked not to file, and does not join runs across them", () => {
    expect(splitRanges([1, 2, 3], ["rc", null, "rc"])).toEqual([
      { fromPage: 1, toPage: 1, documentTypeId: "rc" },
      { fromPage: 3, toPage: 3, documentTypeId: "rc" },
    ]);
    expect(splitRanges([1, 2], [null, null])).toEqual([]);
  });

  it("does not join pages whose numbers are not next to each other", () => {
    expect(splitRanges([1, 3], ["rc", "rc"])).toEqual([
      { fromPage: 1, toPage: 1, documentTypeId: "rc" },
      { fromPage: 3, toPage: 3, documentTypeId: "rc" },
    ]);
  });
});
