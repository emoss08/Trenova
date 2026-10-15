import { routes } from "@/router";
import type { RouteHandle } from "@/types/router";
import type { LoaderFunctionArgs, RouteObject } from "react-router";
import { describe, expect, it, vi } from "vitest";

vi.mock("@trenova/shared/lib/session-loader", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/session-loader")>()),
  requireSession: async () => null,
}));
vi.mock("@/lib/onboarding-gate", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/onboarding-gate")>()),
  resolveOnboardingRedirect: async () => null,
}));
vi.mock("@/lib/route-prefetch", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/route-prefetch")>()),
  createPrefetchLoader: () => () => null,
}));

function find(entries: RouteObject[], path: string): RouteObject | undefined {
  for (const entry of entries) {
    if (entry.path === path) return entry;
    const found = entry.children ? find(entry.children, path) : undefined;
    if (found) return found;
  }
  return undefined;
}

async function locationFor(path: string, search = ""): Promise<string | null> {
  const route = find(routes, path);
  if (typeof route?.loader !== "function") throw new Error(`route ${path} has no loader`);
  const result: unknown = await Promise.resolve(
    route.loader({
      request: new Request(`http://localhost${path}${search}`),
      params: {},
      context: undefined,
    } as unknown as LoaderFunctionArgs),
  ).catch((error: unknown) => error);
  return result instanceof Response ? result.headers.get("Location") : null;
}

describe("Settings route", () => {
  it("is a page of its own, framed as a room beside the module rail", () => {
    const route = find(routes, "/settings");

    expect(route).toBeDefined();
    expect((route?.handle as RouteHandle | undefined)?.frame).toBe("room");
  });

  it("sends a link from an email already sent to the notifications section", async () => {
    await expect(locationFor("/", "?settings=notifications")).resolves.toBe(
      "/settings?section=notifications",
    );
  });

  it("opens Settings at the top for an old link naming a section that no longer exists", async () => {
    await expect(locationFor("/", "?settings=appearance")).resolves.toBe("/settings");
  });

  it("leaves Home alone when the address carries no old Settings link", async () => {
    await expect(locationFor("/")).resolves.toBeNull();
    await expect(locationFor("/", "?tab=today")).resolves.toBeNull();
  });
});
