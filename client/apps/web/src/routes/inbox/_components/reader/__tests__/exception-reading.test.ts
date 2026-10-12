import { describe, expect, it } from "vitest";
import { readException } from "../exception-reading";

describe("readException", () => {
  it("reads the count, the unit and the kind from the words Nova read", () => {
    expect(readException("2 pallets short")).toEqual({ kind: "Shortage", quantity: 2, unit: "pallets" });
    expect(readException("1 case over")).toEqual({ kind: "Overage", quantity: 1, unit: "case" });
    expect(readException("3 cartons damaged in transit")).toEqual({
      kind: "Damage",
      quantity: 3,
      unit: "cartons",
    });
    expect(readException("Short 4 pieces")).toEqual({ kind: "Shortage", quantity: 4, unit: "pieces" });
  });

  it("keeps what it can read and leaves the rest to the person", () => {
    expect(readException("pallets short")).toEqual({ kind: "Shortage", quantity: null, unit: "" });
    expect(readException("12 short")).toEqual({ kind: "Shortage", quantity: 12, unit: "" });
    expect(readException("Signature present")).toEqual({ kind: null, quantity: null, unit: "" });
    expect(readException("")).toEqual({ kind: null, quantity: null, unit: "" });
  });
});
