import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CannedReport } from "@/lib/graphql/reports";
import { CannedGallery, SHOW_ADD_ON_REPORTS_STORAGE_KEY } from "../canned-gallery";

const mocks = vi.hoisted(() => ({
  canned: vi.fn(),
}));

vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  useCannedReports: mocks.canned,
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
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

function cannedReport(overrides: Partial<CannedReport> & { key: string; name: string }) {
  return {
    version: "1",
    description: "",
    category: "Finance",
    tags: [],
    defaultFormat: "csv",
    definition: { version: 2, entity: "invoice", columns: [] },
    available: true,
    planDenial: null,
    ...overrides,
  } satisfies CannedReport;
}

function renderGallery({ search = "", category = "all" } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <TestRouter>
      <QueryClientProvider client={client}>
        <CannedGallery
          search={search}
          sortBy="name_asc"
          category={category}
          onClearFilters={vi.fn()}
        />
      </QueryClientProvider>
    </TestRouter>,
  );
}

function addOnSwitch() {
  return screen.getByRole("switch", { name: "Show reports from add-ons" });
}

beforeEach(() => {
  window.localStorage.clear();
  mocks.canned.mockReturnValue({
    data: [
      cannedReport({ key: "open-invoice-aging", name: "Open Invoice Aging" }),
      cannedReport({
        key: "factoring-schedules",
        name: "Factoring Schedules",
        available: false,
        planDenial: {
          feature: "tms.factoring",
          featureName: "Factoring",
          addOnName: "Accounting & Cash",
        },
      }),
    ],
    isLoading: false,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  window.localStorage.clear();
});

describe("CannedGallery", () => {
  it("hides a report the plan does not include by default, and counts it as hidden", () => {
    renderGallery();

    expect(screen.queryByRole("heading", { name: "Factoring Schedules" })).not.toBeInTheDocument();
    expect(addOnSwitch()).not.toBeChecked();
    expect(screen.getByText("1 hidden")).toBeInTheDocument();
    expect(screen.getByText("1 report")).toBeInTheDocument();
  });

  it("shows a report the plan does not include as a locked card, by name, that cannot be run, once asked", () => {
    renderGallery();
    fireEvent.click(addOnSwitch());

    const locked = screen.getByRole("heading", { name: "Factoring Schedules" }).closest("section");
    expect(locked).toHaveAttribute("data-slot", "plan-locked");
    expect(
      within(locked as HTMLElement).getByText("Part of Accounting & Cash"),
    ).toBeInTheDocument();
    expect(within(locked as HTMLElement).queryByRole("button")).not.toBeInTheDocument();
    expect(within(locked as HTMLElement).queryByRole("link")).not.toBeInTheDocument();
  });

  it("remembers the choice for the person", () => {
    renderGallery();
    fireEvent.click(addOnSwitch());

    expect(window.localStorage.getItem(SHOW_ADD_ON_REPORTS_STORAGE_KEY)).toBe("true");
    cleanup();

    renderGallery();

    expect(addOnSwitch()).toBeChecked();
    expect(screen.getByRole("heading", { name: "Factoring Schedules" })).toBeInTheDocument();
    expect(screen.queryByText("1 hidden")).not.toBeInTheDocument();
  });

  it("offers no control when the plan includes every gallery report", () => {
    mocks.canned.mockReturnValue({
      data: [cannedReport({ key: "open-invoice-aging", name: "Open Invoice Aging" })],
      isLoading: false,
    });

    renderGallery();

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.queryByText(/hidden/)).not.toBeInTheDocument();
  });

  it("drops a category whose every report is hidden instead of drawing it empty", () => {
    mocks.canned.mockReturnValue({
      data: [
        cannedReport({
          key: "open-invoice-aging",
          name: "Open Invoice Aging",
          category: "Billing",
        }),
        cannedReport({
          key: "factoring-schedules",
          name: "Factoring Schedules",
          category: "Customers",
          available: false,
          planDenial: {
            feature: "tms.factoring",
            featureName: "Factoring",
            addOnName: "Accounting & Cash",
          },
        }),
      ],
      isLoading: false,
    });

    renderGallery();

    expect(screen.getByRole("heading", { name: "Billing" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Customers" })).not.toBeInTheDocument();
    expect(screen.getAllByText("1 report")).toHaveLength(1);
  });

  it("says only add-on reports match, with a way to show them, rather than that nothing does", () => {
    renderGallery({ search: "factoring" });

    expect(screen.getByText("Only add-on reports match")).toBeInTheDocument();
    expect(screen.queryByText("Nothing matches")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Show reports from add-ons" }));

    expect(screen.getByRole("heading", { name: "Factoring Schedules" })).toBeInTheDocument();
  });

  it("keeps the run, customize and explore actions on a report the plan includes", () => {
    renderGallery();

    expect(screen.getByRole("heading", { name: "Open Invoice Aging" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Run" })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: "Customize" })).toHaveLength(1);
    expect(screen.getAllByRole("link", { name: "Explore" })).toHaveLength(1);
  });
});
