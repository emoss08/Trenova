import type { FieldChange } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { describeChangedFields, reviewLinkFor } from "../review-diff";

const change = (from: unknown, to: unknown): FieldChange => ({
  from,
  to,
  type: "updated",
  fieldType: "string",
  path: "",
});

describe("describeChangedFields", () => {
  it("leaves the expression to the diff viewer and labels everything else", () => {
    const rows = describeChangedFields({
      expression: change("a", "b"),
      minCharge: change(null, "250"),
      roundingPrecision: change(2, 0),
      "variableDefinitions.0.defaultValue": change(18, 20),
    });

    expect(rows.map((row) => row.path)).toEqual([
      "minCharge",
      "roundingPrecision",
      "variableDefinitions.0.defaultValue",
    ]);
    expect(rows[0]?.label).toBe("Minimum charge");
    expect(rows[0]?.summary).toBe("empty → 250");
    expect(rows[1]?.summary).toBe("2 → 0");
    expect(rows[2]?.label).toBe("Variable 1 default value");
  });

  it("returns nothing when only the expression changed", () => {
    expect(describeChangedFields({ expression: change("a", "b") })).toEqual([]);
  });
});

describe("reviewLinkFor", () => {
  it("opens the studio on the approve step", () => {
    expect(reviewLinkFor("ft_1")).toBe(
      "/billing/configuration-files/formula-templates/ft_1/edit?review=approve",
    );
  });
});
