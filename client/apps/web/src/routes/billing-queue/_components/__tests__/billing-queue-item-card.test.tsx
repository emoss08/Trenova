import { cleanup, render, screen } from "@testing-library/react";
import type { BillingQueueRow } from "@/lib/graphql/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueItemCard, rowNote } from "../billing-queue-item-card";

afterEach(cleanup);

const NOW = Math.floor(Date.now() / 1000);

function item(overrides: Partial<BillingQueueRow> = {}): BillingQueueRow {
  return {
    id: "bqi_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    billToCustomerId: "cus_intel",
    allocatedTotalAmount: null,
    number: "BQ-1",
    status: "ReadyForReview",
    lane: "NeedsYou",
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
  } as unknown as BillingQueueRow;
}

function renderCard(value: BillingQueueRow, isSelected = false) {
  return render(
    <BillingQueueItemCard
      item={value}
      isSelected={isSelected}
      onClick={vi.fn()}
      onAssignBiller={vi.fn()}
      onHold={vi.fn()}
      onCancel={vi.fn()}
    />,
  );
}

// The list lights items as the module rail lights its tiles: hover is the list's glide, and
// the open item is filled as the rail fills the module you are in.
describe("BillingQueueItemCard in the list", () => {
  it("fills the open item as the module rail fills the module you are in", () => {
    const { container } = renderCard(item(), true);

    const card = container.querySelector("[data-glide-target]");
    expect(card).toHaveClass("bg-nav-active", "text-nav-active-foreground");
    expect(screen.getByRole("button", { current: true })).toBeInTheDocument();
  });

  it("leaves a closed item unfilled, with no hover fill of its own, for the list's glide", () => {
    const { container } = renderCard(item());

    const card = container.querySelector("[data-glide-target]");
    expect(card).not.toHaveClass("bg-nav-active");
    expect(card?.className).not.toMatch(/hover:bg-/u);
  });
});

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
        } as BillingQueueRow["billToCustomer"],
        allocatedTotalAmount: "400",
      }),
    );

    expect(screen.getByText("$400.00")).toBeInTheDocument();
    expect(screen.queryByText("$1,000.00")).not.toBeInTheDocument();
    expect(screen.getByText("AMD")).toBeInTheDocument();
    expect(screen.getByText(/on behalf of Intel/i)).toBeInTheDocument();
    expect(screen.getByText("Split")).toBeInTheDocument();
  });
});

describe("rowNote", () => {
  const t = ((text: string, ...args: unknown[]) =>
    args.reduce<string>(
      (out, arg, index) => out.replace(`{${index}}`, String(arg)),
      text,
    )) as unknown as Parameters<typeof rowNote>[1];

  it("leads a row that needs the biller with its open issue and how many more wait", () => {
    const note = rowNote(
      item({
        lane: "NeedsYou",
        headline: {
          code: "PODMissing",
          checkKey: "pod",
          summary: "Signed POD missing",
          open: 3,
          waiting: false,
        },
      }),
      t,
    );

    expect(note).toEqual({ text: "Signed POD missing · 2 more", tone: "warning" });
  });

  it("says nobody is assigned when that is all a needs-you row waits on", () => {
    expect(rowNote(item({ lane: "NeedsYou", headline: null, assignedBillerId: null }), t)).toEqual({
      text: "Nobody is assigned",
      tone: "warning",
    });
  });

  it("leads a ready row with Nova's concern over its agreement", () => {
    expect(
      rowNote(
        item({
          lane: "Ready",
          agentVerdict: "Concern",
          agentSummary: "Fuel is 18% above this lane's usual",
        }),
        t,
      ),
    ).toEqual({ text: "Fuel is 18% above this lane's usual", tone: "concern" });
    expect(
      rowNote(
        item({ lane: "Ready", agentVerdict: "Agrees", agentSummary: "Matches the contract" }),
        t,
      ),
    ).toEqual({ text: "Matches the contract", tone: "nova" });
  });

  it("reads plainly for a ready row Nova agreed with and had nothing to add to", () => {
    expect(rowNote(item({ lane: "Ready", agentVerdict: "Agrees", agentSummary: "" }), t)).toEqual({
      text: "All five checks clear",
      tone: "muted",
    });
  });

  it("reads plainly for a ready row Nova never reviewed", () => {
    expect(rowNote(item({ lane: "Ready", agentVerdict: null, agentSummary: null }), t)).toEqual({
      text: "All five checks clear",
      tone: "muted",
    });
  });

  it("names why a held row is held", () => {
    expect(
      rowNote(item({ lane: "Held", status: "OnHold", holdReasonCode: "CustomerDispute" }), t),
    ).toEqual({ text: "On hold · Customer dispute", tone: "muted" });
  });
});
