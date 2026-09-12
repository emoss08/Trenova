import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import type { BalanceSheet } from "@/types/balance-sheet";
import { BalanceSheetPage } from "../page";

const mocks = vi.hoisted(() => ({ getBalanceSheet: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { accountingReportService: { getBalanceSheet: mocks.getBalanceSheet } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/accounting/fiscal-period-selector", () => ({
  FiscalPeriodSelector: ({ onChange }: { onChange: (id: string) => void }) => (
    <button type="button" onClick={() => onChange("fp_01")}>
      Pick period
    </button>
  ),
}));

function sheet(overrides: Partial<BalanceSheet> = {}): BalanceSheet {
  return {
    fiscalPeriodId: "fp_01",
    assets: { label: "Assets", totalMinor: 1_000_000, lines: [] },
    liabilities: { label: "Liabilities", totalMinor: 300_000, lines: [] },
    equity: { label: "Equity", totalMinor: 350_000, lines: [] },
    currentYearEarningsMinor: 125_000,
    totalAssetsMinor: 1_000_000,
    totalLiabilitiesMinor: 300_000,
    totalEquityMinor: 475_000,
    ...overrides,
  };
}

afterEach(() => {
  cleanup();
});

describe("balance sheet equity", () => {
  beforeEach(() => {
    mocks.getBalanceSheet.mockReset();
  });

  // The figure is the year's result to date, not one period's movement — it is
  // what the year-end close moves into retained earnings.
  it("reports the year's earnings rather than the period's", async () => {
    mocks.getBalanceSheet.mockResolvedValue(sheet());

    renderAccountingPage(<BalanceSheetPage />);
    await userEvent.click(screen.getByRole("button", { name: "Pick period" }));

    expect(await screen.findByText("Current Year Earnings")).toBeInTheDocument();
    expect(screen.getByText("$1,250.00")).toBeInTheDocument();
    expect(screen.queryByText("Current Period Net Income")).not.toBeInTheDocument();
  });

  it("omits the line when the year has no result yet", async () => {
    mocks.getBalanceSheet.mockResolvedValue(
      sheet({ currentYearEarningsMinor: 0, totalEquityMinor: 350_000 }),
    );

    renderAccountingPage(<BalanceSheetPage />);
    await userEvent.click(screen.getByRole("button", { name: "Pick period" }));

    // The section footer and the page total both read "Total Equity".
    expect(await screen.findAllByText("Total Equity")).not.toHaveLength(0);
    expect(screen.queryByText("Current Year Earnings")).not.toBeInTheDocument();
  });
});
