import { cleanup, render, screen } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { BillingCell } from "./billing-cell";

afterEach(cleanup);

function renderCell(overrides: Partial<Shipment>) {
  return render(
    <BillingCell shipment={{ id: "shp_1", status: "ReadyToInvoice", ...overrides } as Shipment} />,
  );
}

describe("BillingCell", () => {
  it.each([
    ["ReadyForReview", "Ready for review"],
    ["InReview", "In review"],
    ["Approved", "Approved"],
    ["Posted", "Posted"],
    ["OnHold", "On hold"],
    ["SentBackToOps", "Sent back to ops"],
    ["Exception", "Exception"],
    ["Canceled", "Canceled"],
  ] as const)("labels %s as %s", (billingTransferStatus, label) => {
    renderCell({ billingTransferStatus });

    expect(screen.getByText(label)).toBeInTheDocument();
    expect(screen.getByTitle(`Billing queue: ${label}`)).toBeInTheDocument();
  });

  it("shows a dash for a shipment billing has never received", () => {
    renderCell({ billingTransferStatus: null });

    expect(screen.getByText("—")).toBeInTheDocument();
    expect(document.querySelector("[title^='Billing queue']")).toBeNull();
  });
});
