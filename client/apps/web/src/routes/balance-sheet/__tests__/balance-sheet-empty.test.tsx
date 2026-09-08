import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { BalanceSheetPage } from "../page";

vi.mock("@/services/api", () => ({
  apiService: { accountingReportService: { getBalanceSheet: vi.fn() } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/accounting/fiscal-period-selector", () => ({
  FiscalPeriodSelector: () => null,
}));

afterEach(() => {
  cleanup();
});

describe("balance sheet empty state", () => {
  it("draws the sheet it will become until a period is chosen", () => {
    renderAccountingPage(<BalanceSheetPage />);

    expect(screen.getByText("Pick a period")).toBeInTheDocument();
    expect(screen.getByText(/liabilities and equity/)).toBeInTheDocument();
  });
});
