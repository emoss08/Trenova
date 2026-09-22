import type { Invoice } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { InvoiceVoidDialog } from "../invoice-void-dialog";

const mocks = vi.hoisted(() => ({
  voidInvoice: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice", () => ({ voidInvoice: mocks.voidInvoice }));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));

const invoice = {
  id: "inv_1",
  number: "INV-1",
  status: "Posted",
  appliedAmount: 0,
} as Invoice;

function renderDialog(onOpenChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <InvoiceVoidDialog invoice={invoice} open onOpenChange={onOpenChange} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return onOpenChange;
}

beforeEach(() => {
  mocks.voidInvoice.mockResolvedValue({
    invoice: { id: "inv_1", number: "INV-1", status: "Voided" },
    adjustmentId: null,
    pendingApproval: false,
    releasedQueueItemIds: ["bqi_1"],
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("InvoiceVoidDialog", () => {
  it("refuses to submit without a reason", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Void invoice" }));

    expect(await screen.findByText("Say why the invoice is being voided")).toBeInTheDocument();
    expect(mocks.voidInvoice).not.toHaveBeenCalled();
  });

  it("sends the reason and the chosen disposition, then closes on an immediate void", async () => {
    const user = userEvent.setup();
    const onOpenChange = renderDialog();

    await user.type(screen.getByLabelText("Reason"), "Duplicate of INV-7");
    await user.click(screen.getByRole("button", { name: /Do not rebill/ }));
    await user.click(await screen.findByText("Release freight for rebilling"));
    await user.click(screen.getByRole("button", { name: "Void invoice" }));

    await waitFor(() =>
      expect(mocks.voidInvoice).toHaveBeenCalledWith({
        invoiceId: "inv_1",
        reason: "Duplicate of INV-7",
        disposition: "Rebill",
      }),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mocks.toastSuccess).toHaveBeenCalled();
  });

  it("keeps the dialog open and says so when the reversal awaits an approver", async () => {
    mocks.voidInvoice.mockResolvedValueOnce({
      invoice: { id: "inv_1", number: "INV-1", status: "Posted" },
      adjustmentId: "adj_9",
      pendingApproval: true,
      releasedQueueItemIds: [],
    });
    const user = userEvent.setup();
    const onOpenChange = renderDialog();

    await user.type(screen.getByLabelText("Reason"), "Customer never received the freight");
    await user.click(screen.getByRole("button", { name: "Void invoice" }));

    expect(await screen.findByText(/awaiting reversal approval/)).toBeInTheDocument();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});
