import { mapFieldErrorPaths } from "@/lib/form-errors";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import type { FieldPath } from "@trenova/shared/types/errors";
import { describe, expect, it } from "vitest";

const refusal = () =>
  new GraphQLRequestError({
    kind: "graphql",
    message: "Validation failed",
    status: 422,
    retryAfter: 30,
    graphQLErrors: [
      {
        message: "Validation failed",
        type: "validation-error",
        extensions: { code: "VALIDATION" },
        errors: [
          { path: ["rows", 0, "name"], message: "Name is required", code: "REQUIRED" },
          "not a field error",
          { path: "rows.1.name", message: "Path is not a list", code: "INVALID" },
          { path: ["total"], message: "Total must be positive", code: "INVALID" },
        ],
      },
      { message: "Something else", type: "internal-error", extensions: {} },
    ],
  });

const shiftRows = (path: FieldPath): FieldPath =>
  path[0] === "rows" && typeof path[1] === "number"
    ? ["lines", path[1] + 3, ...path.slice(2)]
    : path;

describe("mapFieldErrorPaths", () => {
  it("passes every field error's path through the map and keeps the rest of the refusal", () => {
    const mapped = mapFieldErrorPaths(refusal(), shiftRows) as GraphQLRequestError;

    expect(mapped).toBeInstanceOf(GraphQLRequestError);
    expect(mapped.kind).toBe("graphql");
    expect(mapped.message).toBe("Validation failed");
    expect(mapped.status).toBe(422);
    expect(mapped.retryAfter).toBe(30);
    expect(mapped.getFieldErrors().map((error) => [error.field, error.message])).toEqual([
      ["lines.3.name", "Name is required"],
      ["total", "Total must be positive"],
    ]);
    expect(mapped.graphQLErrors).toEqual([
      {
        message: "Validation failed",
        type: "validation-error",
        extensions: {
          code: "VALIDATION",
          errors: [
            { path: ["lines", 3, "name"], message: "Name is required", code: "REQUIRED" },
            "not a field error",
            { path: "rows.1.name", message: "Path is not a list", code: "INVALID" },
            { path: ["total"], message: "Total must be positive", code: "INVALID" },
          ],
        },
        errors: [
          { path: ["lines", 3, "name"], message: "Name is required", code: "REQUIRED" },
          "not a field error",
          { path: "rows.1.name", message: "Path is not a list", code: "INVALID" },
          { path: ["total"], message: "Total must be positive", code: "INVALID" },
        ],
      },
      { message: "Something else", type: "internal-error", extensions: {} },
    ]);
  });

  it("leaves the refusal it was given untouched", () => {
    const original = refusal();

    mapFieldErrorPaths(original, shiftRows);

    expect(original.getFieldErrors().map((error) => error.field)).toEqual(["rows.0.name", "total"]);
  });

  it("returns a refusal with no field errors as it is", () => {
    const failure = new GraphQLRequestError({
      kind: "graphql",
      message: "Not found",
      status: 200,
      graphQLErrors: [{ message: "Not found", type: "resource-not-found", extensions: {} }],
    });

    expect(mapFieldErrorPaths(failure, shiftRows)).toBe(failure);
  });

  it("returns any other failure as it is", () => {
    const failure = new Error("offline");

    expect(mapFieldErrorPaths(failure, shiftRows)).toBe(failure);
    expect(mapFieldErrorPaths("refused", shiftRows)).toBe("refused");
  });
});
