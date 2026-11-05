import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import { RESOURCE_QUERY_KEY_MAP, invalidationFor } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";
import {
  INVOICE_RESOURCE,
  invoiceDeliveryOutcomes,
  invoicePostFailure,
  isInvoiceDeliveryEvent,
  type DeliveryInvoice,
  type InvoiceDeliveryState,
} from "../invoice-delivery-realtime";

/*
Fixtures follow the realtime contract the API publishes for invoices: resource
"invoice", recordId the invoice id, an action naming what changed, and an entity
holding identifiers alone. What an invoice now says comes from the refetched
invoice, never from the event.
*/
function invoiceEvent(
  action: string,
  entity: Record<string, unknown>,
  overrides: Partial<ResourceInvalidationEvent> = {},
): ResourceInvalidationEvent {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    resource: INVOICE_RESOURCE,
    action,
    recordId: "inv_1",
    entity: { id: "inv_1", ...entity },
    ...overrides,
  };
}

function invoice(overrides: Partial<DeliveryInvoice> = {}): DeliveryInvoice {
  return {
    id: "inv_1",
    number: "INV-1",
    sendStatus: "Sending",
    pdfDocumentId: "doc_1",
    lastSendError: null,
    ...overrides,
  } as DeliveryInvoice;
}

const sending: InvoiceDeliveryState = { sendStatus: "Sending", pdfDocumentId: "doc_1" };

describe("invoiceDeliveryOutcomes", () => {
  it("announces a PDF becoming ready", () => {
    const { outcomes, seen } = invoiceDeliveryOutcomes(
      invoice({ pdfDocumentId: "doc_2" }),
      sending,
    );

    expect(outcomes).toEqual([{ kind: "pdf-ready", number: "INV-1" }]);
    expect(seen.pdfDocumentId).toBe("doc_2");
  });

  it("announces a send settling, with the error the invoice now shows", () => {
    expect(invoiceDeliveryOutcomes(invoice({ sendStatus: "Sent" }), sending).outcomes).toEqual([
      { kind: "sent", number: "INV-1" },
    ]);
    expect(
      invoiceDeliveryOutcomes(
        invoice({ sendStatus: "PartiallySent", lastSendError: "One address bounced" }),
        sending,
      ).outcomes,
    ).toEqual([{ kind: "partially-sent", number: "INV-1", error: "One address bounced" }]);
    expect(
      invoiceDeliveryOutcomes(invoice({ sendStatus: "Failed", lastSendError: "Rejected" }), sending)
        .outcomes,
    ).toEqual([{ kind: "failed", number: "INV-1", error: "Rejected" }]);
  });

  it("announces each change once", () => {
    const failed = invoice({ sendStatus: "Failed" });
    const first = invoiceDeliveryOutcomes(failed, sending);

    expect(first.outcomes).toHaveLength(1);
    expect(invoiceDeliveryOutcomes(failed, first.seen).outcomes).toEqual([]);
  });

  it("stays quiet about a status nobody is waiting on", () => {
    expect(invoiceDeliveryOutcomes(invoice({ sendStatus: "NotSent" }), sending).outcomes).toEqual(
      [],
    );
  });

  it("stays quiet about a refetch that moved nothing", () => {
    expect(invoiceDeliveryOutcomes(invoice(), sending).outcomes).toEqual([]);
  });
});

describe("isInvoiceDeliveryEvent", () => {
  it("recognises the background work a person waits on", () => {
    expect(isInvoiceDeliveryEvent(invoiceEvent("pdf.generated", {}), "inv_1")).toBe(true);
    expect(isInvoiceDeliveryEvent(invoiceEvent("send.updated", {}), "inv_1")).toBe(true);
  });

  it("ignores other invoices and other changes, so a cache catching up stays quiet", () => {
    expect(isInvoiceDeliveryEvent(invoiceEvent("send.updated", {}), "inv_2")).toBe(false);
    expect(isInvoiceDeliveryEvent(invoiceEvent("updated", {}), "inv_1")).toBe(false);
  });
});

describe("invoicePostFailure", () => {
  it("names who was already told", () => {
    expect(
      invoicePostFailure(invoiceEvent("post.failed", { notifiedUserId: "usr_1" }), "inv_1"),
    ).toEqual({ notifiedUserId: "usr_1" });
  });

  it("reads a failure that told nobody", () => {
    expect(invoicePostFailure(invoiceEvent("post.failed", {}), "inv_1")).toEqual({
      notifiedUserId: null,
    });
  });

  it("ignores other invoices and other actions", () => {
    expect(invoicePostFailure(invoiceEvent("post.failed", {}), "inv_2")).toBeNull();
    expect(invoicePostFailure(invoiceEvent("updated", {}), "inv_1")).toBeNull();
  });
});

describe("invoice and billing queue invalidation", () => {
  it("reaches the invoice page, its lists, the billing queue and receivables", () => {
    expect(RESOURCE_QUERY_KEY_MAP[INVOICE_RESOURCE]).toEqual(
      expect.arrayContaining([
        "invoice",
        "invoice-list",
        "invoice-register",
        "billingQueue",
        "billing-queue-list",
        "ar",
        "customerPayment",
      ]),
    );
  });

  it("refetches only what is on screen for invoice and billing queue events", () => {
    expect(invalidationFor(invoiceEvent("send.updated", {})).activeOnly).toBe(true);
    expect(
      invalidationFor(invoiceEvent("updated", {}, { resource: "billing_queue" })).activeOnly,
    ).toBe(true);
  });

  it("refreshes an open billing queue when a shipment's charges change", () => {
    const { activeOnly, activeRoots } = invalidationFor(
      invoiceEvent("updated", {}, { resource: "shipments" }),
    );
    expect(activeOnly).toBe(false);
    expect(activeRoots).toEqual(["billing-queue-list", "billingQueue"]);
  });
});
