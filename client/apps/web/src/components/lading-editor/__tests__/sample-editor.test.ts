import { describe, expect, it } from "vitest";
import { x12Decorations } from "../sample-editor";

function marks(text: string): [number, number, string][] {
  const found: [number, number, string][] = [];
  x12Decorations(text).between(0, text.length, (from, to, value) => {
    found.push([from, to, value.spec.class as string]);
  });
  return found;
}

describe("x12Decorations", () => {
  it("marks an X12 sample's segment IDs, separators and numbers in the syntax colours", () => {
    expect(marks("ST*204~B2**TRNV~")).toEqual([
      [0, 2, "cm-syntax-meta"],
      [2, 3, "cm-syntax-operator"],
      [3, 6, "cm-syntax-number"],
      [6, 7, "cm-syntax-operator"],
      [7, 9, "cm-syntax-keyword"],
      [9, 10, "cm-syntax-operator"],
      [10, 11, "cm-syntax-operator"],
      [15, 16, "cm-syntax-operator"],
    ]);
  });

  it("leaves a JSON or CSV sample plain", () => {
    expect(marks('{"order": "A-1"}')).toEqual([]);
    expect(marks("order,pickup\nA-1,03/10/2026")).toEqual([]);
  });
});
