import type { InvoiceTableRowFieldsFragment } from "@trenova/graphql/generated/graphql";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceItemCard } from "../invoice-item-card";

afterEach(cleanup);

const NOW = 1_789_325_147;

function row(
  overrides: Partial<InvoiceTableRowFieldsFragment> = {},
): InvoiceTableRowFieldsFragment {
  return {
    id: "inv_1",
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    orderId: null,
    customerId: "cus_1",
    shipperCustomerId: null,
    isSplitBill: false,
    number: "INV-1",
    billType: "Invoice",
    scope: "Shipment",
    periodStart: null,
    periodEnd: null,
    shipmentCount: 1,
    status: "Posted",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: NOW,
    dueDate: NOW,
    billToName: "AMD",
    subtotalAmount: "100",
    otherAmount: "0",
    totalAmount: "100",
    appliedAmount: "0",
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    isAdjustmentArtifact: false,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    customer: { id: "cus_1", name: "AMD", code: "AMD" },
    ...overrides,
  } as InvoiceTableRowFieldsFragment;
}

function renderCard(invoice: InvoiceTableRowFieldsFragment, onPost = vi.fn()) {
  render(
    <MemoryRouter>
      <InvoiceItemCard invoice={invoice} isSelected={false} onClick={vi.fn()} onPost={onPost} />
    </MemoryRouter>,
  );
  return onPost;
}

describe("InvoiceItemCard for voided and disputed invoices", () => {
  it("reads Voided and drops the settlement badge, since nothing is owed", () => {
    renderCard(row({ status: "Voided" }));

    expect(screen.getByText("Voided")).toBeInTheDocument();
    expect(screen.queryByText("Unpaid")).toBeNull();
  });

  it("flags a disputed invoice in the list", () => {
    renderCard(row({ disputeStatus: "Disputed" }));

    expect(screen.getByText("Disputed")).toBeInTheDocument();
  });

  it("will not offer to post anything but a draft", async () => {
    const user = userEvent.setup();
    renderCard(row({ status: "Voided" }));

    await user.pointer({ keys: "[MouseRight]", target: screen.getByText("INV-1") });

    expect(await screen.findByRole("menuitem", { name: "Post Invoice" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });
});
