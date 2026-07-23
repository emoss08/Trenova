import { describe, expect, it } from "vitest";
import {
  generateBreadcrumbSegments,
  generateFallbackTitle,
  getPageTitle,
} from "../route-utils";

describe("generateBreadcrumbSegments", () => {
  it("produces cumulative paths and Title Case labels", () => {
    const segments = generateBreadcrumbSegments("/admin/api-keys");
    expect(segments).toEqual([
      { path: "/admin", label: "Admin" },
      { path: "/admin/api-keys", label: "Api Keys" },
    ]);
  });

  it("returns empty array for root path", () => {
    expect(generateBreadcrumbSegments("/")).toEqual([]);
  });

  it("strips trailing slash", () => {
    const segments = generateBreadcrumbSegments("/settings/");
    expect(segments).toEqual([{ path: "/settings", label: "Settings" }]);
  });

  it("converts kebab-case to Title Case", () => {
    const segments = generateBreadcrumbSegments("/my-profile");
    expect(segments[0].label).toBe("My Profile");
  });

  it("splits camelCase", () => {
    const segments = generateBreadcrumbSegments("/myProfile");
    expect(segments[0].label).toContain("Profile");
  });
});

describe("generateFallbackTitle", () => {
  it("converts last segment to Title Case", () => {
    expect(generateFallbackTitle("/admin/settings")).toBe("Settings");
  });

  it("returns Home for root path", () => {
    expect(generateFallbackTitle("/")).toBe("Home");
  });

  it("converts kebab-case", () => {
    expect(generateFallbackTitle("/api-keys")).toBe("Api Keys");
  });

  it("handles trailing slash", () => {
    expect(generateFallbackTitle("/settings/")).toBe("Settings");
  });
});

describe("getPageTitle", () => {
  it("falls back to generateFallbackTitle for unknown route", () => {
    expect(getPageTitle("/nonexistent/route")).toBe("Route");
  });

  it("returns fallback Home for root", () => {
    expect(getPageTitle("/")).toBe("Home");
  });

  it("returns a string for any path", () => {
    expect(typeof getPageTitle("/some/deep/path")).toBe("string");
    expect(getPageTitle("/some/deep/path").length).toBeGreaterThan(0);
  });
});
