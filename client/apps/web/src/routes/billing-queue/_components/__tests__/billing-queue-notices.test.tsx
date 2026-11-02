import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import { billingQueueItemSchema } from "@trenova/shared/types/billing-queue";
import { queueItem } from "./split-shipment-fixture";

const { getItem } = vi.hoisted(() => ({ getItem: vi.fn() }));

vi.mock("@/lib/queries/billing-queue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/billing-queue")>();
  return {
    ...actual,
    billingQueue: {
      ...actual.billingQueue,
      get: (id: string) => ({ queryKey: ["billingQueue", "get", id], queryFn: () => getItem(id) }),
    },
  };
});
vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateCharges: vi.fn(), reassignCharge: vi.fn() } },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../billing-queue-action-bar", () => ({ BillingQueueActionBar: () => null }));
vi.mock("../billing-queue-documents-tab", () => ({ BillingQueueDocumentsTab: () => null }));
vi.mock("../billing-queue-rerate-dialog", () => ({ BillingQueueRerateDialog: () => null }));
vi.mock("../billing-queue-reassign-charge-dialog", () => ({
  BillingQueueReassignChargeDialog: () => null,
}));
vi.mock("@/routes/shipment/_components/comments", () => ({ default: () => null }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderPane() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter>
          <BillingQueueDetailPane selectedItemId="bqi_peak" onDocumentSelect={vi.fn()} />
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

/**
 * Peak's item as the server sends it after an adjusted invoice was credited and rebilled at
 * current rates: the replacement waits on a review, rerated 4.25% above the original, and is
 * flagged for an exception the biller raised.
 */
function rebilledItem() {
  return billingQueueItemSchema.parse({
    ...queueItem({ payer: "peak" }),
    isAdjustmentOrigin: true,
    rebillStrategy: "Rerate",
    requiresReplacementReview: true,
    rerateVariancePercent: "4.25",
    sourceInvoiceId: "inv_01original",
    sourceCreditMemoInvoiceId: "inv_01creditmemo",
    sourceInvoiceAdjustmentId: "iadj_01adjustment",
    correctionGroupId: "icg_01group",
    adjustmentContext: {
      rebill: {
        replacementLines: [{ id: "il_01freight" }, { id: "il_01lumper" }],
        subtotalAmount: "1250.50",
        otherAmount: "185.00",
        totalAmount: "1435.50",
        accountingDate: 1_780_416_000,
      },
    },
    exceptionReasonCode: "IncorrectRates",
    exceptionNotes: "Customer disputes the fuel surcharge.",
  });
}

// Every note, exception and hold above the tabs is the same blue notice, so none reads louder
// than another, and the adjusted invoice's rebill says what it is without a record ID in sight.
describe("billing queue review pane notices", () => {
  it("draws every notice in the same blue", async () => {
    getItem.mockResolvedValue(rebilledItem());
    renderPane();

    const rebill = await screen.findByTestId("billing-queue-adjustment-origin");
    const exception = screen.getByText("Incorrect Rates").closest('[data-slot="alert"]');

    expect(rebill).toHaveClass("bg-info-subtle");
    expect(exception).toHaveClass("bg-info-subtle");
    expect(exception).toHaveTextContent("Customer disputes the fuel surcharge.");
  });

  it("says how an adjusted invoice is rebilled, what it waits on and where it came from", async () => {
    getItem.mockResolvedValue(rebilledItem());
    renderPane();

    const rebill = await screen.findByTestId("billing-queue-adjustment-origin");

    expect(rebill).toHaveTextContent("Rebills an adjusted invoice");
    expect(rebill).toHaveTextContent("Rerate · Recalculate from current rates");
    expect(rebill).toHaveTextContent("Rerated +4.25% from the original invoice");
    expect(rebill).toHaveTextContent("Needs a replacement review before its invoice is created");
    expect(rebill).toHaveTextContent("Original invoice");
    expect(rebill).toHaveTextContent("Credit memo");
    expect(rebill.textContent).not.toMatch(/inv_|iadj_|icg_/u);
  });

  it("opens the replacement the adjustment priced in place, and folds it away again", async () => {
    const user = userEvent.setup();
    getItem.mockResolvedValue(rebilledItem());
    renderPane();

    const rebill = await screen.findByTestId("billing-queue-adjustment-origin");
    expect(within(rebill).queryByText("Replacement total")).not.toBeInTheDocument();

    await user.click(within(rebill).getByRole("button", { name: "Details" }));

    const details = within(rebill).getByLabelText("Replacement invoice");
    expect(details).toHaveTextContent("Replacement total$1,435.50");
    expect(details).toHaveTextContent("Freight$1,250.50");
    expect(details).toHaveTextContent("Accessorials$185.00");
    expect(details).toHaveTextContent("Lines2 lines");
    expect(details).toHaveTextContent("Accounting date");
    expect(rebill.textContent).not.toMatch(/il_|inv_|iadj_|icg_/u);

    await user.click(within(rebill).getByRole("button", { name: "Hide details" }));

    await waitFor(() =>
      expect(within(rebill).queryByText("Replacement total")).not.toBeInTheDocument(),
    );
  });
});
