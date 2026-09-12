import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import type { Invoice, InvoiceLine } from "@trenova/shared/types/invoice";
import { InvoiceChargesTab } from "../invoice-charges-tab";

function line(overrides: Partial<InvoiceLine> & Pick<InvoiceLine, "id" | "lineNumber">) {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    shipmentId: null,
    shipmentProNumber: null,
    shipmentBol: null,
    type: "Freight",
    description: "Linehaul",
    quantity: 1,
    unitPrice: 100,
    amount: 100,
    ...overrides,
  } as InvoiceLine;
}

function invoiceWith(lines: InvoiceLine[]): Invoice {
  return {
    currencyCode: "USD",
    subtotalAmount: lines.reduce((sum, l) => sum + (l.amount ?? 0), 0),
    otherAmount: 0,
    totalAmount: lines.reduce((sum, l) => sum + (l.amount ?? 0), 0),
    lines,
  } as unknown as Invoice;
}

afterEach(cleanup);

describe("InvoiceChargesTab", () => {
  it("renders a flat table with no group chrome for a single-shipment invoice", () => {
    render(
      <InvoiceChargesTab
        invoice={invoiceWith([
          line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
          line({ id: "l2", lineNumber: 2, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
        ])}
      />,
    );

    // No collapse control and no per-shipment subtotal row. The footer's own
    // "Subtotal" must survive, so match only the "<heading> subtotal" form.
    expect(screen.queryByRole("button", { expanded: true })).not.toBeInTheDocument();
    expect(screen.queryByText("PRO-1")).not.toBeInTheDocument();
    expect(screen.queryByText(/\S+ subtotal$/i)).not.toBeInTheDocument();
    expect(screen.getByText("Subtotal")).toBeInTheDocument();
    expect(screen.getByText("Total")).toBeInTheDocument();
  });

  it("renders a subheader and a subtotal per shipment when several are billed", () => {
    render(
      <InvoiceChargesTab
        invoice={invoiceWith([
          line({
            id: "l1",
            lineNumber: 1,
            shipmentId: "shp_1",
            shipmentProNumber: "PRO-1",
            amount: 100,
          }),
          line({
            id: "l2",
            lineNumber: 2,
            shipmentId: "shp_2",
            shipmentProNumber: "PRO-2",
            amount: 250,
          }),
          line({
            id: "l3",
            lineNumber: 3,
            shipmentId: "shp_3",
            shipmentProNumber: "PRO-3",
            amount: 400,
          }),
        ])}
      />,
    );

    expect(screen.getByText("PRO-1")).toBeInTheDocument();
    expect(screen.getByText("PRO-2")).toBeInTheDocument();
    expect(screen.getByText("PRO-3")).toBeInTheDocument();

    expect(screen.getByText("PRO-1 subtotal")).toBeInTheDocument();
    expect(screen.getByText("PRO-2 subtotal")).toBeInTheDocument();
    expect(screen.getByText("PRO-3 subtotal")).toBeInTheDocument();

    expect(screen.getAllByRole("button", { expanded: true })).toHaveLength(3);
  });

  it("gives unattributed order-level charges their own trailing section", () => {
    render(
      <InvoiceChargesTab
        invoice={invoiceWith([
          line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
          line({
            id: "oc",
            lineNumber: 2,
            shipmentId: null,
            type: "Accessorial",
            description: "Customs brokerage",
          }),
        ])}
      />,
    );

    expect(screen.getByText("Order charges")).toBeInTheDocument();
    expect(screen.getByText("Order charges subtotal")).toBeInTheDocument();
    expect(screen.getByText("Customs brokerage")).toBeInTheDocument();
  });

  it("collapses a shipment section without losing its subtotal", async () => {
    const user = userEvent.setup();
    render(
      <InvoiceChargesTab
        invoice={invoiceWith([
          line({
            id: "l1",
            lineNumber: 1,
            shipmentId: "shp_1",
            shipmentProNumber: "PRO-1",
            description: "Linehaul A",
          }),
          line({
            id: "l2",
            lineNumber: 2,
            shipmentId: "shp_2",
            shipmentProNumber: "PRO-2",
            description: "Linehaul B",
          }),
        ])}
      />,
    );

    expect(screen.getByText("Linehaul A")).toBeInTheDocument();

    const toggle = screen.getByRole("button", { name: /PRO-1/ });
    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Linehaul A")).not.toBeInTheDocument();
    // The subtotal stays visible so a collapsed section still reconciles.
    expect(screen.getByText("PRO-1 subtotal")).toBeInTheDocument();
    expect(screen.getByText("Linehaul B")).toBeInTheDocument();
  });

  it("collapses to one row per shipment when the customer is on Summary", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1", amount: 100 }),
      line({
        id: "l2",
        lineNumber: 2,
        shipmentId: "shp_1",
        shipmentProNumber: "PRO-1",
        type: "Accessorial",
        description: "Detention",
        amount: 50,
      }),
      line({ id: "l3", lineNumber: 3, shipmentId: "shp_2", shipmentProNumber: "PRO-2", amount: 250 }),
    ];
    const invoice = { ...invoiceWith(lines), detail: "Summary" } as Invoice;

    render(<InvoiceChargesTab invoice={invoice} />);

    // The individual charge descriptions are not shown...
    expect(screen.queryByText("Detention")).not.toBeInTheDocument();
    // ...but every shipment is, with its own total.
    expect(screen.getByText("PRO-1")).toBeInTheDocument();
    expect(screen.getByText("PRO-2")).toBeInTheDocument();
    expect(screen.getByText("2 charges")).toBeInTheDocument();
    expect(screen.getByText("$150.00")).toBeInTheDocument();
  });

  it("shows every charge line when the customer is on Detailed", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
      line({
        id: "l2",
        lineNumber: 2,
        shipmentId: "shp_2",
        shipmentProNumber: "PRO-2",
        type: "Accessorial",
        description: "Detention",
      }),
    ];
    const invoice = { ...invoiceWith(lines), detail: "Detailed" } as Invoice;

    render(<InvoiceChargesTab invoice={invoice} />);

    expect(screen.getByText("Detention")).toBeInTheDocument();
  });

  it("sums each section from its own lines, not from the invoice header", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", amount: 125.5 }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_2", amount: 74.5 }),
    ];
    const invoice = {
      ...invoiceWith(lines),
      // Deliberately disagrees with the lines.
      subtotalAmount: 9999,
    } as Invoice;

    render(<InvoiceChargesTab invoice={invoice} />);

    const first = screen.getByText("shp_1 subtotal").closest("tr");
    expect(within(first as HTMLElement).getByText("$125.50")).toBeInTheDocument();
  });
});
