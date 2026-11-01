import { describe, expect, it } from "vitest";
import { withSavedVersion } from "../saved-version";

describe("withSavedVersion", () => {
  it("takes the version the save gave the record", () => {
    expect(withSavedVersion({ name: "Halstead", version: 3 }, { id: "cus_1", version: 4 })).toEqual(
      { name: "Halstead", version: 4 },
    );
  });

  it("takes a version of zero rather than treating it as missing", () => {
    expect(withSavedVersion({ version: 2 }, { version: 0 })).toEqual({ version: 0 });
  });

  it("keeps the values when the save returns no version", () => {
    const values = { name: "Halstead", version: 3 };

    expect(withSavedVersion(values, { id: "cus_1" })).toBe(values);
    expect(withSavedVersion(values, { version: "4" })).toBe(values);
    expect(withSavedVersion(values, null)).toBe(values);
    expect(withSavedVersion(values, undefined)).toBe(values);
  });

  it("adds no version to a form that holds none", () => {
    const values = { name: "Halstead" };

    expect(withSavedVersion(values, { version: 4 })).toBe(values);
  });
});
