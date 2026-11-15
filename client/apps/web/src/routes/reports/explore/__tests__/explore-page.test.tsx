import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CannedReport } from "@/lib/graphql/reports";
import { ReportExplorePage } from "../page";

const mocks = vi.hoisted(() => ({
  canned: vi.fn(),
  preview: vi.fn(),
}));

vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  useCannedReports: mocks.canned,
  useReportDefinition: () => ({ data: undefined, isLoading: false, isError: false }),
  useReportPreview: mocks.preview,
}));
vi.mock("@/hooks/use-breadcrumb-label", () => ({ useBreadcrumbLabel: vi.fn() }));

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

function renderExplore(cannedKey: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter initialEntries={[`/reports/explore?canned=${cannedKey}`]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route path="/reports/explore" element={<ReportExplorePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
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
    isError: false,
  });
  mocks.preview.mockReturnValue({
    data: undefined,
    isError: false,
    isPending: true,
    isFetching: false,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ReportExplorePage", () => {
  it("says a gallery report the plan leaves out is part of its add-on", () => {
    renderExplore("factoring-schedules");

    expect(screen.getByText("Not on your plan")).toBeInTheDocument();
    expect(screen.getByText("Part of Accounting & Cash")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Factoring Schedules is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Export/ })).toBeNull();
    expect(document.body.textContent).not.toContain("tms.factoring");
  });

  it("explains a preview the plan refuses instead of showing the error", () => {
    mocks.preview.mockReturnValue({
      data: undefined,
      isError: true,
      isPending: false,
      isFetching: false,
      error: {
        graphQLErrors: [
          {
            message: "Plan Restricted",
            extensions: {
              code: "PLAN_RESTRICTED",
              params: { feature: "tms.accounting", addOn: "Accounting & Cash" },
            },
          },
        ],
      },
    });

    renderExplore("open-invoice-aging");

    expect(screen.getByText("Not on your plan")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Open Invoice Aging is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("Plan Restricted")).toBeNull();
  });

  it("explores an available gallery report", () => {
    renderExplore("open-invoice-aging");

    expect(screen.getByText("Open Invoice Aging")).toBeInTheDocument();
    expect(screen.queryByText("Not on your plan")).toBeNull();
  });
});
