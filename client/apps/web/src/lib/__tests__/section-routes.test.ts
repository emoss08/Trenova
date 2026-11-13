import { navigationSections } from "@/lib/navigation-sections";
import { routes as pageRoutes } from "@/router";
import { matchRoutes, type RouteObject } from "react-router";
import { describe, expect, it } from "vitest";
import { sectionIndexRouteId, withSectionIndexes } from "../section-routes";

function leafOf(tree: RouteObject[], path: string): RouteObject | undefined {
  return matchRoutes(tree, path)?.at(-1)?.route;
}

/**
 * Every name in the trail is a link, so every section the navigation names
 * has to open something: its own page when it has one, the section index
 * otherwise. Dispatch management once led to the not-found page.
 */
describe("section indexes in the app's routes", () => {
  const routes = withSectionIndexes(pageRoutes);

  it("opens a page or a section index at every section the navigation names, never not-found", () => {
    const unfilled = navigationSections
      .map((section) => section.path)
      .filter((path) => {
        const leaf = leafOf(routes, path);
        return leaf === undefined || leaf.path === "*";
      });
    expect(unfilled).toEqual([]);
  });

  it("mounts the section index at the dispatch and billing sections a crumb leads to", () => {
    for (const path of [
      "/dispatch",
      "/billing",
      "/billing/configuration-files",
      "/accounting/ar",
    ]) {
      expect(leafOf(routes, path)?.id).toBe(sectionIndexRouteId(path));
    }
  });

  it("keeps the page a section already has", () => {
    for (const path of ["/accounting", "/reports", "/edi/overview", "/desk"]) {
      expect(leafOf(routes, path)?.id).not.toBe(sectionIndexRouteId(path));
    }
  });

  it("puts Settings' index inside the settings frame rather than beside it", () => {
    const matched = matchRoutes(routes, "/admin") ?? [];
    expect(matched.at(-1)?.route.id).toBe(sectionIndexRouteId("/admin"));
    expect(matched.at(-1)?.route.index).toBe(true);
    expect(matched.at(-2)?.route.path).toBe("admin");
  });
});

describe("withSectionIndexes", () => {
  const section = (path: string) => ({
    path,
    label: path,
    moduleId: "billing" as const,
    groupId: null,
  });

  it("mounts a section beside the not-found route, ahead of it, and leaves the input untouched", () => {
    const tree: RouteObject[] = [
      { path: "/billing/queue", element: null },
      { path: "*", element: null },
    ];
    const result = withSectionIndexes(tree, [section("/billing")]);

    expect(result.map((route) => route.id ?? route.path)).toEqual([
      "/billing/queue",
      sectionIndexRouteId("/billing"),
      "*",
    ]);
    expect(tree).toHaveLength(2);
  });

  it("treats a layout with an index as a page and one without as a frame to fill", () => {
    const tree: RouteObject[] = [
      { path: "/desk", element: null, children: [{ index: true, element: null }] },
      { path: "/admin", element: null, children: [{ path: "users", element: null }] },
      { path: "*", element: null },
    ];
    const result = withSectionIndexes(tree, [section("/desk"), section("/admin")]);

    expect(result.map((route) => route.id ?? route.path)).toEqual(["/desk", "/admin", "*"]);
    expect(result[1].children?.map((route) => route.id ?? route.path)).toEqual([
      sectionIndexRouteId("/admin"),
      "users",
    ]);
    expect(tree[1].children).toHaveLength(1);
  });

  it("refuses a tree with no not-found route to mount beside", () => {
    expect(() => withSectionIndexes([{ path: "/a", element: null }], [section("/b")])).toThrow(
      "withSectionIndexes needs a not-found route",
    );
  });
});
