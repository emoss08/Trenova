import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { CustomerStatementPage } from "../page";

const mocks = vi.hoisted(() => ({ fetchArCustomerStatement: vi.fn() }));

vi.mock("@/lib/graphql/accounts-receivable", () => ({
  fetchArCustomerStatement: mocks.fetchArCustomerStatement,
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));

const STATEMENT = {
  customerName: "Acme Freight",
  statementDate: 1_757_000_000,
  openingBalanceMinor: 0,
  endingBalanceMinor: 0,
  totalChargesMinor: 0,
  totalPaymentsMinor: 0,
  aging: {
    currentMinor: 0,
    days1To30Minor: 0,
    days31To60Minor: 0,
    days61To90Minor: 0,
    daysOver90Minor: 0,
    totalOpenMinor: 0,
  },
  openItems: [],
  transactions: [],
};

function renderStatement() {
  return renderAccountingPage(
    <Routes>
      <Route path="/statement/:customerId" element={<CustomerStatementPage />} />
    </Routes>,
    ["/statement/cus_1"],
  );
}

beforeEach(() => {
  mocks.fetchArCustomerStatement.mockResolvedValue(STATEMENT);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("customer statement empty states", () => {
  it("says nothing touched the account when the period has no transactions", async () => {
    renderStatement();

    expect(await screen.findByText("Nothing in this period")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // The dates are the only thing that can hide a transaction; once one is
  // set, the empty statement offers to drop them rather than only widen.
  it("offers to clear the dates once one narrows the statement", async () => {
    const user = userEvent.setup();
    renderStatement();
    await screen.findByText("Nothing in this period");

    const from = screen.getByLabelText("Start Date");
    await user.type(from, "2026-01-01");
    expect(await screen.findByRole("button", { name: "Clear filters" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getByLabelText("Start Date")).toHaveValue("");
  });
});
