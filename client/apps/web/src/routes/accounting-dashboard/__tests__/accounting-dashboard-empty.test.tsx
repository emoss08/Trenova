import { cleanup, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { AccountingDashboardPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchArDashboardKpis: vi.fn(),
  fetchArDsoTrend: vi.fn(),
}));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchArDashboardKpis: mocks.fetchArDashboardKpis,
  fetchArDsoTrend: mocks.fetchArDsoTrend,
  fetchArAgingTrend: vi.fn().mockResolvedValue([]),
  fetchArCashFlowForecast: vi.fn().mockResolvedValue([]),
  fetchArCollectionPerformance: vi.fn().mockResolvedValue(null),
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

function kpis(overview: Partial<Record<string, number>> = {}) {
  return {
    asOfDate: 1_757_000_000,
    overview: {
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
    cei: 0,
    avgDaysToPay: 0,
    overduePercent: 0,
    writeOffRatio: 0,
    disputeRate: 0,
    shortPayRate: 0,
  };
}

function week(over: Partial<{ billedMinor: number; arBalanceMinor: number }> = {}) {
  return { periodEnd: 1_756_000_000, dsoDays: 0, arBalanceMinor: 0, billedMinor: 0, ...over };
}

beforeEach(() => {
  mocks.fetchArDashboardKpis.mockResolvedValue(kpis());
  mocks.fetchArDsoTrend.mockResolvedValue([week(), week()]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("accounting dashboard empty state", () => {
  it("draws the dashboard it will become when nothing has ever been invoiced", async () => {
    renderAccountingPage(<AccountingDashboardPage />);

    expect(await screen.findByText("Nothing on the books yet")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open the billing queue" })).toHaveAttribute(
      "href",
      "/billing/queue",
    );
    expect(screen.queryByText("AR Outstanding")).not.toBeInTheDocument();
  });

  // A clear book with history is a healthy dashboard, not an empty one: the
  // figures should read as zero, with the trend still drawn.
  it("keeps the dashboard when the book is clear today but carried billing this year", async () => {
    mocks.fetchArDsoTrend.mockResolvedValue([week(), week({ billedMinor: 125_000 })]);
    renderAccountingPage(<AccountingDashboardPage />);

    expect(await screen.findByText("AR Outstanding")).toBeInTheDocument();
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
  });

  it("keeps the dashboard while an invoice is open", async () => {
    mocks.fetchArDashboardKpis.mockResolvedValue(
      kpis({ openInvoiceCount: 2, totalOpenMinor: 40_000 }),
    );
    renderAccountingPage(<AccountingDashboardPage />);

    expect(await screen.findByText("AR Outstanding")).toBeInTheDocument();
    expect(screen.queryByText("Nothing on the books yet")).not.toBeInTheDocument();
  });
});
