import type { InvoiceArContext } from "@/lib/graphql/invoice";
import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceDisputesTab } from "../invoice-disputes-tab";

const mocks = vi.hoisted(() => ({
  open: vi.fn(),
  resolve: vi.fn(),
  withdraw: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice", () => ({
  openInvoiceDispute: mocks.open,
  resolveInvoiceDispute: mocks.resolve,
  withdrawInvoiceDispute: mocks.withdraw,
}));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

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
  totalAmount: 1000,
  appliedAmount: 0,
} as Invoice;

const openCase = {
  id: "idsp_1",
  invoiceId: "inv_1",
  customerId: "cus_1",
  status: "Open" as const,
  reasonCode: "RateDiscrepancy" as const,
  disputedAmount: "300.00",
  disputedAmountMinor: 30_000,
  notes: "Rate on the quote was lower",
  openedById: "usr_1",
  openedAt: NOW,
  resolvedById: null,
  resolvedAt: null,
  resolution: null,
  resolutionAdjustmentId: null,
  resolutionNotes: "",
  version: 1,
  createdAt: NOW,
  updatedAt: NOW,
};

function arContext(overrides: Partial<InvoiceArContext> = {}): InvoiceArContext {
  return {
    id: "inv_1",
    openBalance: "1000.00",
    disputes: [],
    openDispute: null,
    paymentApplications: [],
    creditApplications: [],
    lateChargeAssessments: [],
    relatedInvoices: [],
    referenceInvoice: null,
    ...overrides,
  } as InvoiceArContext;
}

function renderTab(context: InvoiceArContext) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <InvoiceDisputesTab invoice={invoice} arContext={context} isLoading={false} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.open.mockResolvedValue(openCase);
  mocks.resolve.mockResolvedValue({ ...openCase, status: "Resolved", resolution: "InvoiceUpheld" });
  mocks.withdraw.mockResolvedValue({ ...openCase, status: "Withdrawn" });
});

describe("InvoiceDisputesTab", () => {
  it("opens a dispute with a reason, an amount and notes", async () => {
    const user = userEvent.setup();
    renderTab(arContext());

    await user.click(screen.getByRole("button", { name: "Open dispute" }));
    await user.click(await screen.findByRole("button", { name: "Select reason" }));
    await user.click(await screen.findByText("Service failure"));
    await user.clear(screen.getByLabelText("Disputed amount"));
    await user.type(screen.getByLabelText("Disputed amount"), "300");
    await user.type(screen.getByLabelText("Notes"), "Delivered two days late");
    await user.click(screen.getByRole("button", { name: "Open dispute", hidden: false }));

    await waitFor(() =>
      expect(mocks.open).toHaveBeenCalledWith({
        invoiceId: "inv_1",
        reasonCode: "ServiceFailure",
        disputedAmount: "300",
        notes: "Delivered two days late",
      }),
    );
  });

  it("lists the open case and offers to resolve or withdraw it, but not to open another", () => {
    renderTab(arContext({ disputes: [openCase], openDispute: openCase }));

    expect(screen.getByText("Rate discrepancy")).toBeInTheDocument();
    expect(screen.getByText("$300.00")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Resolve" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Withdraw" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open dispute" })).toBeNull();
  });

  it("requires the settling adjustment when the resolution issued a credit", async () => {
    const user = userEvent.setup();
    renderTab(arContext({ disputes: [openCase], openDispute: openCase }));

    await user.click(screen.getByRole("button", { name: "Resolve" }));
    await user.click(await screen.findByRole("button", { name: "Select resolution" }));
    await user.click(await screen.findByText("Credit issued"));
    await user.click(screen.getByRole("button", { name: "Resolve dispute" }));

    expect(
      await screen.findByText("Name the executed adjustment that settled this dispute"),
    ).toBeInTheDocument();
    expect(mocks.resolve).not.toHaveBeenCalled();
  });

  it("withdraws the open case", async () => {
    const user = userEvent.setup();
    renderTab(arContext({ disputes: [openCase], openDispute: openCase }));

    await user.click(screen.getByRole("button", { name: "Withdraw" }));
    await user.click(await screen.findByRole("button", { name: "Withdraw dispute" }));

    await waitFor(() =>
      expect(mocks.withdraw).toHaveBeenCalledWith({ disputeId: "idsp_1", notes: null }),
    );
  });
});
