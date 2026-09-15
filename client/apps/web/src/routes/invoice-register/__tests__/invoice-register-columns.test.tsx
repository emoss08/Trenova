import { translate } from "@trenova/shared/i18n/runtime";
import { cleanup, render, screen } from "@testing-library/react";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import type { InvoiceRegisterRow } from "@/lib/graphql/invoice-table";
import { getColumns } from "../_components/invoice-register-columns";

afterEach(cleanup);

const NOW = 1_789_325_147;

function row(overrides: Partial<InvoiceRegisterRow> = {}): InvoiceRegisterRow {
  return {
    id: "inv_1",
    billingQueueItemId: "bqi_1",
    shipmentId: "shp_1",
    orderId: null,
    customerId: "cus_amd",
    shipperCustomerId: "cus_intel",
    isSplitBill: true,
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
    billToCode: "AMD",
    subtotalAmount: "0",
    otherAmount: "400",
    totalAmount: "400",
    appliedAmount: "0",
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    isAdjustmentArtifact: false,
    version: 1,
    createdAt: NOW,
    updatedAt: NOW,
    openBalance: "400",
    daysPastDue: 45,
    ediSendStatus: "NotSent",
    customer: { id: "cus_amd", name: "AMD", code: "AMD" },
    shipperCustomer: { id: "cus_intel", name: "Intel", code: "INTEL" },
    ...overrides,
  } as InvoiceRegisterRow;
}

function renderCell(
  column: ColumnDef<InvoiceRegisterRow> | undefined,
  original: InvoiceRegisterRow,
) {
  const cell = column?.cell;
  if (typeof cell !== "function") throw new Error("column has no cell renderer");
  return render(<MemoryRouter>{cell({ row: { original } } as never)}</MemoryRouter>);
}

describe("invoice register columns", () => {
  const columns = getColumns(translate);
  const byId = (id: string) => columns.find((column) => column.id === id);

  it("lays the register out as an AR clerk reads it", () => {
    expect(columns.map((column) => column.id)).toEqual([
      "number",
      "billType",
      "status",
      "billTo",
      "shipper",
      "invoiceDate",
      "dueDate",
      "totalAmount",
      "openBalance",
      "daysPastDue",
      "settlementStatus",
      "disputeStatus",
      "ediSendStatus",
      "scope",
      "split",
    ]);
  });

  it("shows what is still owed and how overdue it is, filtering on the stored balance", () => {
    renderCell(byId("openBalance"), row());
    expect(screen.getByText("$400.00")).toBeInTheDocument();
    expect(byId("openBalance")?.meta?.apiField).toBe("balanceDueMinor");
    cleanup();

    renderCell(byId("daysPastDue"), row());
    expect(screen.getByText("45d overdue")).toBeInTheDocument();
    expect(byId("daysPastDue")?.meta?.apiField).toBe("dueDate");
    cleanup();

    renderCell(byId("daysPastDue"), row({ daysPastDue: null, openBalance: "0" }));
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("reads a voided invoice, its dispute, and where its EDI 210 stands", () => {
    renderCell(byId("status"), row({ status: "Voided" }));
    expect(screen.getByText("Voided")).toBeInTheDocument();
    cleanup();

    renderCell(byId("disputeStatus"), row({ disputeStatus: "Disputed" }));
    expect(screen.getByText("Disputed")).toBeInTheDocument();
    cleanup();

    renderCell(byId("ediSendStatus"), row({ ediSendStatus: "DeadLettered" }));
    expect(screen.getByText("Dead-lettered")).toBeInTheDocument();
  });

  it("filters the bill-to by name and the dates by range", () => {
    expect(byId("billTo")?.meta?.apiField).toBe("billToName");
    expect(byId("billTo")?.meta?.filterType).toBe("text");
    expect(byId("invoiceDate")?.meta?.filterType).toBe("date");
    expect(byId("dueDate")?.meta?.filterType).toBe("date");
    expect(byId("status")?.meta?.filterType).toBe("select");
  });

  it("names the shipper only when the invoice bills someone else's freight", () => {
    renderCell(byId("shipper"), row());
    expect(screen.getByText("Intel")).toBeInTheDocument();
    cleanup();

    const { container } = renderCell(
      byId("shipper"),
      row({
        shipperCustomerId: "cus_amd",
        shipperCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
      }),
    );
    expect(container).toHaveTextContent("—");
  });

  it("marks a split bill and links the number to the invoice workspace", () => {
    renderCell(byId("split"), row());
    expect(screen.getByText("Split bill")).toBeInTheDocument();
    cleanup();

    renderCell(byId("number"), row());
    expect(screen.getByRole("link", { name: "INV-1" })).toHaveAttribute(
      "href",
      "/billing/invoices?item=inv_1",
    );
  });
});
