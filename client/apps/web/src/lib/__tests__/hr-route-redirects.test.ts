import { routes } from "@/router";
import type { LoaderFunctionArgs, RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function joinPath(parent: string, child: string): string {
  return child.startsWith("/") ? child : `${parent.replace(/\/$/, "")}/${child}`;
}

function collect(
  entries: RouteObject[],
  parent = "",
  found = new Map<string, RouteObject>(),
): Map<string, RouteObject> {
  for (const entry of entries) {
    const path = entry.path === undefined ? parent : joinPath(parent, entry.path);
    if (entry.path !== undefined) found.set(path, entry);
    if (entry.children) collect(entry.children, path, found);
  }
  return found;
}

const routesByPath = collect(routes);

/**
 * Links already sent to people — notification rows, bookmarks — still carry the
 * old dispatch paths, so each one keeps working and lands on its HR home.
 */
const MOVED = [
  ["/dispatch/workers", "/hr/workers"],
  ["/dispatch/configuration-files/checklist-templates", "/hr/checklist-templates"],
  ["/dispatch/configuration-files/review-templates", "/hr/review-templates"],
  ["/dispatch/configuration-files/training-courses", "/hr/training-courses"],
  ["/dispatch/configuration-files/credential-types", "/hr/credential-types"],
  ["/dispatch/configuration-files/holidays", "/hr/holidays"],
  ["/dispatch/configuration-files/pto-policies", "/hr/pto-policies"],
] as const;

async function locationFor(path: string, search = ""): Promise<string | null> {
  const route = routesByPath.get(path);
  if (!route) throw new Error(`route ${path} is not registered`);
  if (typeof route.loader !== "function") throw new Error(`route ${path} has no loader`);

  // A redirect loader answers synchronously; anything else may throw its
  // Response instead of returning it.
  const result: unknown = await Promise.resolve(
    route.loader({
      request: new Request(`http://localhost${path}${search}`),
      params: {},
      context: undefined,
    } as unknown as LoaderFunctionArgs),
  ).catch((error: unknown) => error);

  return result instanceof Response ? result.headers.get("Location") : null;
}

describe("HR routes", () => {
  it("registers every page under /hr", () => {
    for (const [, next] of MOVED) {
      expect(routesByPath.has(next)).toBe(true);
    }
  });

  it("redirects the old dispatch paths to their HR home", async () => {
    for (const [legacy, next] of MOVED) {
      await expect(locationFor(legacy)).resolves.toBe(next);
    }
  });

  it("keeps the query string, so a deep link still opens the panel it named", async () => {
    await expect(
      locationFor("/dispatch/workers", "?panelType=edit&panelEntityId=wrk_1&tab=credentials"),
    ).resolves.toBe("/hr/workers?panelType=edit&panelEntityId=wrk_1&tab=credentials");
  });
});
