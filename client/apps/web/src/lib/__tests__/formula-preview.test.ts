import { testExpressionResponseSchema } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { previewFailure, previewWarnings, unguardedFields } from "../formula-preview";

const nullable = (scope: string, name: string) => ({
  scope,
  from: 0,
  to: name.length,
  severity: "warning",
  code: "nullable",
  message:
    "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: {1}",
  args: [`\`${name}\``, `coalesce(${name}, 0)`],
  name,
  fixes: [],
});

const unknown = {
  scope: "breakdowns[0].expression",
  from: 0,
  to: 3,
  severity: "error",
  code: "unknown-variable",
  message:
    "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
  args: ["`foo`"],
  name: "foo",
  fixes: [],
};

const missing = {
  scope: "expression",
  code: "missingField",
  message:
    "no value for {0} on this record; write {1} so the formula still prices when it is missing",
  args: ["weight", "coalesce(weight, 0)"],
  from: 0,
  to: 6,
  fields: ["weight"],
  guards: ["coalesce(weight, 0)"],
};

describe("the preview's findings", () => {
  it("reads the server's answer with every field it may leave out", () => {
    expect(testExpressionResponseSchema.parse({ valid: true, result: "125.5" })).toMatchObject({
      valid: true,
      result: "125.5",
      diagnostics: [],
    });
  });

  it("explains a failure by the evaluation error first, then by the first check error", () => {
    const ran = testExpressionResponseSchema.parse({
      valid: false,
      diagnostics: [nullable("expression", "weight"), unknown],
      error: missing,
    });
    expect(previewFailure(ran)).toEqual({
      ...missing,
      fields: ["weight"],
      guards: ["coalesce(weight, 0)"],
    });

    const refused = testExpressionResponseSchema.parse({
      valid: false,
      diagnostics: [nullable("expression", "weight"), unknown],
      error: null,
    });
    expect(previewFailure(refused)).toMatchObject({ code: "unknown-variable", name: "foo" });
    expect(previewFailure(testExpressionResponseSchema.parse({ valid: true }))).toBeNull();
    expect(previewFailure(undefined)).toBeNull();
  });

  it("lists warnings on every scope or on one, and the unguarded fields once each", () => {
    const result = testExpressionResponseSchema.parse({
      valid: true,
      diagnostics: [
        nullable("expression", "weight"),
        nullable("breakdowns[1].expression", "weight"),
        nullable("breakdowns[1].expression", "pieces"),
        unknown,
      ],
    });

    expect(previewWarnings(result).map((warning) => warning.scope)).toEqual([
      "expression",
      "breakdowns[1].expression",
      "breakdowns[1].expression",
    ]);
    expect(previewWarnings(result, "expression")).toHaveLength(1);
    expect(unguardedFields(result)).toEqual(["weight", "pieces"]);
  });
});
