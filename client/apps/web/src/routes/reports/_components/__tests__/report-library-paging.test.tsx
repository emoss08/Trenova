import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ReportDefinitionGrid } from "../report-definition-grid";

/**
 * The library grid read a fixed first hundred reports and showed nothing to say
 * a hundred and first existed — and because it narrows by category and status
 * over what it holds, a filter could report an empty library that was merely an
 * unloaded one. These pin the way out of both.
 */

const mocks = vi.hoisted(() => ({
  definitions: vi.fn(),
  fetchNextPage: vi.fn(),
}));

vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  useReportDefinitionList: mocks.definitions,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
  usePermissions: () => ({
    canRead: true,
    canCreate: true,
    canUpdate: true,
    canDelete: true,
    canExport: true,
    isLoading: false,
  }),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

function definition(name: string, overrides: Record<string, unknown> = {}) {
  return {
    id: `rdef_${name}`,
    name,
    description: "",
    category: "Operations",
    tags: [],
    kind: "custom",
    cannedKey: null,
    ownerId: "usr_1",
    visibility: "private",
    status: "active",
    diagnostics: [],
    definition: { entity: "shipment", columns: [] },
    defaultFormat: "csv",
    currentRevision: 1,
    lastRunAt: null,
    version: 1,
    createdAt: 1_700_000_000,
    updatedAt: 1_700_000_000,
    ...overrides,
  };
}

function stubLibrary(
  rows: ReturnType<typeof definition>[],
  overrides: { hasNextPage?: boolean; isFetchingNextPage?: boolean } = {},
) {
  mocks.definitions.mockReturnValue({
    data: rows,
    isLoading: false,
    hasNextPage: overrides.hasNextPage ?? false,
    isFetchingNextPage: overrides.isFetchingNextPage ?? false,
    fetchNextPage: mocks.fetchNextPage,
  });
}

function renderGrid(props: Partial<Parameters<typeof ReportDefinitionGrid>[0]> = {}): void {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui: ReactElement = (
    <ReportDefinitionGrid
      search=""
      sortBy="name_asc"
      category="all"
      status="all"
      onClearFilters={vi.fn()}
      {...props}
    />
  );

  render(
    <MemoryRouter initialEntries={["/reports"]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route path="/reports" element={ui} />
          <Route path="/reports/builder" element={<p>builder opened</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const LOAD_MORE = "Load more reports";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

beforeEach(() => {
  stubLibrary([definition("Aging Detail")]);
});

describe("the report library grid", () => {
  it("says nothing about more reports when the library is fully loaded", () => {
    renderGrid();

    expect(screen.getByText("Aging Detail")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: LOAD_MORE })).toBeNull();
  });

  it("offers the rest of the library when there is more of it", async () => {
    stubLibrary([definition("Aging Detail")], { hasNextPage: true });
    const user = userEvent.setup();
    renderGrid();

    await user.click(screen.getByRole("button", { name: LOAD_MORE }));

    expect(mocks.fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("will not ask for the same page twice while it is arriving", () => {
    stubLibrary([definition("Aging Detail")], { hasNextPage: true, isFetchingNextPage: true });
    renderGrid();

    expect(screen.getByRole("button", { name: /Loading/ })).toBeDisabled();
  });

  // Category and status are narrowed over the pages already held, so an empty
  // result with pages behind it is not an empty library.
  it("does not claim the library is empty when a filter hid an unloaded page", async () => {
    stubLibrary([definition("Aging Detail", { category: "Operations" })], { hasNextPage: true });
    const user = userEvent.setup();
    renderGrid({ category: "Finance" });

    expect(screen.getByText(/pages loaded so far/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: LOAD_MORE }));
    expect(mocks.fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("still says the library is empty when there is genuinely nothing left to load", () => {
    stubLibrary([]);
    renderGrid();

    expect(screen.getByRole("heading", { name: "No reports yet" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: LOAD_MORE })).toBeNull();
  });
});
