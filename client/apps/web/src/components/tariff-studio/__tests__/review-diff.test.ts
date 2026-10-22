import type { FieldChange } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { describeChangedFields } from "../review-diff";

const change = (from: unknown, to: unknown): FieldChange => ({
  from,
  to,
  type: "updated",
  fieldType: "string",
  path: "",
});

describe("describeChangedFields", () => {
  it("leaves the source to the diff and labels the rest, a pinned library by its place", () => {
    const rows = describeChangedFields({
      source: change("tariff \"a\"", "tariff \"b\""),
      description: change("", "Per mile"),
      "imports.0.version": change("flv_1", "flv_2"),
    });

    expect(rows).toEqual([
      { path: "description", label: "Description", summary: "empty → Per mile" },
      { path: "imports.0.version", label: "Pinned library 1 version", summary: "flv_1 → flv_2" },
    ]);
  });

  it("returns nothing when only the source changed", () => {
    expect(describeChangedFields({ source: change("a", "b") })).toEqual([]);
  });
});
