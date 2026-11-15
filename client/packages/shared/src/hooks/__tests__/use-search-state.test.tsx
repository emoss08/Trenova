import { createTestRouter, TestRouter } from "@trenova/shared/testing/test-router";
import {
  createRootRoute,
  createRoute,
  Outlet,
  RouterProvider,
  useNavigate,
} from "@tanstack/react-router";
import { act, cleanup, render, renderHook, screen } from "@testing-library/react";
import {
  searchBoolean,
  searchCodec,
  searchInteger,
  searchLiteral,
  searchString,
} from "@trenova/shared/lib/router/search-codec";
import { useEffect, type ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { readSearchState, useSearchState, useSearchValue } from "../use-search-state";

const tagsCodec = searchCodec<string[]>({
  parse: (value) => (value ? value.split("|") : []),
  serialize: (value) => value.join("|"),
  eq: (a, b) => a.join("|") === b.join("|"),
}).withDefault([]);

const table = {
  pageIndex: searchInteger.withDefault(1),
  query: searchString.withDefault(""),
  open: searchBoolean.withDefault(false),
  tab: searchLiteral(["general", "security"]).withOptions({ history: "push" }),
  tags: tagsCodec,
  panel: searchString,
};

function setup(initial: string) {
  const router = createTestRouter(undefined, [initial]);
  const wrapper = ({ children }: { children: ReactNode }) => (
    <TestRouter router={router}>{children}</TestRouter>
  );
  const hook = renderHook(() => useSearchState(table), { wrapper });
  return { router, hook };
}

afterEach(cleanup);

describe("readSearchState", () => {
  it("reads each key with its parser and falls back to its default or null", () => {
    expect(
      readSearchState(table, { pageIndex: "3", open: "true", tab: "nope", tags: "a|b" }),
    ).toEqual({
      pageIndex: 3,
      query: "",
      open: true,
      tab: null,
      tags: ["a", "b"],
      panel: null,
    });
  });

  it("reads a route prefetch's URLSearchParams the same way", () => {
    expect(readSearchState(table, new URLSearchParams("pageIndex=2&panel=shp_1"))).toMatchObject({
      pageIndex: 2,
      panel: "shp_1",
    });
  });
});

describe("useSearchState", () => {
  it("reads the address", () => {
    const { hook } = setup("/shipments?pageIndex=4&query=acme&tab=security");

    expect(hook.result.current[0]).toMatchObject({ pageIndex: 4, query: "acme", tab: "security" });
  });

  it("writes only the keys it changes, keeping the rest of the address", async () => {
    const { router, hook } = setup("/shipments?query=acme&other=kept#rows");

    await act(() => hook.result.current[1]({ pageIndex: 2 }));

    expect(router.state.location.pathname).toBe("/shipments");
    expect(router.state.location.searchStr).toBe("?query=acme&other=kept&pageIndex=2");
    expect(router.state.location.hash).toBe("rows");
    expect(hook.result.current[0].pageIndex).toBe(2);
  });

  it("leaves out a key that is removed, back at its default or written as nothing", async () => {
    const { router, hook } = setup("/shipments?pageIndex=3&query=acme&panel=shp_1&tags=a");

    await act(() => hook.result.current[1]({ pageIndex: 1, query: "", panel: null, tags: [] }));

    expect(router.state.location.searchStr).toBe("");
  });

  it("clears every key it names when handed null, and only those", async () => {
    const { router, hook } = setup("/shipments?pageIndex=3&query=acme&other=kept");

    await act(() => hook.result.current[1](null));

    expect(router.state.location.searchStr).toBe("?other=kept");
  });

  it("starts each change from the address as it stands, so quick changes keep each other", async () => {
    const { router, hook } = setup("/shipments");
    const [, set] = hook.result.current;

    await act(async () => {
      void set((current) => ({ pageIndex: current.pageIndex + 1 }));
      void set((current) => ({ pageIndex: current.pageIndex + 1, query: "acme" }));
    });

    expect(router.state.location.searchStr).toBe("?pageIndex=3&query=acme");
  });

  it("replaces the history entry unless a key it changes, the hook or the call asks to push", async () => {
    const { router, hook } = setup("/settings");
    const entry = () => router.state.location.state.__TSR_index;

    await act(() => hook.result.current[1]({ query: "a" }));
    expect(entry()).toBe(0);

    await act(() => hook.result.current[1]({ tab: "security" }));
    expect(entry()).toBe(1);

    await act(() => hook.result.current[1]({ query: "b" }, { history: "push" }));
    expect(entry()).toBe(2);
  });

  it("keeps each value the same object until its key changes", async () => {
    const { hook } = setup("/shipments?tags=a|b");
    const first = hook.result.current[0];

    await act(() => hook.result.current[1]({ pageIndex: 2 }));

    expect(hook.result.current[0].tags).toBe(first.tags);
    expect(hook.result.current[0]).not.toBe(first);
  });

  it("keeps the values object while nothing it names changes", async () => {
    const { hook } = setup("/shipments?pageIndex=2");
    const first = hook.result.current[0];

    hook.rerender();

    expect(hook.result.current[0]).toBe(first);
  });

  it("keeps the setter from one render to the next", () => {
    const { hook } = setup("/shipments");
    const first = hook.result.current[1];

    hook.rerender();

    expect(hook.result.current[1]).toBe(first);
  });
});

describe("two hooks writing in the same moment", () => {
  it("write the address once, with both changes, as one history entry", async () => {
    const router = createTestRouter(undefined, ["/shipments?view=board"]);
    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestRouter router={router}>{children}</TestRouter>
    );
    const board = renderHook(
      () => useSearchValue("expanded", searchString.withOptions({ history: "push" })),
      { wrapper },
    );
    const panel = renderHook(() => useSearchState({ panel: searchString }), { wrapper });

    await act(async () => {
      void board.result.current[1]("shp_1");
      void panel.result.current[1]({ panel: "shp_1" });
    });

    expect(router.state.location.searchStr).toBe("?view=board&expanded=shp_1&panel=shp_1");
    expect(router.state.location.state.__TSR_index).toBe(1);
  });
});

describe("useSearchValue", () => {
  it("reads and writes one key", async () => {
    const router = createTestRouter(undefined, ["/audit?window=7"]);
    const { result } = renderHook(() => useSearchValue("window", searchInteger.withDefault(30)), {
      wrapper: ({ children }) => <TestRouter router={router}>{children}</TestRouter>,
    });

    expect(result.current[0]).toBe(7);
    await act(() => result.current[1]((current) => current * 2));

    expect(router.state.location.searchStr).toBe("?window=14");
  });
});

describe("a page being left", () => {
  it("keeps reading the address it was drawn at while the next page is on its way", async () => {
    const seen: string[] = [];
    function Table() {
      const [{ query }] = useSearchState(table);
      useEffect(() => void seen.push(query), [query]);
      const navigate = useNavigate();
      return (
        <button type="button" onClick={() => void navigate({ href: "/customers?query=other" })}>
          Customers
        </button>
      );
    }
    const rootRoute = createRootRoute({ component: Outlet });
    const router = createTestRouter(
      rootRoute.addChildren([
        createRoute({ getParentRoute: () => rootRoute, path: "/shipments", component: Table }),
        createRoute({
          getParentRoute: () => rootRoute,
          path: "/customers",
          beforeLoad: () => new Promise<void>(() => undefined),
          component: () => <p>Customers</p>,
        }),
      ]),
      ["/shipments?query=acme"],
    );
    await router.load();
    render(<RouterProvider router={router} />);

    await act(async () => screen.getByRole("button", { name: "Customers" }).click());

    expect(router.latestLocation.pathname).toBe("/customers");
    expect(seen).toEqual(["acme"]);
  });
});
