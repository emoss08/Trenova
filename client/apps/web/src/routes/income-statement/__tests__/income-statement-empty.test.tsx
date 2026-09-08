import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { IncomeStatementPage } from "../page";

vi.mock("@/services/api", () => ({
  apiService: { accountingReportService: { getIncomeStatement: vi.fn() } },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/accounting/fiscal-period-selector", () => ({
  FiscalPeriodSelector: () => null,
}));

afterEach(() => {
  cleanup();
});

describe("income statement empty state", () => {
  it("draws the statement it will become until a period is chosen", () => {
    renderAccountingPage(<IncomeStatementPage />);

    expect(screen.getByText("Pick a period")).toBeInTheDocument();
    expect(screen.getByText(/net income/)).toBeInTheDocument();
  });
});
