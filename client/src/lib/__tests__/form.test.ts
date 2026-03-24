import { describe, expect, it } from "vitest";
import { checkSectionErrors } from "../form";

describe("checkSectionErrors", () => {
  it("returns true when top-level error exists", () => {
    const errors = { name: { message: "Required", type: "required" } };
    expect(checkSectionErrors(errors as any, ["name"] as any)).toBe(true);
  });

  it("returns true for nested path", () => {
    const errors = { profile: { dob: { message: "Invalid", type: "invalid" } } };
    expect(checkSectionErrors(errors as any, ["profile.dob"] as any)).toBe(true);
  });

  it("returns false when no error at path", () => {
    const errors = { name: { message: "Required", type: "required" } };
    expect(checkSectionErrors(errors as any, ["email"] as any)).toBe(false);
  });

  it("returns false for empty errors object", () => {
    expect(checkSectionErrors({} as any, ["name"] as any)).toBe(false);
  });

  it("returns false for empty fields array", () => {
    const errors = { name: { message: "Required", type: "required" } };
    expect(checkSectionErrors(errors as any, [] as any)).toBe(false);
  });

  it("traverses deep path correctly", () => {
    const errors = { a: { b: { c: { message: "Deep", type: "invalid" } } } };
    expect(checkSectionErrors(errors as any, ["a.b.c"] as any)).toBe(true);
  });

  it("returns false for non-existent path gracefully", () => {
    const errors = { name: { message: "Required", type: "required" } };
    expect(checkSectionErrors(errors as any, ["x.y.z"] as any)).toBe(false);
  });

  it("returns true if any field in array has error", () => {
    const errors = { email: { message: "Invalid", type: "invalid" } };
    expect(checkSectionErrors(errors as any, ["name", "email"] as any)).toBe(true);
  });
});
