import { cleanup, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { AccountingDashboardPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchArDashboardKpis: vi.fn(),
  fetchArDsoTrend: vi.fn(),
}));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchAccountingBriefing: vi.fn().mockResolvedValue({
    segments: [
      {
        text: "Nothing is owed to you and no cash is waiting to be applied.",
        link: null,
        customerId: null,
      },
    ],
    actions: [],
    narrated: false,
    generatedAt: 1_757_000_000,
    generation: 1,
  }),
  fetchArDashboardKpis: mocks.fetchArDashboardKpis,
  fetchArDsoTrend: mocks.fetchArDsoTrend,
  fetchArAgingTrend: vi.fn().mockResolvedValue([]),
  fetchArCashFlowForecast: vi.fn().mockResolvedValue([]),
  fetchArTopOverdueCustomers: vi.fn().mockResolvedValue([]),
  fetchArCollectionsWorklist: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

const BUCKETS = {
  currentMinor: 0,
  days1To30Minor: 0,
  days31To60Minor: 0,
  days61To90Minor: 0,
  daysOver90Minor: 0,
  totalOpenMinor: 0,
};

function kpis(overview: Partial<Record<string, number | boolean>> = {}) {
  return {
    asOfDate: 1_757_000_000,
    overview: {
      hasInvoiced: false,
      totalOpenMinor: 0,
      overdueMinor: 0,
      unappliedCashMinor: 0,
      disputedOpenMinor: 0,
      openInvoiceCount: 0,
      overdueInvoiceCount: 0,
      disputedInvoiceCount: 0,
      avgDaysPastDue: 0,
      buckets: BUCKETS,
      ...overview,
    },
    currentDsoDays: 0,
    dsoDeltaDays: 0,
    overduePercent: 0,
    collection: {
      totals: {
        periodStart: 1_749_000_000,
        periodEnd: 1_757_000_000,
        beginningOpenMinor: 0,
        endingOpenMinor: 0,
        endingCurrentMinor: 0,
        creditSalesMinor: 0,
        collectedMinor: 0,
        avgDaysToPay: 0,
        shortPayMinor: 0,
        shortPayApplicationCount: 0,
        applicationCount: 0,
        disputedInvoiceCount: 0,
        postedInvoiceCount: 0,
      },
      cei: 0,
      writeOffRatio: 0,
      disputeRate: 0,
      shortPayRate: 0,
    },
  };
}

function week() {
  return { periodEnd: 1_756_000_000, dsoDays: 0, arBalanceMinor: 0, billedMinor: 0 };
}

beforeEach(() => {
  mocks.fetchArDashboardKpis.mockResolvedValue(kpis());
  mocks.fetchArDsoTrend.mockResolvedValue([week(), week()]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const STREAMED_BRIEF_TIMEOUT_MS = 5_000;
const PAGE_TIMEOUT_MS = 3_000;

describe("accounting dashboard empty state", () => {
  it("draws the dashboard it will become when nothing has ever been invoiced", async () => {
    renderAccountingPage(<AccountingDashboardPage />);

    expect(
      await screen.findByText("Nothing on the books yet", undefined, { timeout: PAGE_TIMEOUT_MS }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the billing queue" })).toHaveAttribute(
      "href",
      "/billing/queue",
    );
    expect(screen.queryByText("AR outstanding")).not.toBeInTheDocument();
  });

  // A clear book with history is a healthy dashboard, not an empty one: the
  // figures should read as zero, with the trend still drawn.
  it("keeps the dashboard when the book is clear today but has invoiced before", async () => {
    mocks.fetchArDashboardKpis.mockResolvedValue(kpis({ hasInvoiced: true }));
    renderAccountingPage(<AccountingDashboardPage />);

    expect(await screen.findByText("AR outstanding")).toBeInTheDocument();
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
  });

  it("leads a dashboard with history with the day's brief", async () => {
    mocks.fetchArDashboardKpis.mockResolvedValue(kpis({ hasInvoiced: true }));
    renderAccountingPage(<AccountingDashboardPage />);

    await waitFor(
      () =>
        expect(screen.getByRole("region", { name: "Accounting brief" })).toHaveTextContent(
          /Nothing is owed to you and no cash is waiting to be applied\./,
        ),
      { timeout: STREAMED_BRIEF_TIMEOUT_MS },
    );
  });

  it("decides a never-invoiced book from the KPIs without reading the trend", async () => {
    renderAccountingPage(<AccountingDashboardPage />);

    expect(
      await screen.findByText("Nothing on the books yet", undefined, { timeout: PAGE_TIMEOUT_MS }),
    ).toBeInTheDocument();
    expect(mocks.fetchArDsoTrend).not.toHaveBeenCalled();
  });

  it("does not brief a book that has never invoiced", async () => {
    renderAccountingPage(<AccountingDashboardPage />);

    expect(
      await screen.findByText("Nothing on the books yet", undefined, { timeout: PAGE_TIMEOUT_MS }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Accounting brief" })).not.toBeInTheDocument();
  });

  it("holds the dashboard's skeleton, and reads no panel, until the KPIs answer", async () => {
    mocks.fetchArDashboardKpis.mockReturnValue(new Promise(() => {}));
    renderAccountingPage(<AccountingDashboardPage />);

    expect(
      await screen.findByRole("generic", { name: "Loading the accounting dashboard" }),
    ).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
    expect(mocks.fetchArDsoTrend).not.toHaveBeenCalled();
  });

  it("shows the dashboard when the KPIs cannot be read, for each panel to say what failed", async () => {
    mocks.fetchArDashboardKpis.mockRejectedValue(new Error("down"));
    renderAccountingPage(<AccountingDashboardPage />);

    await waitFor(() => expect(mocks.fetchArDsoTrend).toHaveBeenCalled());
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("generic", { name: "Loading the accounting dashboard" }),
    ).not.toBeInTheDocument();
  });

  it("keeps the dashboard while an invoice is open", async () => {
    mocks.fetchArDashboardKpis.mockResolvedValue(
      kpis({ hasInvoiced: true, openInvoiceCount: 2, totalOpenMinor: 40_000 }),
    );
    renderAccountingPage(<AccountingDashboardPage />);

    expect(await screen.findByText("AR outstanding")).toBeInTheDocument();
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
  });
});
