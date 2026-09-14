import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
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
    description: "Freight charge",
    quantity: 1,
    unitPrice: 100,
    amount: 100,
    accessorialChargeId: null,
    chargeCode: null,
    chargeMethod: null,
    rateUnit: null,
    rate: null,
    rateBasisAmount: null,
    formulaTemplateName: null,
    ...overrides,
  } as InvoiceLine;
}

function accessorial(overrides: Partial<InvoiceLine> & Pick<InvoiceLine, "id" | "lineNumber">) {
  return line({ type: "Accessorial", description: "Accessorial charge", ...overrides });
}

function sumOf(lines: InvoiceLine[], type?: InvoiceLine["type"]) {
  return lines
    .filter((l) => type === undefined || l.type === type)
    .reduce((sum, l) => sum + (l.amount ?? 0), 0);
}

function invoiceWith(lines: InvoiceLine[], overrides: Partial<Invoice> = {}): Invoice {
  return {
    id: "inv_1",
    billingQueueItemId: "bqi_1",
    scope: "Shipment",
    currencyCode: "USD",
    detail: "Detailed",
    subtotalAmount: sumOf(lines, "Freight"),
    otherAmount: sumOf(lines, "Accessorial"),
    totalAmount: sumOf(lines),
    lines,
    ...overrides,
  } as unknown as Invoice;
}

function renderTab(invoice: Invoice) {
  return render(
    <MemoryRouter>
      <InvoiceChargesTab invoice={invoice} />
    </MemoryRouter>,
  );
}

function sectionHeading(heading: RegExp) {
  return screen
    .getByRole("button", { name: heading })
    .closest("[data-section-heading]") as HTMLElement;
}

function chargeRow(label: string) {
  return screen.getByText(label).closest("li") as HTMLElement;
}

afterEach(cleanup);

describe("InvoiceChargesTab", () => {
  it("lists a single shipment's charges with no section or expand controls", () => {
    renderTab(
      invoiceWith([
        line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
        accessorial({
          id: "l2",
          lineNumber: 2,
          shipmentId: "shp_1",
          shipmentProNumber: "PRO-1",
          description: "Detention",
          amount: 50,
          unitPrice: 50,
        }),
      ]),
    );

    expect(screen.queryByRole("button", { expanded: true })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /expand all|collapse all/i }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("PRO-1")).not.toBeInTheDocument();
    expect(screen.getByText("Line Haul")).toBeInTheDocument();
    expect(screen.getByText("Detention")).toBeInTheDocument();
    expect(screen.getByText("2 charges")).toBeInTheDocument();
  });

  it("names each accessorial with its code and how its amount was reached", () => {
    renderTab(
      invoiceWith([
        line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", amount: 2450, unitPrice: 2450 }),
        accessorial({
          id: "l2",
          lineNumber: 2,
          shipmentId: "shp_1",
          description: "Detention",
          chargeCode: "DET",
          chargeMethod: "PerUnit",
          rateUnit: "Hour",
          rate: 75,
          quantity: 2,
          unitPrice: 75,
          amount: 150,
        }),
        accessorial({
          id: "l3",
          lineNumber: 3,
          shipmentId: "shp_1",
          description: "Fuel Surcharge",
          chargeCode: "FSC",
          chargeMethod: "Percentage",
          rate: 10,
          rateBasisAmount: 2450,
          amount: 245,
          unitPrice: 245,
        }),
        accessorial({
          id: "l4",
          lineNumber: 4,
          shipmentId: "shp_1",
          description: "Lumper Fee",
          chargeCode: "LMP",
          chargeMethod: "Flat",
          rate: 150,
          amount: 150,
          unitPrice: 150,
        }),
      ]),
    );

    const accessorials = screen.getByRole("list", { name: "Accessorials" });
    expect(within(accessorials).getAllByRole("listitem")).toHaveLength(3);

    const detention = chargeRow("Detention");
    expect(within(detention).getByText("DET")).toBeInTheDocument();
    expect(within(detention).getByText("$75.00 × 2 hours · Line 2")).toBeInTheDocument();
    expect(within(detention).getByText("$150.00")).toBeInTheDocument();

    const fuel = chargeRow("Fuel Surcharge");
    expect(within(fuel).getByText("FSC")).toBeInTheDocument();
    expect(within(fuel).getByText("10% of line haul ($2,450.00) · Line 3")).toBeInTheDocument();

    const lumper = chargeRow("Lumper Fee");
    expect(within(lumper).getByText("Flat rate · Line 4")).toBeInTheDocument();

    expect(screen.getByText("Subtotal").parentElement).toHaveTextContent("$545.00");
  });

  it("shows the rating formula and base rate the line haul was priced from", () => {
    renderTab(
      invoiceWith([
        line({
          id: "l1",
          lineNumber: 1,
          shipmentId: "shp_1",
          amount: 2450,
          unitPrice: 2450,
          rate: 3.5,
          formulaTemplateName: "Per Mile",
        }),
      ]),
    );

    expect(screen.getByText("Rating:").parentElement).toHaveTextContent("Per Mile");
    expect(within(chargeRow("Base Rate")).getByText("$3.50")).toBeInTheDocument();
    expect(within(chargeRow("Line Haul")).getByText("$2,450.00")).toBeInTheDocument();
  });

  it("leaves out the rating and base rate when the freight line recorded neither", () => {
    renderTab(invoiceWith([line({ id: "l1", lineNumber: 1, shipmentId: "shp_1" })]));

    expect(screen.queryByText("Rating:")).not.toBeInTheDocument();
    expect(screen.queryByText("Base Rate")).not.toBeInTheDocument();
  });

  it("says when a shipment carries no accessorial charges", () => {
    renderTab(invoiceWith([line({ id: "l1", lineNumber: 1, shipmentId: "shp_1" })]));

    expect(screen.getByText("No accessorial charges")).toBeInTheDocument();
    expect(screen.queryByText("Subtotal")).not.toBeInTheDocument();
  });

  it("keeps the line number the customer's PDF prints beside each charge", () => {
    renderTab(invoiceWith([line({ id: "l1", lineNumber: 7, shipmentId: "shp_1" })]));

    expect(within(chargeRow("Line Haul")).getByText("Line 7")).toBeInTheDocument();
  });

  it("puts each shipment's subtotal on its section heading", () => {
    renderTab(
      invoiceWith([
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
      ]),
    );

    expect(screen.getAllByRole("button", { expanded: true })).toHaveLength(3);
    expect(within(sectionHeading(/PRO-1/)).getByText("$100.00")).toBeInTheDocument();
    expect(within(sectionHeading(/PRO-2/)).getByText("$250.00")).toBeInTheDocument();
    expect(within(sectionHeading(/PRO-3/)).getByText("$400.00")).toBeInTheDocument();
    expect(screen.getByText("3 charges across 3 shipments")).toBeInTheDocument();
  });

  it("links each shipment section to the shipment", () => {
    renderTab(
      invoiceWith([
        line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
        line({ id: "l2", lineNumber: 2, shipmentId: "shp_2", shipmentProNumber: "PRO-2" }),
      ]),
    );

    expect(screen.getByRole("link", { name: "Open shipment PRO-1" })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
  });

  it("gives unattributed order-level charges their own trailing section with no shipment link", () => {
    renderTab(
      invoiceWith([
        line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
        accessorial({
          id: "oc",
          lineNumber: 2,
          description: "Customs brokerage",
          amount: 75,
          unitPrice: 75,
        }),
      ]),
    );

    const orderHeading = sectionHeading(/Order charges/);
    expect(within(orderHeading).getByText("$75.00")).toBeInTheDocument();
    expect(within(orderHeading).queryByRole("link")).not.toBeInTheDocument();
    expect(
      within(screen.getByRole("list", { name: "Order charges" })).getByText("Customs brokerage"),
    ).toBeInTheDocument();
  });

  it("collapses a section to its heading without losing its subtotal", async () => {
    const user = userEvent.setup();
    renderTab(
      invoiceWith([
        line({
          id: "l1",
          lineNumber: 1,
          shipmentId: "shp_1",
          shipmentProNumber: "PRO-1",
          amount: 100,
        }),
        accessorial({
          id: "l2",
          lineNumber: 2,
          shipmentId: "shp_1",
          shipmentProNumber: "PRO-1",
          description: "Detention A",
          amount: 25.5,
          unitPrice: 25.5,
        }),
        line({ id: "l3", lineNumber: 3, shipmentId: "shp_2", shipmentProNumber: "PRO-2" }),
        accessorial({
          id: "l4",
          lineNumber: 4,
          shipmentId: "shp_2",
          shipmentProNumber: "PRO-2",
          description: "Detention B",
        }),
      ]),
    );

    const toggle = screen.getByRole("button", { name: /PRO-1/ });
    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Detention A")).not.toBeInTheDocument();
    expect(within(sectionHeading(/PRO-1/)).getByText("$125.50")).toBeInTheDocument();
    expect(screen.getByText("Detention B")).toBeInTheDocument();
  });

  it("expands and collapses every section at once", async () => {
    const user = userEvent.setup();
    renderTab(
      invoiceWith([
        line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
        line({ id: "l2", lineNumber: 2, shipmentId: "shp_2", shipmentProNumber: "PRO-2" }),
        accessorial({ id: "oc", lineNumber: 3, description: "Customs brokerage" }),
      ]),
    );

    await user.click(screen.getByRole("button", { name: "Collapse all" }));
    expect(screen.queryAllByRole("button", { expanded: true })).toHaveLength(0);
    expect(screen.getAllByRole("button", { expanded: false })).toHaveLength(3);

    // One open section is enough for the control to offer opening the rest.
    await user.click(screen.getByRole("button", { name: /PRO-2/ }));
    await user.click(screen.getByRole("button", { name: "Expand all" }));
    expect(screen.getAllByRole("button", { expanded: true })).toHaveLength(3);
    expect(screen.getByRole("button", { name: "Collapse all" })).toBeInTheDocument();
  });

  it("opens only the first section when an invoice carries more than eight shipments", () => {
    const lines = Array.from({ length: 9 }, (_, index) =>
      line({
        id: `l${index}`,
        lineNumber: index + 1,
        shipmentId: `shp_${index}`,
        shipmentProNumber: `PRO-${index}`,
      }),
    );
    renderTab(invoiceWith(lines));

    const expanded = screen.getAllByRole("button", { expanded: true });
    expect(expanded).toHaveLength(1);
    expect(expanded[0]).toHaveAccessibleName(/PRO-0/);
  });

  it("starts a Summary customer's sections closed and says what their copy shows", async () => {
    const user = userEvent.setup();
    const lines = [
      line({
        id: "l1",
        lineNumber: 1,
        shipmentId: "shp_1",
        shipmentProNumber: "PRO-1",
        amount: 100,
      }),
      accessorial({
        id: "l2",
        lineNumber: 2,
        shipmentId: "shp_1",
        shipmentProNumber: "PRO-1",
        description: "Detention",
        amount: 50,
        unitPrice: 50,
      }),
      line({
        id: "l3",
        lineNumber: 3,
        shipmentId: "shp_2",
        shipmentProNumber: "PRO-2",
        amount: 250,
      }),
    ];

    renderTab(invoiceWith(lines, { detail: "Summary" }));

    expect(screen.getByText("Customer copy lists one line per shipment")).toBeInTheDocument();
    expect(screen.queryByText("Detention")).not.toBeInTheDocument();
    expect(within(sectionHeading(/PRO-1/)).getByText("2 charges")).toBeInTheDocument();
    expect(within(sectionHeading(/PRO-1/)).getByText("$150.00")).toBeInTheDocument();

    // The breakdown is still on the invoice; the biller can open it.
    await user.click(screen.getByRole("button", { name: /PRO-1/ }));
    expect(screen.getByText("Detention")).toBeInTheDocument();
  });

  it("shows every charge line for a Detailed customer", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
      accessorial({
        id: "l2",
        lineNumber: 2,
        shipmentId: "shp_2",
        shipmentProNumber: "PRO-2",
        description: "Detention",
      }),
    ];

    renderTab(invoiceWith(lines, { detail: "Detailed" }));

    expect(screen.getByText("Customer copy lists every charge")).toBeInTheDocument();
    expect(screen.getByText("Detention")).toBeInTheDocument();
  });

  it("sums each section from its own lines, not from the invoice header", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", amount: 125.5 }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_2", amount: 74.5 }),
    ];

    renderTab(invoiceWith(lines, { subtotalAmount: 9999 } as Partial<Invoice>));

    expect(within(sectionHeading(/shp_1/)).getByText("$125.50")).toBeInTheDocument();
  });

  it("totals freight, accessorials and the invoice from the header amounts they name", () => {
    const lines = [
      line({ id: "l1", lineNumber: 1, amount: 100 }),
      accessorial({ id: "l2", lineNumber: 2, description: "Detention", amount: 40 }),
    ];

    // Header values that no line sums to, so the footer can only have read them.
    renderTab(
      invoiceWith(lines, {
        subtotalAmount: 1200,
        otherAmount: 345.67,
        totalAmount: 1545.67,
      } as Partial<Invoice>),
    );

    const totals = screen.getByRole("region", { name: "Invoice totals" });
    expect(
      within(within(totals).getByText("Freight").parentElement as HTMLElement).getByText(
        "$1,200.00",
      ),
    ).toBeInTheDocument();
    expect(
      within(within(totals).getByText("Accessorials").parentElement as HTMLElement).getByText(
        "$345.67",
      ),
    ).toBeInTheDocument();
    expect(
      within(within(totals).getByText("Total").parentElement as HTMLElement).getByText("$1,545.67"),
    ).toBeInTheDocument();
  });

  it("falls back to quantity and unit price for lines written before the charge method was recorded", () => {
    renderTab(
      invoiceWith([
        line({ id: "flat", lineNumber: 1, shipmentId: "shp_1", unitPrice: 900, amount: 900 }),
        accessorial({
          id: "units",
          lineNumber: 2,
          shipmentId: "shp_1",
          description: "Detention",
          quantity: 3,
          unitPrice: 50,
          amount: 150,
        }),
        accessorial({
          id: "single",
          lineNumber: 3,
          shipmentId: "shp_1",
          description: "Accessorial charge",
          quantity: 1,
          unitPrice: 75,
          amount: 75,
        }),
      ]),
    );

    expect(within(chargeRow("Detention")).getByText("3 × $50.00 · Line 2")).toBeInTheDocument();
    expect(within(chargeRow("Accessorial charge")).getByText("Line 3")).toBeInTheDocument();
  });

  it("points an invoice with no charges back to the queue item it came from", () => {
    renderTab(invoiceWith([]));

    expect(screen.getByRole("heading", { name: "No charges on this invoice" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open queue item" })).toHaveAttribute(
      "href",
      "/billing/queue?item=bqi_1&includePosted=true",
    );
    expect(screen.queryByRole("region", { name: "Invoice totals" })).not.toBeInTheDocument();
  });

  it("does not offer a consolidated invoice's anchor queue item as where its charges came from", () => {
    renderTab(invoiceWith([], { scope: "Consolidated" }));

    expect(screen.getByRole("heading", { name: "No charges on this invoice" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Open queue item" })).not.toBeInTheDocument();
  });

  it("opens the next invoice on its own default sections, not the previous invoice's", async () => {
    const user = userEvent.setup();
    const shipmentLines = [
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_2", shipmentProNumber: "PRO-2" }),
    ];
    const { rerender } = renderTab(invoiceWith(shipmentLines));

    await user.click(screen.getByRole("button", { name: "Collapse all" }));
    expect(screen.queryAllByRole("button", { expanded: true })).toHaveLength(0);

    // Same shipments on a different invoice, as a corrected artifact would carry.
    rerender(
      <MemoryRouter>
        <InvoiceChargesTab invoice={invoiceWith(shipmentLines, { id: "inv_2" })} />
      </MemoryRouter>,
    );

    expect(screen.getAllByRole("button", { expanded: true })).toHaveLength(2);
  });
});
