import { recordPath } from "@/config/record-links";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoicePreview } from "../_components/preview/invoice-preview";
import { isPaletteRecordType } from "../palette-entities";
import { record } from "./palette-test-fixtures";

const { responses } = vi.hoisted(() => ({ responses: new Map<string, unknown>() }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return {
    ...actual,
    requestGraphQL: vi.fn(async ({ operationName }: { operationName: string }) => {
      if (!responses.has(operationName)) {
        throw new Error(`No response for ${operationName}`);
      }
      return responses.get(operationName);
    }),
  };
});

const DAY = 86_400;

/** An invoice as `DeskInvoiceCard` returns it: posted, $1,250.00 open, 12 days past due. */
function deskInvoice(fields: Record<string, unknown> = {}) {
  const now = Math.floor(Date.now() / 1000);
  return {
    invoice: {
      id: "inv_1",
      number: "INV-1042",
      billType: "Invoice",
      status: "Posted",
      scope: "Shipment",
      currencyCode: "USD",
      paymentTerm: "Net30",
      invoiceDate: now - 42 * DAY,
      dueDate: now - 12 * DAY,
      postedAt: now - 42 * DAY,
      periodStart: null,
      periodEnd: null,
      shipmentCount: 1,
      shipmentProNumber: "S26101755026772",
      shipmentBol: "BOL-77",
      orderId: null,
      orderNumber: null,
      billToName: "Morales & Jenkins Manufacturing",
      isSplitBill: false,
      factoringCompanyId: null,
      subtotalAmount: "1250.00",
      otherAmount: "0",
      totalAmount: "1250.00",
      appliedAmount: "0",
      openBalance: "1250.00",
      creditRemaining: "0",
      daysPastDue: 12,
      settlementStatus: "Unpaid",
      disputeStatus: "None",
      sendStatus: "Sent",
      sentAt: now - 42 * DAY,
      voidedAt: null,
      voidReason: null,
      voidDisposition: null,
      memoReason: null,
      memoKind: null,
      ratingNotes: null,
      ediSendStatus: null,
      ediSentAt: null,
      pdfDocumentId: null,
      lastEdiError: null,
      lastEdiMessageId: null,
      createdAt: now - 42 * DAY,
      updatedAt: now - 42 * DAY,
      customer: { id: "cus_1", code: "MORJ", name: "Morales & Jenkins Manufacturing" },
      voidedBy: null,
      referenceInvoice: null,
      relatedInvoices: [],
      chargeSummary: [
        {
          type: "Freight",
          description: "Linehaul",
          chargeCode: "LH",
          chargeMethod: "Flat",
          rateUnit: null,
          rate: "1250.00",
          rateBasisAmount: null,
          quantity: "1",
          formulaTemplateId: null,
          formulaTemplateName: null,
          lineCount: 1,
          shipmentCount: 1,
          amount: "1250.00",
        },
      ],
      paymentApplications: [],
      creditApplications: [],
      disputes: [],
      ediSendPlan: null,
      ...fields,
    },
  };
}

function draw(onNavigate = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <InvoicePreview
        record={record({ entityType: "invoice", id: "inv_1", title: "INV-1042" })}
        onNavigate={onNavigate}
      />
    </QueryClientProvider>,
  );
  return onNavigate;
}

afterEach(() => {
  responses.clear();
});

describe("InvoicePreview", () => {
  it("says how far past due an invoice is and what is still owed", async () => {
    responses.set("DeskInvoiceCard", deskInvoice());

    draw();

    expect(await screen.findByText("INV-1042")).toBeInTheDocument();
    expect(screen.getByText("12 days past due")).toBeInTheDocument();
    expect(screen.getAllByText(/\$1,250\.00/).length).toBeGreaterThan(0);
  });

  it("names why an open dispute was raised", async () => {
    responses.set(
      "DeskInvoiceCard",
      deskInvoice({
        disputeStatus: "Disputed",
        disputes: [
          {
            id: "idsp_1",
            status: "Open",
            reasonCode: "RateDiscrepancy",
            disputedAmount: "250.00",
            notes: "The customer agreed $1,000",
            openedAt: Math.floor(Date.now() / 1000) - 3 * DAY,
            resolvedAt: null,
            resolution: null,
            resolutionNotes: null,
            openedBy: { id: "usr_1", name: "Dana Reyes" },
            resolvedBy: null,
          },
        ],
      }),
    );

    draw();

    expect(await screen.findByText("In dispute: Rate discrepancy")).toBeInTheDocument();
  });

  it("opens the customer the invoice bills", async () => {
    responses.set("DeskInvoiceCard", deskInvoice());

    const onNavigate = draw();
    fireEvent.click(await screen.findByRole("button", { name: /Morales & Jenkins Manufacturing/ }));

    expect(onNavigate).toHaveBeenCalledWith(recordPath("customer", "cus_1"));
  });

  it("says when the invoice cannot be read", async () => {
    draw();

    expect(await screen.findByText(/Couldn't load a preview/)).toBeInTheDocument();
  });
});

describe("isPaletteRecordType", () => {
  it("knows an invoice an answer cites, beside the kinds the palette searches", () => {
    expect(isPaletteRecordType("invoice")).toBe(true);
    expect(isPaletteRecordType("shipment")).toBe(true);
    expect(isPaletteRecordType("trailer")).toBe(false);
  });
});
