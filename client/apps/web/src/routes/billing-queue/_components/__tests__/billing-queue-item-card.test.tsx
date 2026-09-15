import { cleanup, render, screen } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueItemCard } from "../billing-queue-item-card";

afterEach(cleanup);

const NOW = Math.floor(Date.now() / 1000);

function item(overrides: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    billToCustomerId: "cus_intel",
    allocatedTotalAmount: null,
    number: "BQ-1",
    status: "ReadyForReview",
    billType: "Invoice",
    isAdjustmentOrigin: false,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    shipment: {
      id: "shp_1",
      proNumber: "PRO-1",
      totalChargeAmount: 1000,
      customerId: "cus_intel",
      customer: { id: "cus_intel", name: "Intel", code: "INTEL" },
    },
    billToCustomer: { id: "cus_intel", name: "Intel", code: "INTEL" },
    ...overrides,
  } as unknown as BillingQueueItem;
}

function renderCard(value: BillingQueueItem) {
  return render(
    <BillingQueueItemCard
      item={value}
      isSelected={false}
      onClick={vi.fn()}
      onAssignBiller={vi.fn()}
      onHold={vi.fn()}
      onCancel={vi.fn()}
    />,
  );
}

describe("BillingQueueItemCard payer", () => {
  it("bills the shipment's own customer for the whole amount by default", () => {
    renderCard(item());

    expect(screen.getByText("$1,000.00")).toBeInTheDocument();
    expect(screen.getByText("Intel")).toBeInTheDocument();
    expect(screen.queryByText(/on behalf of/i)).not.toBeInTheDocument();
    expect(screen.queryByText("Split")).not.toBeInTheDocument();
  });

  // A split shipment has one queue item per payer. The card has to show the
  // payer's own amount, not the shipment total, or two cards for one shipment
  // both read as billing the full freight.
  it("shows the payer's allocated share and whose freight it is", () => {
    renderCard(
      item({
        billToCustomerId: "cus_amd",
        billToCustomer: {
          id: "cus_amd",
          name: "AMD",
          code: "AMD",
        } as BillingQueueItem["billToCustomer"],
        allocatedTotalAmount: "400.00",
      }),
    );

    expect(screen.getByText("$400.00")).toBeInTheDocument();
    expect(screen.queryByText("$1,000.00")).not.toBeInTheDocument();
    expect(screen.getByText("AMD")).toBeInTheDocument();
    expect(screen.getByText(/on behalf of Intel/i)).toBeInTheDocument();
    expect(screen.getByText("Split")).toBeInTheDocument();
  });
});
