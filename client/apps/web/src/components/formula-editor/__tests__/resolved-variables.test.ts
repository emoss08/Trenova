import { describe, expect, it } from "vitest";
import { flattenResolvedVariables } from "../resolved-variables";

describe("flattenResolvedVariables", () => {
  it("flattens nested objects into dotted keys and keeps scalars", () => {
    expect(
      flattenResolvedVariables({
        totalDistance: 500,
        hasHazmat: false,
        customer: { name: "Acme", code: "ACME", address: { state: "GA" } },
      }),
    ).toEqual({
      totalDistance: 500,
      hasHazmat: false,
      "customer.name": "Acme",
      "customer.code": "ACME",
      "customer.address.state": "GA",
    });
  });

  it("drops nulls, arrays, and functions", () => {
    expect(
      flattenResolvedVariables({
        weight: null,
        stops: [{ sequence: 1 }],
        lookup: () => 0,
        pieces: 3,
      }),
    ).toEqual({ pieces: 3 });
  });
});
