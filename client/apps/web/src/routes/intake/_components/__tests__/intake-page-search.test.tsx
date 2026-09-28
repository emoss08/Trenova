import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { IntakePage } from "../../page";

// The queue's panes are replaced by stand-ins so the test drives only what
// the page owns: the search box's text, and how it and the address agree.
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("../batch-list", () => ({
  BatchList: ({
    search,
    onSearchChange,
    empty,
  }: {
    search: string;
    onSearchChange: (value: string) => void;
    empty: { title: string; action?: ReactNode };
  }) => (
    <>
      <input
        aria-label="Search"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
      />
      <p>{empty.title}</p>
      {empty.action}
    </>
  ),
}));
vi.mock("../batch-workspace", () => ({ BatchWorkspace: () => null }));
vi.mock("../intake-rail", () => ({
  IntakeRail: ({
    filter,
    onChange,
  }: {
    filter: { mine: boolean };
    onChange: (filter: { mine: boolean }) => void;
  }) => (
    <button type="button" onClick={() => onChange({ ...filter, mine: !filter.mine })}>
      Only mine
    </button>
  ),
}));
vi.mock("@/lib/queries/capture", () => ({
  captureBatchesQuery: (filter: unknown) => ({
    queryKey: ["capture", "batches", filter],
    queryFn: () => new Promise(() => {}),
    initialPageParam: null,
    getNextPageParam: () => undefined,
  }),
}));
vi.mock("@/lib/queries", () => ({
  queries: {
    capture: {
      batchCount: (filter: unknown) => ({
        queryKey: ["capture", "batchCount", filter],
        queryFn: () => Promise.resolve(0),
      }),
    },
  },
}));

const DEBOUNCE_SETTLED_MS = 600;

function renderAt(initialEntries: string[]) {
  const router = createMemoryRouter([{ path: "/intake", element: <IntakePage /> }], {
    initialEntries,
    initialIndex: initialEntries.length - 1,
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
  return router;
}

function query(router: ReturnType<typeof createMemoryRouter>) {
  return new URLSearchParams(router.state.location.search).get("q");
}

async function settle() {
  await act(() => new Promise((resolve) => setTimeout(resolve, DEBOUNCE_SETTLED_MS)));
}

describe("IntakePage search", () => {
  it("shows what the address says after the address changes under it", async () => {
    const router = renderAt(["/intake?q=acme"]);
    expect(screen.getByRole("textbox", { name: "Search" })).toHaveValue("acme");

    await act(() => router.navigate("/intake"));
    await settle();

    expect(screen.getByRole("textbox", { name: "Search" })).toHaveValue("");
    expect(query(router)).toBeNull();
  });

  it("undoes a search with Back", async () => {
    const user = userEvent.setup();
    const router = renderAt(["/intake"]);

    await user.type(screen.getByRole("textbox", { name: "Search" }), "acme");
    await waitFor(() => expect(query(router)).toBe("acme"));

    await act(() => router.navigate(-1));
    await settle();

    expect(router.state.location.search).toBe("");
    expect(screen.getByRole("textbox", { name: "Search" })).toHaveValue("");
  });

  it("refines a search in place rather than adding a step for every pause", async () => {
    const user = userEvent.setup();
    const router = renderAt(["/intake"]);
    const box = screen.getByRole("textbox", { name: "Search" });

    await user.type(box, "acme");
    await waitFor(() => expect(query(router)).toBe("acme"));
    await user.type(box, " freight");
    await waitFor(() => expect(query(router)).toBe("acme freight"));

    await act(() => router.navigate(-1));
    await settle();

    expect(query(router)).toBeNull();
    expect(box).toHaveValue("");
  });

  it("says when nothing matches the search and clears it from there", async () => {
    const user = userEvent.setup();
    const router = renderAt(["/intake?q=acme"]);

    expect(screen.getByText("No stack matches “acme”")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear the search" }));

    await waitFor(() => expect(query(router)).toBeNull());
    expect(screen.getByRole("textbox", { name: "Search" })).toHaveValue("");
    expect(screen.getByText("Nothing is waiting to be filed")).toBeInTheDocument();
  });

  it("reaches the source and owner filters from the small-screen bar", async () => {
    const user = userEvent.setup();
    const router = renderAt(["/intake"]);

    await user.click(screen.getByRole("button", { name: "Filters" }));
    const sheet = await screen.findByRole("dialog", { name: "Views and filters" });
    await user.click(within(sheet).getByRole("button", { name: "Only mine" }));

    await waitFor(() =>
      expect(new URLSearchParams(router.state.location.search).get("mine")).toBe("1"),
    );
  });
});
