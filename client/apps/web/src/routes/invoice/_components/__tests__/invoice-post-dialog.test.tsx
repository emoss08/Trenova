import type { InvoicePostPreview } from "@trenova/shared/types/invoice";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoicePostDialog } from "../invoice-post-dialog";

const { getPostPreview, post } = vi.hoisted(() => ({
  getPostPreview: vi.fn(),
  post: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { invoiceService: { getPostPreview, post } },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function preview(overrides: Partial<InvoicePostPreview> = {}): InvoicePostPreview {
  return {
    invoiceId: "inv_1",
    number: "INV-3001",
    billToName: "Acme Logistics",
    currencyCode: "USD",
    totalMinor: 125000,
    refusals: [],
    alreadyPosted: false,
    journal: {
      accountingDate: 1_700_000_000,
      entryStatus: "Posted",
      requiresApproval: false,
      lines: [
        {
          glAccountId: "gla_ar",
          accountCode: "1200",
          accountName: "Accounts receivable",
          description: "Invoice INV-3001",
          debitMinor: 125000,
          creditMinor: 0,
        },
        {
          glAccountId: "gla_rev",
          accountCode: "4000",
          accountName: "Freight revenue",
          description: "Invoice INV-3001",
          debitMinor: 0,
          creditMinor: 125000,
        },
      ],
    },
    shipmentsBilled: ["PRO-88"],
    queueItemPosted: true,
    accountingSync: [],
    edi: null,
    ...overrides,
  };
}

function renderDialog(onOpenChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <InvoicePostDialog invoiceId="inv_1" open onOpenChange={onOpenChange} />
    </QueryClientProvider>,
  );
  return onOpenChange;
}

describe("InvoicePostDialog", () => {
  it("shows the entry posting books, by account, before anything is posted", async () => {
    getPostPreview.mockResolvedValue(preview());
    renderDialog();

    expect(await screen.findByText("1200 · Accounts receivable")).toBeInTheDocument();
    expect(screen.getByText("4000 · Freight revenue")).toBeInTheDocument();
    expect(screen.getAllByText("$1,250.00")).toHaveLength(2);
    expect(screen.getByText("Marks shipment PRO-88 invoiced")).toBeInTheDocument();
    expect(getPostPreview).toHaveBeenCalledWith("inv_1");
    expect(post).not.toHaveBeenCalled();
  });

  it("posts only when confirmed, then closes", async () => {
    getPostPreview.mockResolvedValue(preview());
    post.mockResolvedValue({ id: "inv_1", number: "INV-3001", status: "Posted" });
    const onOpenChange = renderDialog();

    await userEvent.click(await screen.findByRole("button", { name: "Post invoice" }));

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(post).toHaveBeenCalledTimes(1);
    expect(post.mock.calls[0][0]).toBe("inv_1");
  });

  it("names why a post would be refused and offers no way to post", async () => {
    getPostPreview.mockResolvedValue(
      preview({ refusals: ["The fiscal period for this date is closed"], journal: null }),
    );
    renderDialog();

    expect(
      await screen.findByText("The fiscal period for this date is closed"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Post invoice" })).toBeDisabled();
    expect(screen.queryByText("Ledger entry")).not.toBeInTheDocument();
  });
});
