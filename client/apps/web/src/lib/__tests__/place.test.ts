import { describe, expect, it } from "vitest";
import { cityStateLabel } from "@/lib/place";

describe("cityStateLabel", () => {
  it("names the city and the state, or whichever is known", () => {
    expect(cityStateLabel("Atlanta", "GA")).toBe("Atlanta, GA");
    expect(cityStateLabel("Atlanta", null)).toBe("Atlanta");
    expect(cityStateLabel("", "GA")).toBe("GA");
    expect(cityStateLabel(undefined, undefined)).toBeNull();
  });
});
