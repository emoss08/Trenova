import type { NavModule } from "@/config/navigation.types";
import { describe, expect, it } from "vitest";
import {
  buildNavigationSections,
  findNavigationSection,
  navigationSections,
} from "../navigation-sections";

const Icon = () => null;

function module(overrides: Partial<NavModule> & Pick<NavModule, "id">): NavModule {
  return {
    label: "Module",
    icon: Icon,
    basePath: "#",
    navigation: [],
    ...overrides,
  };
}

describe("buildNavigationSections", () => {
  it("names a module's base path after the module and its declared prefixes after its short name", () => {
    const sections = buildNavigationSections([
      module({
        id: "edi",
        label: "Electronic data interchange",
        shortLabel: "EDI",
        basePath: "/edi/overview",
        routePrefixes: ["/edi"],
      }),
    ]);
    expect(sections).toEqual([
      {
        path: "/edi/overview",
        label: "Electronic data interchange",
        moduleId: "edi",
        groupId: null,
      },
      { path: "/edi", label: "EDI", moduleId: "edi", groupId: null },
    ]);
  });

  it("names the folder a group's pages share after the group", () => {
    const sections = buildNavigationSections([
      module({
        id: "accounting",
        label: "Accounting",
        basePath: "/accounting",
        navigation: [
          {
            id: "ar",
            label: "Accounts receivable",
            items: [
              { id: "aging", label: "Aging", path: "/accounting/ar/aging" },
              { id: "payments", label: "Payments", path: "/accounting/ar/payments/" },
            ],
          },
        ],
      }),
    ]);
    expect(sections).toContainEqual({
      path: "/accounting/ar",
      label: "Accounts receivable",
      moduleId: "accounting",
      groupId: "ar",
    });
  });

  it("gives no section to a group whose pages share no folder below the module's own", () => {
    const sections = buildNavigationSections([
      module({
        id: "hr",
        basePath: "/hr",
        navigation: [
          {
            id: "people",
            label: "People",
            items: [
              { id: "workers", label: "Workers", path: "/hr/workers" },
              { id: "my-team", label: "My team", path: "/hr/my-team" },
            ],
          },
          {
            id: "factoring",
            label: "Factoring",
            items: [
              { id: "overview", label: "Overview", path: "/hr/factoring" },
              { id: "submissions", label: "Submissions", path: "/hr/factoring/submissions" },
            ],
          },
        ],
      }),
    ]);
    expect(sections.map((section) => section.path)).toEqual(["/hr"]);
  });

  it("skips Home's root and a module with no base path, and lets the first claim on a path win", () => {
    const sections = buildNavigationSections([
      module({ id: "home", basePath: "/" }),
      module({ id: "shipment", basePath: "#", label: "Shipments" }),
      module({ id: "admin", label: "Settings", basePath: "/admin" }),
      module({ id: "organization", label: "Organization", basePath: "/admin" }),
    ]);
    expect(sections).toEqual([
      { path: "/admin", label: "Settings", moduleId: "admin", groupId: null },
    ]);
  });
});

describe("findNavigationSection", () => {
  it("finds the app's dispatch and billing configuration sections whatever the trailing slash", () => {
    expect(findNavigationSection("/dispatch/")).toMatchObject({
      label: "Dispatch management",
      moduleId: "dispatch",
      groupId: null,
    });
    expect(findNavigationSection("/billing/configuration-files")).toMatchObject({
      label: "Configuration files",
      moduleId: "billing",
    });
  });

  it("is null for a page that is not a section", () => {
    expect(findNavigationSection("/dispatch/outlook")).toBeNull();
    expect(navigationSections.some((section) => section.path === "/dispatch/outlook")).toBe(false);
  });
});
