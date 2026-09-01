import { describe, expect, it } from "vitest";
import { coerceVariableDefaultValue } from "../variable-definition-editor";

describe("coerceVariableDefaultValue", () => {
  it("coerces numeric strings for Number variables", () => {
    expect(coerceVariableDefaultValue("Number", "3.25")).toBe(3.25);
    expect(coerceVariableDefaultValue("Number", "0")).toBe(0);
  });

  it("keeps non-numeric input for Number variables so validation can surface it", () => {
    expect(coerceVariableDefaultValue("Number", "abc")).toBe("abc");
  });

  it("coerces boolean strings for Boolean variables", () => {
    expect(coerceVariableDefaultValue("Boolean", "true")).toBe(true);
    expect(coerceVariableDefaultValue("Boolean", "false")).toBe(false);
    expect(coerceVariableDefaultValue("Boolean", "yes")).toBe("yes");
  });

  it("leaves String variables untouched", () => {
    expect(coerceVariableDefaultValue("String", "42")).toBe("42");
  });

  it("maps empty input to undefined so no default is persisted", () => {
    expect(coerceVariableDefaultValue("Number", "")).toBeUndefined();
    expect(coerceVariableDefaultValue("Number", null)).toBeUndefined();
    expect(coerceVariableDefaultValue("Number", undefined)).toBeUndefined();
  });

  it("passes already-typed values through", () => {
    expect(coerceVariableDefaultValue("Number", 5)).toBe(5);
    expect(coerceVariableDefaultValue("Boolean", true)).toBe(true);
  });
});

describe("coerceVariableDefaultValue across type changes", () => {
  it("turns a stored number into text when the type becomes String", () => {
    expect(coerceVariableDefaultValue("String", 5)).toBe("5");
  });

  it("keeps a number a number and a boolean a boolean", () => {
    expect(coerceVariableDefaultValue("Number", 5)).toBe(5);
    expect(coerceVariableDefaultValue("Boolean", true)).toBe(true);
  });
});
