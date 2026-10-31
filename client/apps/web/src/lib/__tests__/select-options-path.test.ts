import { describe, expect, it } from "vitest";
import { selectOptionsPath } from "../select-options-path";

describe("selectOptionsPath", () => {
  it("adds paging, filters and each id to the link", () => {
    expect(
      selectOptionsPath(
        "/documents/select-options/",
        { limit: "10", offset: "0" },
        { resourceId: "shp_1", status: ["Active", "Draft"], skipped: undefined },
        ["doc_1", "doc_2"],
      ),
    ).toBe(
      "/documents/select-options/?limit=10&offset=0&resourceId=shp_1&status=Active&status=Draft&ids=doc_1&ids=doc_2",
    );
  });

  it("leaves a link with nothing to add as it is", () => {
    expect(selectOptionsPath("/users/select-options/", {})).toBe("/users/select-options/");
  });
});
