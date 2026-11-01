import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueInvoiceTab } from "../billing-queue-invoice-tab";

const mocks = vi.hoisted(() => ({ updateInvoiceDetails: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateInvoiceDetails: mocks.updateInvoiceDetails } },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: Object.assign(vi.fn(), { error: vi.fn(), success: vi.fn() }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function item(overrides: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    status: "InReview",
    lane: "NeedsYou",
    billType: "Invoice",
    version: 7,
    poNumber: null,
    invoiceMemo: "Thanks",
    paymentTerm: "Net15",
    invoiceDate: null,
    packetDocumentIds: ["doc_pod"],
    billToCustomer: {
      id: "cus_1",
      name: "Acme Foods",
      billingProfile: { paymentTerm: "Net30" },
    },
    shipment: {
      id: "shp_1",
      customerId: "cus_1",
      customer: { id: "cus_1", name: "Acme Foods" },
    },
    ...overrides,
  } as unknown as BillingQueueItem;
}

function renderTab(value: BillingQueueItem) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <BillingQueueInvoiceTab item={value} />
    </QueryClientProvider>,
  );
}

describe("BillingQueueInvoiceTab", () => {
  it("saves a typed PO with everything else the invoice already says, against the version shown", async () => {
    mocks.updateInvoiceDetails.mockImplementation(async (_id, payload) => ({
      ...item(),
      ...payload,
      version: 8,
    }));
    renderTab(item());

    await userEvent.type(screen.getByRole("textbox", { name: /Customer PO/ }), "PO-55821");

    await waitFor(
      () =>
        expect(mocks.updateInvoiceDetails).toHaveBeenCalledWith("bqi_1", {
          version: 7,
          poNumber: "PO-55821",
          invoiceMemo: "Thanks",
          paymentTerm: "Net15",
          invoiceDate: null,
          packetDocumentIds: ["doc_pod"],
        }),
      { timeout: 3000 },
    );
    expect(mocks.updateInvoiceDetails).toHaveBeenCalledTimes(1);
  });

  it("saves nothing until something changes", async () => {
    renderTab(item());
    await new Promise((resolve) => setTimeout(resolve, 900));
    expect(mocks.updateInvoiceDetails).not.toHaveBeenCalled();
  });

  it("reads back what the invoice will say once the item is approved", () => {
    renderTab(item({ status: "Approved", poNumber: "PO-1", paymentTerm: null }));

    expect(screen.queryByRole("textbox", { name: /Customer PO/ })).not.toBeInTheDocument();
    expect(screen.getByText("PO-1")).toBeInTheDocument();
    expect(screen.getByText("Net 30")).toBeInTheDocument();
    expect(screen.getByText("The day it's approved")).toBeInTheDocument();
  });
});
