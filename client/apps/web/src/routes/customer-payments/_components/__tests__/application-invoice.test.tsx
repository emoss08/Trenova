import { clearPermissions, seedPermissions } from "@/test/permissions";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { Operation } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { ApplicationInvoice } from "../application-invoice";

type Invoice = ComponentProps<typeof ApplicationInvoice>["invoice"];

// The invoice an application carries, as CustomerPaymentDetail selects it. The schema types
// the field `invoice: Invoice` (nullable): the server sends null when the reader may not see
// invoices or the invoice cannot be found.
const invoice = {
  id: "inv_01KSJG9AG6XFQ8HDX5309293A9",
  number: "INV-1042",
  invoiceDate: 1_779_811_265,
  dueDate: 1_782_403_265,
  totalAmount: "1250.00",
  settlementStatus: "Unpaid",
  disputeStatus: "None",
  billToName: "Acme Foods",
} as NonNullable<Invoice>;

afterEach(() => {
  cleanup();
  clearPermissions();
});

function renderCell(value: Invoice) {
  return render(
    <TestRouter>
      <ApplicationInvoice invoice={value} />
    </TestRouter>,
  );
}

describe("ApplicationInvoice", () => {
  it("links the invoice by its number and names who it bills", () => {
    seedPermissions({ invoice: Operation.Read });
    renderCell(invoice);

    const link = screen.getByRole("link", { name: "INV-1042" });
    expect(link.getAttribute("href")).toBe(`/billing/invoices?item=${invoice.id}`);
    expect(screen.getByText("Acme Foods")).toBeTruthy();
  });

  it("reads as withheld, never as the invoice's id, for someone who may not read invoices", () => {
    seedPermissions({ customer_payment: Operation.Read });
    const { container } = renderCell(null);

    const reference = container.querySelector('[data-slot="locked-reference"]');
    expect(reference?.textContent).toBe("Hidden");
    expect(reference?.getAttribute("aria-description")).toBe("You don't have access to invoices");
    expect(container.textContent).not.toContain("inv_");
  });

  it("says the invoice cannot be found when the reader may see invoices but none came back", () => {
    seedPermissions({ invoice: Operation.Read });
    const { container } = renderCell(null);

    expect(container.textContent).toBe("Invoice not found");
    expect(container.querySelector('[data-slot="locked-reference"]')).toBeNull();
  });
});
