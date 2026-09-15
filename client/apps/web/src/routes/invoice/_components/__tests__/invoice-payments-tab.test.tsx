import type { InvoiceArContext } from "@/lib/graphql/invoice";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoicePaymentsTab } from "../invoice-payments-tab";

const mocks = vi.hoisted(() => ({
  unapply: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice", () => ({
  unapplyCreditMemoApplication: mocks.unapply,
  applyCreditMemo: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../apply-credit-memo-dialog", () => ({ ApplyCreditMemoDialog: () => null }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const NOW = 1_789_325_147;

const invoice = {
  id: "inv_1",
  number: "INV-1",
  status: "Posted",
  billType: "Invoice",
  customerId: "cus_1",
  currencyCode: "USD",
  totalAmount: "1000.00",
  appliedAmount: "250.00",
  settlementStatus: "PartiallyPaid",
  dueDate: NOW - 40 * 86400,
} as Invoice;

function arContext(overrides: Partial<InvoiceArContext> = {}): InvoiceArContext {
  return {
    id: "inv_1",
    number: "INV-1",
    status: "Posted",
    billType: "Invoice",
    currencyCode: "USD",
    appliedAmount: "250.00",
    openBalance: "750.00",
    balanceDueMinor: 75_000,
    creditRemaining: "0",
    daysPastDue: 40,
    paymentApplications: [
      {
        id: "cpa_1",
        customerPaymentId: "cp_1",
        invoiceId: "inv_1",
        appliedAmountMinor: 15_000,
        shortPayAmountMinor: 500,
        lineNumber: 1,
        createdAt: NOW,
        payment: {
          id: "cp_1",
          paymentDate: NOW,
          amountMinor: 15_000,
          status: "Posted",
          paymentMethod: "ACH",
          referenceNumber: "ACH-4411",
        },
      },
    ],
    creditApplications: [
      {
        id: "cma_1",
        creditMemoInvoiceId: "inv_cm",
        invoiceId: "inv_1",
        appliedAmountMinor: 10_000,
        accountingDate: NOW,
        lineNumber: 1,
        status: "Applied",
        unappliedAt: null,
        unappliedById: null,
        unappliedReason: "",
        createdById: "usr_1",
        createdAt: NOW,
        updatedAt: NOW,
        creditMemo: { id: "inv_cm", number: "CM-7", billType: "CreditMemo", status: "Posted" },
        invoice: { id: "inv_1", number: "INV-1", billType: "Invoice", status: "Posted" },
      },
    ],
    disputes: [],
    openDispute: null,
    lateChargeAssessments: [],
    relatedInvoices: [],
    referenceInvoice: null,
    ...overrides,
  } as InvoiceArContext;
}

function renderTab(context: InvoiceArContext | null, value: Invoice = invoice) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <InvoicePaymentsTab invoice={value} arContext={context} isLoading={false} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.unapply.mockResolvedValue({ id: "cma_1", status: "Unapplied" });
});

describe("InvoicePaymentsTab", () => {
  it("shows what is owed, what was paid, and the days past due", () => {
    renderTab(arContext());

    const totals = within(screen.getByTestId("invoice-payment-totals"));
    expect(totals.getByText("Open balance").nextSibling).toHaveTextContent("$750.00");
    expect(totals.getByText("Applied").nextSibling).toHaveTextContent("$250.00");
    expect(screen.getByText("40 days past due")).toBeInTheDocument();
  });

  it("lists cash applications with their payment and credit applications with their memo", () => {
    renderTab(arContext());

    expect(screen.getByText("ACH-4411")).toBeInTheDocument();
    expect(screen.getByText("$150.00")).toBeInTheDocument();
    expect(screen.getByText("CM-7")).toBeInTheDocument();
  });

  it("unapplies a credit memo application on request", async () => {
    const user = userEvent.setup();
    renderTab(arContext());

    await user.click(screen.getByRole("button", { name: "Unapply" }));

    await waitFor(() =>
      expect(mocks.unapply).toHaveBeenCalledWith({ applicationId: "cma_1", reason: null }),
    );
  });

  it("links to recording a payment against this invoice while it is open", () => {
    renderTab(arContext());

    expect(screen.getByRole("link", { name: "Record payment" })).toHaveAttribute(
      "href",
      "/accounting/ar/payments?panelType=create&customerId=cus_1&invoiceIds=inv_1",
    );
  });

  it("says nothing has been applied yet instead of drawing empty tables", () => {
    renderTab(
      arContext({ paymentApplications: [], creditApplications: [], openBalance: "1000.00" }),
    );

    expect(screen.getByText("No payments applied yet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Unapply" })).toBeNull();
  });
});
