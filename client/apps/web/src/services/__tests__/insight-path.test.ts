import { activeInsightsPath } from "@/services/insight";
import { describe, expect, it } from "vitest";

/**
 * A page asks for its own slice by naming a surface. The home widget names
 * none and must keep getting everything, so the parameter is only sent when a
 * surface is given: an empty `surface=` would be read by the server as absent
 * anyway, but a URL that says nothing it does not mean is easier to trust.
 */
describe("activeInsightsPath", () => {
  it("names the surface a page is asking for", () => {
    expect(activeInsightsPath(4, "Accounting")).toBe("/insights/?limit=4&surface=Accounting");
  });

  it("asks for the whole view when no surface is given", () => {
    expect(activeInsightsPath(6)).toBe("/insights/?limit=6");
  });
});
