import { describe, expect, it } from "vitest";
import { formulaValueSourceSchema } from "../formula-template";

describe("formulaValueSourceSchema", () => {
  it("accepts values the engine fed from an external source", () => {
    expect(formulaValueSourceSchema.parse("provided")).toBe("provided");
  });
});
