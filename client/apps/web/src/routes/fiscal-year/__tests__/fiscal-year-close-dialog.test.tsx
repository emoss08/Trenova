import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AlertDialog } from "@trenova/shared/components/ui/alert-dialog";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FiscalYearClosePlan } from "@/types/fiscal-year";
import { FiscalYearCloseAlertDialogContent } from "../_components/fiscal-year-alert-dialog-content";

const mocks = vi.hoisted(() => ({
  closePreview: vi.fn(),
  close: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

vi.mock("@/services/api", () => ({
  apiService: {
    fiscalYearService: {
      closePreview: mocks.closePreview,
      close: mocks.close,
      reopen: vi.fn(),
      activate: vi.fn(),
    },
  },
}));

function plan(overrides: Partial<FiscalYearClosePlan> = {}): FiscalYearClosePlan {
  return {
    fiscalYearId: "fyr_01",
    fiscalYearName: "FY 2031",
    nextFiscalYearId: "fyr_02",
    nextFiscalYearName: "FY 2032",
    retainedEarningsAccountId: "gla_re",
    retainedEarningsAccountCode: "3900",
    retainedEarningsAccountName: "Retained Earnings",
    revenueMinor: 1_000_000,
    costOfRevenueMinor: 400_000,
    operatingExpenseMinor: 250_000,
    netIncomeMinor: 350_000,
    closingEntry: {
      kind: "Closing",
      fiscalYearId: "fyr_01",
      fiscalPeriodId: "",
      fiscalPeriodName: "Adjusting Period - FY 2031",
      createsPeriod: true,
      accountingDate: 1_956_527_999,
      description: "Year-end close of FY 2031",
      totalDebitMinor: 1_000_000,
      totalCreditMinor: 1_000_000,
      lines: [
        {
          glAccountId: "gla_rev",
          accountCode: "4000",
          accountName: "Line Haul Revenue",
          accountCategory: "Revenue",
          debitMinor: 1_000_000,
          creditMinor: 0,
          isRetainedEarnings: false,
        },
      ],
    },
    openingEntry: {
      kind: "Opening",
      fiscalYearId: "fyr_02",
      fiscalPeriodId: "fp_01",
      fiscalPeriodName: "Period 1 - FY 2032",
      createsPeriod: false,
      accountingDate: 1_956_528_000,
      description: "Opening balances carried forward from FY 2031 to FY 2032",
      totalDebitMinor: 700_000,
      totalCreditMinor: 700_000,
      lines: [
        {
          glAccountId: "gla_cash",
          accountCode: "1000",
          accountName: "Cash",
          accountCategory: "Asset",
          debitMinor: 700_000,
          creditMinor: 0,
          isRetainedEarnings: false,
        },
      ],
    },
    subledgerChecks: [
      {
        key: "accounts_receivable",
        label: "Accounts Receivable",
        glAccountId: "gla_ar",
        accountCode: "1110",
        accountName: "Accounts Receivable",
        glBalanceMinor: 400_000,
        subledgerBalanceMinor: 400_000,
        differenceMinor: 0,
        toleranceMinor: 0,
        reconciled: true,
        enforced: false,
      },
    ],
    revision: 0,
    canClose: true,
    blockers: [],
    ...overrides,
  };
}

// endDate is in the past so the unrelated early-close warning stays out of the way.
const record = { id: "fyr_01", year: 2031, endDate: 1_000_000 } as never;

function renderCloseDialog() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <AlertDialog open>{children}</AlertDialog>
      </QueryClientProvider>
    );
  }

  return render(<FiscalYearCloseAlertDialogContent record={record} onClose={vi.fn()} />, {
    wrapper: Wrapper,
  });
}

describe("FiscalYearCloseAlertDialogContent", () => {
  beforeEach(() => {
    mocks.closePreview.mockReset();
    mocks.close.mockReset();
  });

  it("shows the entries the close will post and the account the result rolls into", async () => {
    mocks.closePreview.mockResolvedValue(plan());

    renderCloseDialog();

    expect(await screen.findByText("Net income")).toBeInTheDocument();
    expect(screen.getByText("$3,500.00")).toBeInTheDocument();
    expect(screen.getByText(/3900 Retained Earnings/)).toBeInTheDocument();
    expect(screen.getByText(/carried forward into FY 2032/)).toBeInTheDocument();
    expect(screen.getByText("Closing entry")).toBeInTheDocument();
    expect(
      screen.getByText(/Adjusting Period - FY 2031 \(created by this close\)/),
    ).toBeInTheDocument();
    expect(screen.getByText("Opening entry")).toBeInTheDocument();
  });

  // The GL carries one AR total; the customer detail behind it lives in the
  // subledger, so the close shows that the two still agree.
  it("shows the control accounts it reconciled against their subledgers", async () => {
    mocks.closePreview.mockResolvedValue(plan());

    renderCloseDialog();

    expect(await screen.findByText(/Accounts Receivable \(1110\)/)).toBeInTheDocument();
    expect(screen.getByText(/reconciled · \$4,000\.00/)).toBeInTheDocument();
  });

  it("flags a control account that no longer matches its subledger", async () => {
    mocks.closePreview.mockResolvedValue(
      plan({
        subledgerChecks: [
          {
            key: "accounts_receivable",
            label: "Accounts Receivable",
            glAccountId: "gla_ar",
            accountCode: "1110",
            accountName: "Accounts Receivable",
            glBalanceMinor: 400_000,
            subledgerBalanceMinor: 375_000,
            differenceMinor: 25_000,
            toleranceMinor: 0,
            reconciled: false,
            enforced: false,
          },
        ],
      }),
    );

    renderCloseDialog();

    expect(await screen.findByText(/off by \$250\.00/)).toBeInTheDocument();
    // Reported, but the carrier has not asked for it to stop the close.
    expect(screen.getByText(/not enforced/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post and Close Year" })).toBeEnabled();
  });

  it("refuses to close while the preview reports blockers", async () => {
    mocks.closePreview.mockResolvedValue(
      plan({
        canClose: false,
        closingEntry: null,
        openingEntry: null,
        blockers: [
          {
            field: "defaultRetainedEarningsAccountId",
            code: "INVALID",
            message: "Set a default retained earnings account in Accounting Control.",
            category: "accounting",
          },
          {
            field: "nextFiscalYear",
            code: "INVALID",
            message: "Create the fiscal year that follows FY 2031 before closing it.",
            category: "accounting",
          },
        ],
      }),
    );

    renderCloseDialog();

    expect(await screen.findByText("2 issues block this close")).toBeInTheDocument();
    expect(
      screen.getByText("Set a default retained earnings account in Accounting Control."),
    ).toBeInTheDocument();

    const action = screen.getByRole("button", { name: "Post and Close Year" });
    expect(action).toBeDisabled();

    await userEvent.click(action);
    expect(mocks.close).not.toHaveBeenCalled();
  });

  it("posts the close when nothing blocks it", async () => {
    mocks.closePreview.mockResolvedValue(plan());
    mocks.close.mockResolvedValue({ id: "fyr_01" });

    renderCloseDialog();

    const action = await screen.findByRole("button", { name: "Post and Close Year" });
    await waitFor(() => expect(action).toBeEnabled());

    await userEvent.click(action);
    await waitFor(() => expect(mocks.close).toHaveBeenCalledWith("fyr_01"));
  });
});
