import { withRouteCatalogs } from "@/lib/route-catalogs";
import type { RouteObject } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const catalogs = vi.hoisted(() => {
  let release: () => void = () => undefined;
  let fail: (error: Error) => void = () => undefined;
  return {
    whenCatalogsReady: vi.fn(
      () =>
        new Promise<void>((resolve, reject) => {
          release = resolve;
          fail = reject;
        }),
    ),
    release: () => release(),
    fail: (error: Error) => fail(error),
  };
});

vi.mock("@trenova/shared/i18n/runtime", () => ({
  whenCatalogsReady: catalogs.whenCatalogsReady,
}));

function Page() {
  return null;
}

async function isSettled(promise: Promise<unknown>): Promise<boolean> {
  let settled = false;
  void promise.then(
    () => {
      settled = true;
    },
    () => {
      settled = true;
    },
  );
  await new Promise((resolve) => setTimeout(resolve, 0));
  return settled;
}

function lazyOf(route: RouteObject | undefined): () => Promise<unknown> {
  if (typeof route?.lazy !== "function") throw new Error("expected a lazy route");
  return route.lazy as () => Promise<unknown>;
}

beforeEach(() => {
  catalogs.whenCatalogsReady.mockClear();
});

describe("withRouteCatalogs", () => {
  it("holds a lazy route until the catalogs its code required have loaded", async () => {
    const [route] = withRouteCatalogs([
      { path: "/shipments", lazy: async () => ({ Component: Page }) },
    ]);

    const loading = lazyOf(route)();
    expect(await isSettled(loading)).toBe(false);
    expect(catalogs.whenCatalogsReady).toHaveBeenCalledTimes(1);

    catalogs.release();
    await expect(loading).resolves.toEqual({ Component: Page });
  });

  it("asks for the catalogs only after the route's module has evaluated", async () => {
    let evaluated = false;
    const [route] = withRouteCatalogs([
      {
        path: "/shipments",
        lazy: async () => {
          await Promise.resolve();
          evaluated = true;
          return { Component: Page };
        },
      },
    ]);
    catalogs.whenCatalogsReady.mockImplementationOnce(async () => {
      expect(evaluated).toBe(true);
    });

    await lazyOf(route)();
    expect(catalogs.whenCatalogsReady).toHaveBeenCalledTimes(1);
  });

  it("still renders the page, in English, when a catalog fails to download", async () => {
    const [route] = withRouteCatalogs([
      { path: "/shipments", lazy: async () => ({ Component: Page }) },
    ]);

    const loading = lazyOf(route)();
    await vi.waitFor(() => expect(catalogs.whenCatalogsReady).toHaveBeenCalled());
    catalogs.fail(new Error("chunk failed to load"));

    await expect(loading).resolves.toEqual({ Component: Page });
  });

  it("passes on a page module that fails to load", async () => {
    const [route] = withRouteCatalogs([
      {
        path: "/shipments",
        lazy: async () => {
          throw new Error("page chunk failed");
        },
      },
    ]);

    await expect(lazyOf(route)()).rejects.toThrow("page chunk failed");
    expect(catalogs.whenCatalogsReady).not.toHaveBeenCalled();
  });

  it("reaches lazy routes at every depth and leaves eager ones alone", async () => {
    const loader = vi.fn(() => null);
    const tree = withRouteCatalogs([
      {
        element: null,
        loader,
        children: [
          {
            path: "/admin",
            children: [{ path: "users", lazy: async () => ({ Component: Page }) }],
          },
          { path: "/eager", Component: Page },
        ],
      },
    ]);

    const [admin, eager] = tree[0].children ?? [];
    const users = admin.children?.[0];
    const loading = lazyOf(users)();
    expect(await isSettled(loading)).toBe(false);
    catalogs.release();
    await loading;

    expect(tree[0].loader).toBe(loader);
    expect(eager).toEqual({ path: "/eager", Component: Page });
    expect(eager.lazy).toBeUndefined();
  });

  it("holds each property of a lazy route object", async () => {
    const [route] = withRouteCatalogs([
      { path: "/shipments", lazy: { Component: async () => Page } },
    ]);
    const lazy = route.lazy;
    if (typeof lazy !== "object" || typeof lazy.Component !== "function") {
      throw new Error("expected a lazy route object");
    }

    const loading = lazy.Component();
    expect(await isSettled(loading)).toBe(false);
    catalogs.release();
    await expect(loading).resolves.toBe(Page);
  });

  it("does not change the route tree it was given", () => {
    const original = async () => ({ Component: Page });
    const routes: RouteObject[] = [{ path: "/", children: [{ path: "a", lazy: original }] }];

    withRouteCatalogs(routes);

    expect(routes[0].children?.[0].lazy).toBe(original);
  });
});
