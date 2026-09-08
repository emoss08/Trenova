import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CannedGallery } from "../canned-gallery";
import { DashboardGallery } from "../dashboard-gallery";
import { ReportDefinitionGrid } from "../report-definition-grid";

const mocks = vi.hoisted(() => ({
  definitions: vi.fn(),
  canned: vi.fn(),
  dashboards: vi.fn(),
  createDashboard: vi.fn(),
}));

vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  useReportDefinitionList: mocks.definitions,
  useCannedReports: mocks.canned,
  useReportDashboards: mocks.dashboards,
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
vi.mock("../use-create-dashboard-action", () => ({
  useCreateDashboardAction: () => ({ create: mocks.createDashboard, isPending: false }),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

function renderWithin(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
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

beforeEach(() => {
  mocks.definitions.mockReturnValue({ data: [], isLoading: false });
  mocks.canned.mockReturnValue({ data: [], isLoading: false });
  mocks.dashboards.mockReturnValue({ data: [], isLoading: false, isError: false, error: null });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("reports empty states", () => {
  it("draws the library it will become and offers the builder when there are no reports", async () => {
    const user = userEvent.setup();
    renderWithin(
      <ReportDefinitionGrid
        search=""
        sortBy="name_asc"
        category="all"
        status="all"
        onClearFilters={vi.fn()}
      />,
    );

    expect(screen.getByRole("heading", { name: "No reports yet" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "New report" }));
    expect(await screen.findByText("builder opened")).toBeInTheDocument();
  });

  it("offers to clear the library's search and filters when they hide every report", async () => {
    const onClearFilters = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <ReportDefinitionGrid
        search="zzz"
        sortBy="name_asc"
        category="all"
        status="draft"
        onClearFilters={onClearFilters}
      />,
    );

    expect(screen.getByRole("heading", { name: "Nothing matches" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New report" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });

  it("offers to clear the gallery's search when it hides every canned report", async () => {
    const onClearFilters = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <CannedGallery
        search="zzz"
        sortBy="name_asc"
        category="all"
        onClearFilters={onClearFilters}
      />,
    );

    expect(screen.getByRole("heading", { name: "Nothing matches" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });

  it("draws the dashboards it will become and offers to create the first", async () => {
    const user = userEvent.setup();
    renderWithin(<DashboardGallery search="" sortBy="name_asc" onClearFilters={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "No dashboards yet" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "New dashboard" }));
    expect(mocks.createDashboard).toHaveBeenCalledTimes(1);
  });

  it("offers to clear a dashboard search instead of creating", async () => {
    const onClearFilters = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <DashboardGallery search="zzz" sortBy="name_asc" onClearFilters={onClearFilters} />,
    );

    expect(screen.getByRole("heading", { name: "Nothing matches" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New dashboard" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });

  // A failed load must not look like an empty library.
  it("keeps the error notice apart from the empty sketch", () => {
    mocks.dashboards.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("boom"),
    });
    renderWithin(<DashboardGallery search="" sortBy="name_asc" onClearFilters={vi.fn()} />);

    expect(screen.getByText("Dashboards could not be loaded")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "No dashboards yet" })).not.toBeInTheDocument();
  });
});
