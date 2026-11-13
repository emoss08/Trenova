import { describe, expect, it } from "vitest";
import {
  generateBreadcrumbSegments,
  generateFallbackTitle,
  getPageTitle,
  trailCrumbs,
} from "../route-utils";

describe("generateBreadcrumbSegments", () => {
  it("produces cumulative paths and Title Case labels", () => {
    const segments = generateBreadcrumbSegments("/admin/api-keys");
    expect(segments).toEqual([
      { path: "/admin", label: "Settings", source: "navigation" },
      { path: "/admin/api-keys", label: "API keys", source: "navigation" },
    ]);
  });

  it("uses the navigation label for every segment that has one", () => {
    const segments = generateBreadcrumbSegments("/hr/my-team");
    expect(segments).toEqual([
      { path: "/hr", label: "Human resource management", source: "navigation" },
      { path: "/hr/my-team", label: "My team", source: "navigation" },
    ]);
  });

  it("does not prefix-match a nav path onto a deeper segment", () => {
    const segments = generateBreadcrumbSegments("/hr/workers/wrk_01HZX2K3M4N5P6Q7R8S9T0V1");
    expect(segments.map((segment) => segment.label)).toEqual([
      "Human resource management",
      "Workers",
      "Details",
    ]);
  });

  it("folds an edit action into the record crumb it acts on", () => {
    const segments = generateBreadcrumbSegments(
      "/billing/configuration-files/formula-templates/ft_01HZX2K3M4N5P6Q7R8S9T0V1/edit",
    );
    expect(segments).toEqual([
      { path: "/billing", label: "Billing management", source: "navigation" },
      {
        path: "/billing/configuration-files",
        label: "Configuration files",
        source: "navigation",
      },
      {
        path: "/billing/configuration-files/formula-templates",
        label: "Formula templates",
        source: "navigation",
      },
      {
        path: "/billing/configuration-files/formula-templates/ft_01HZX2K3M4N5P6Q7R8S9T0V1/edit",
        label: "Details",
        source: "derived",
      },
    ]);
  });

  it("prefers a page-published label over the nav and segment labels", () => {
    const path = "/billing/configuration-files/formula-templates/ft_01HZX2K3M4N5P6Q7R8S9T0V1/edit";
    const segments = generateBreadcrumbSegments(path, {
      [path]: "Base Linehaul",
      "/billing/configuration-files/formula-templates": "Templates",
    });
    expect(segments.map((segment) => segment.label)).toEqual([
      "Billing management",
      "Configuration files",
      "Templates",
      "Base Linehaul",
    ]);
  });

  it("keeps a create segment as its own crumb", () => {
    const segments = generateBreadcrumbSegments("/admin/roles/new", {
      "/admin/roles/new": "New Role",
    });
    expect(segments.map((segment) => segment.label)).toEqual(["Settings", "Roles", "New Role"]);
  });

  it("does not fold an action word that does not follow a record id", () => {
    const segments = generateBreadcrumbSegments("/settings/edit");
    expect(segments.map((segment) => segment.label)).toEqual(["Settings", "Edit"]);
  });

  it("returns empty array for root path", () => {
    expect(generateBreadcrumbSegments("/")).toEqual([]);
  });

  it("strips trailing slash", () => {
    const segments = generateBreadcrumbSegments("/settings/");
    expect(segments).toEqual([{ path: "/settings", label: "Settings", source: "derived" }]);
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

describe("generateBreadcrumbSegments group labels", () => {
  it("names the prefix a navigation group's pages share after the group", () => {
    const segments = generateBreadcrumbSegments("/accounting/ar/aging");
    expect(segments.map((segment) => segment.label)).toEqual([
      "Accounting management",
      "Accounts receivable",
      "AR aging",
    ]);
  });

  it("names a module's own root after the module, even when its base path is a page inside it", () => {
    const segments = generateBreadcrumbSegments("/edi/lading");
    expect(segments.map((segment) => segment.label)).toEqual(["EDI", "Lading studio"]);
  });

  it("does not let a group claim the module itself when its pages sit directly under it", () => {
    const segments = generateBreadcrumbSegments("/hr/credential-types");
    expect(segments.map((segment) => segment.label)).toEqual([
      "Human resource management",
      "Credential types",
    ]);
  });
});

describe("trailCrumbs", () => {
  const french = (label: string) => `fr:${label}`;

  it("leaves out folders of the URL that no page or section fills", () => {
    const path = "/accounting/journal-entries/source/invoice/inv_01HZX2K3M4N5P6Q7R8S9T0V1";
    expect(
      trailCrumbs(generateBreadcrumbSegments(path), "Journal entry", (label) => label),
    ).toEqual([{ path: "/accounting", label: "Accounting management" }]);
  });

  it("keeps every section above a page as a link, in the reader's language", () => {
    const segments = generateBreadcrumbSegments("/billing/configuration-files/formula-libraries");
    expect(trailCrumbs(segments, "fr:Formula libraries", french)).toEqual([
      { path: "/billing", label: "fr:Billing management" },
      { path: "/billing/configuration-files", label: "fr:Configuration files" },
    ]);
  });

  it("shows a record name a page published as it was written, untranslated", () => {
    const parent = "/billing/configuration-files/factoring-companies/fac_01HZX2K3M4N5P6Q7R8S9T0V1";
    const segments = generateBreadcrumbSegments(`${parent}/agreements`, {
      [parent]: "Apex Capital",
    });
    expect(trailCrumbs(segments, "Agreements", french).at(-1)).toEqual({
      path: parent,
      label: "Apex Capital",
    });
  });

  it("drops a parent that only repeats the title", () => {
    const segments = generateBreadcrumbSegments("/shipment-management/shipments");
    expect(trailCrumbs(segments, "Shipments", (label) => label)).toEqual([]);
  });

  it("leads the freight outlook with the dispatch section it sits in", () => {
    const segments = generateBreadcrumbSegments("/dispatch/outlook");
    expect(trailCrumbs(segments, "Freight outlook", (label) => label)).toEqual([
      { path: "/dispatch", label: "Dispatch management" },
    ]);
  });

  it("is empty for a page with no parents", () => {
    expect(trailCrumbs(generateBreadcrumbSegments("/settings"), "Settings", french)).toEqual([]);
  });
});
