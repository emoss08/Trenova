import { describe, expect, it } from "vitest";
import { locationTimezoneSchema } from "../location";

describe("locationTimezoneSchema", () => {
  it("accepts an IANA zone or nothing at all", () => {
    expect(locationTimezoneSchema.parse("America/Chicago")).toBe("America/Chicago");
    expect(locationTimezoneSchema.parse("")).toBe("");
    expect(locationTimezoneSchema.parse(null)).toBe("");
    expect(locationTimezoneSchema.parse(undefined)).toBe("");
  });

  it("rejects a zone the runtime does not know", () => {
    expect(() => locationTimezoneSchema.parse("Mars/Olympus_Mons")).toThrow();
  });
});
