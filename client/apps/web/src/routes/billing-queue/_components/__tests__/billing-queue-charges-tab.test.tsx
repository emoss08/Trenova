import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueChargesTab } from "../billing-queue-charges-tab";
import { ACME_SHARE_JSON, DETENTION_ID, queueItem } from "./split-shipment-fixture";

const { updateCharges, reassignCharge } = vi.hoisted(() => ({
  updateCharges: vi.fn(),
  reassignCharge: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateCharges, reassignCharge } },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("../billing-queue-charge-dialog", () => ({ BillingQueueChargeDialog: () => null }));
vi.mock("../billing-queue-rerate-dialog", () => ({ BillingQueueRerateDialog: () => null }));
vi.mock("../billing-queue-reassign-charge-dialog", () => ({
  BillingQueueReassignChargeDialog: ({
    open,
    line,
  }: {
    open: boolean;
    line: { description: string } | null;
  }) => (open && line ? <p>Reassigning {line.description}</p> : null),
}));

afterEach(cleanup);
beforeEach(() => {
  vi.clearAllMocks();
});

function renderTab(item: BillingQueueItem) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <NuqsTestingAdapter>
      <QueryClientProvider client={client}>
        <BillingQueueChargesTab item={item} />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

describe("BillingQueueChargesTab payer bill", () => {
  // Acme pays $1,350 of the freight and all of the detention. The tab is
  // Acme's invoice, not the shipment's charge sheet.
  it("shows what the payer owes, not the shipment total", () => {
    renderTab(queueItem({ payer: "acme" }));

    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$2,504.38");
    expect(screen.queryByText(/shipment total/)).not.toBeInTheDocument();

    const freight = screen.getByTestId("payer-line-freight");
    expect(freight).toHaveTextContent("$1,350.00");
    expect(freight).toHaveTextContent("of $2,850.00");
    expect(freight).not.toHaveTextContent("Amount split");

    const detention = screen.getByTestId(`payer-line-${DETENTION_ID}`);
    expect(detention).toHaveTextContent("Detention Fee");
    expect(detention).toHaveTextContent("$1,154.38");
    expect(detention).not.toHaveTextContent("of $");

    expect(screen.queryByText(/Billed to other payers/)).not.toBeInTheDocument();
  });

  it("moves charges another payer owes out of the bill", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak" }));

    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$1,500.00");
    expect(screen.getByTestId("payer-line-freight")).toHaveTextContent("$1,500.00");
    expect(screen.queryByTestId(`payer-line-${DETENTION_ID}`)).not.toBeInTheDocument();
    expect(screen.getByText("No accessorial charges for this payer")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Billed to other payers \(1\)/ }));
    const other = await screen.findByTestId(`other-payer-line-${DETENTION_ID}`);
    expect(other).toHaveTextContent("Detention Fee");
    expect(other).toHaveTextContent("Acme Manufacturing");
    expect(other).toHaveTextContent("$1,154.38");
  });

  it("warns that charges are shared while they can still be changed", () => {
    renderTab(queueItem({ payer: "peak" }));

    expect(
      screen.getByText(
        "Charges belong to the shipment, so changing one here also changes the other payers' bills.",
      ),
    ).toBeInTheDocument();
  });

  it("drops the shared-charges warning once nothing on the item can change", () => {
    renderTab(queueItem({ payer: "peak", status: "Approved" }));

    expect(screen.queryByText(/Charges belong to the shipment/)).not.toBeInTheDocument();
  });

  it("falls back to the shipment's charges when the split cannot be resolved", () => {
    renderTab(
      queueItem({
        payer: "acme",
        share: {
          ...ACME_SHARE_JSON,
          lines: [],
          payers: [],
          freightAmount: "0",
          accessorialAmount: "0",
          totalAmount: "0",
          resolutionError: "Amount allocations for this charge must add up to 3000.00",
        },
      }),
    );

    expect(
      screen.getByText("Amount allocations for this charge must add up to 3000.00"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("payer-bill-total")).not.toBeInTheDocument();
    expect(screen.getByText("$4,004.38")).toBeInTheDocument();
  });
});

describe("BillingQueueChargesTab safe edits", () => {
  // Deleting a charge the freight split does not touch still re-resolves every
  // split; an amount split that stops adding up is refused with a named field,
  // and the biller can convert it to percentages and try again.
  it("offers to convert a stale amount split and retries with the flag", async () => {
    const user = userEvent.setup();
    updateCharges
      .mockRejectedValueOnce(
        new ApiRequestError(422, {
          type: "https://api.trenova.app/problems/validation-error",
          title: "Validation failed",
          status: 422,
          errors: [
            {
              field: "convertAmountSplitsToPercent",
              code: "INVALID_OPERATION",
              message:
                "The freight split no longer adds up: the payer amounts total 2850.00 but the charge is now 3000.00. Convert the split to percentages to keep each payer's proportion, or change the split on the shipment.",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Delete Detention Fee" }));

    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(/The freight split no longer adds up/)).toBeInTheDocument();
    expect(updateCharges).toHaveBeenCalledTimes(1);
    const firstPayload = updateCharges.mock.calls[0][1];
    expect(firstPayload.convertAmountSplitsToPercent).toBeUndefined();

    await user.click(within(dialog).getByRole("button", { name: "Convert to percentages" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(2));
    expect(updateCharges.mock.calls[1]).toEqual([
      "bqi_acme",
      { ...firstPayload, convertAmountSplitsToPercent: true },
    ]);
  });

  it("does not offer charge-payer changes once the item is approved", () => {
    renderTab(queueItem({ payer: "acme", status: "Approved" }));

    expect(screen.queryByRole("button", { name: /Change payer/ })).not.toBeInTheDocument();
  });

  it("opens the reassignment for a charge another payer owes", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak", status: "ReadyForReview" }));

    await user.click(screen.getByRole("button", { name: /Billed to other payers/ }));
    await user.click(await screen.findByRole("button", { name: "Change payer for Detention Fee" }));

    expect(await screen.findByText("Reassigning Detention Fee")).toBeInTheDocument();
  });
});
