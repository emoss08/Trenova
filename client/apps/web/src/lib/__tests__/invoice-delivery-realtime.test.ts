import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import { RESOURCE_QUERY_KEY_MAP, invalidationFor } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";
import {
  INVOICE_RESOURCE,
  invoiceDeliveryOutcome,
  type InvoiceDeliveryState,
} from "../invoice-delivery-realtime";

/*
Fixtures follow the realtime contract the API publishes for invoices:
resource "invoice", recordId the invoice id, an action naming what changed
("pdf.generated", "send.updated", "edi.updated", "balance.updated", "updated")
and entity a snapshot { id, number, sendStatus, lastSendError, pdfDocumentId }.
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
    entity: { id: "inv_1", number: "INV-1", ...entity },
    ...overrides,
  };
}

const sending: InvoiceDeliveryState = { sendStatus: "Sending", pdfDocumentId: "doc_1" };

describe("invoiceDeliveryOutcome", () => {
  it("announces a PDF that just became ready", () => {
    const { outcome, seen } = invoiceDeliveryOutcome(
      invoiceEvent("pdf.generated", { pdfDocumentId: "doc_2", sendStatus: "NotSent" }),
      "inv_1",
      { sendStatus: "NotSent", pdfDocumentId: null },
    );
    expect(outcome).toEqual({ kind: "pdf-ready", number: "INV-1" });
    expect(seen.pdfDocumentId).toBe("doc_2");
  });

  it("announces a send settling as sent, failed or partly sent", () => {
    expect(
      invoiceDeliveryOutcome(invoiceEvent("send.updated", { sendStatus: "Sent" }), "inv_1", sending)
        .outcome,
    ).toEqual({ kind: "sent", number: "INV-1" });
    expect(
      invoiceDeliveryOutcome(
        invoiceEvent("send.updated", {
          sendStatus: "Failed",
          lastSendError: "Domain not verified",
        }),
        "inv_1",
        sending,
      ).outcome,
    ).toEqual({ kind: "failed", number: "INV-1", error: "Domain not verified" });
    expect(
      invoiceDeliveryOutcome(
        invoiceEvent("send.updated", { sendStatus: "PartiallySent" }),
        "inv_1",
        sending,
      ).outcome,
    ).toEqual({ kind: "partially-sent", number: "INV-1", error: null });
  });

  it("stays quiet when a repeated event changes nothing", () => {
    const failed = invoiceEvent("send.updated", { sendStatus: "Failed", lastSendError: "x" });
    const first = invoiceDeliveryOutcome(failed, "inv_1", sending);
    expect(first.outcome?.kind).toBe("failed");
    expect(invoiceDeliveryOutcome(failed, "inv_1", first.seen).outcome).toBeNull();
  });

  it("stays quiet for a send still in flight, another invoice or another resource", () => {
    const notSent: InvoiceDeliveryState = { sendStatus: "NotSent", pdfDocumentId: "doc_1" };
    expect(
      invoiceDeliveryOutcome(
        invoiceEvent("send.updated", { sendStatus: "Sending" }),
        "inv_1",
        notSent,
      ).outcome,
    ).toBeNull();
    expect(
      invoiceDeliveryOutcome(invoiceEvent("send.updated", { sendStatus: "Sent" }), "inv_2", sending)
        .outcome,
    ).toBeNull();
    expect(
      invoiceDeliveryOutcome(
        invoiceEvent("send.updated", { sendStatus: "Sent" }, { resource: "billing_queue" }),
        "inv_1",
        sending,
      ).outcome,
    ).toBeNull();
    expect(
      invoiceDeliveryOutcome(invoiceEvent("edi.updated", { sendStatus: "Sent" }), "inv_1", sending)
        .outcome,
    ).toBeNull();
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
