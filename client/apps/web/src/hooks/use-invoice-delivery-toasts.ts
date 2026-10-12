import {
  RESOURCE_EVENT_NAME,
  parseInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { useT } from "@trenova/shared/i18n/use-t";
import { realtimeClient } from "@trenova/shared/services/realtime";
import type { Invoice } from "@trenova/shared/types/invoice";
import { useEffect, useRef } from "react";
import { toast } from "sonner";
import { invoiceDeliveryOutcome, type InvoiceDeliveryState } from "@/lib/invoice-delivery-realtime";

type DeliveryInvoice = Pick<Invoice, "id" | "sendStatus" | "pdfDocumentId">;

/**
 * Announces, on the invoice someone has open, the background work they are
 * waiting on: the PDF finishing and an email send settling. The page itself
 * refreshes through the realtime connection; this only adds the toast.
 */
export function useInvoiceDeliveryToasts(invoice: DeliveryInvoice) {
  const t = useT();
  const seenRef = useRef<InvoiceDeliveryState>({
    sendStatus: invoice.sendStatus,
    pdfDocumentId: invoice.pdfDocumentId ?? null,
  });

  useEffect(() => {
    seenRef.current = {
      sendStatus: invoice.sendStatus,
      pdfDocumentId: invoice.pdfDocumentId ?? null,
    };
  }, [invoice.id, invoice.sendStatus, invoice.pdfDocumentId]);

  const invoiceId = invoice.id;
  useEffect(() => {
    if (!invoiceId) {
      return undefined;
    }

    return realtimeClient.on(RESOURCE_EVENT_NAME, (data) => {
      const event = parseInvalidationEvent(data);
      if (!event) {
        return;
      }

      const { outcome, seen } = invoiceDeliveryOutcome(event, invoiceId, seenRef.current);
      seenRef.current = seen;
      if (!outcome) {
        return;
      }

      switch (outcome.kind) {
        case "pdf-ready":
          toast.success(t("{0} PDF is ready", outcome.number));
          break;
        case "sent":
          toast.success(t("{0} was emailed", outcome.number));
          break;
        case "partially-sent":
          toast.warning(t("{0} was only partly emailed", outcome.number), {
            description: outcome.error ?? undefined,
          });
          break;
        case "failed":
          toast.error(t("{0} could not be emailed", outcome.number), {
            description: outcome.error ?? undefined,
          });
          break;
      }
    });
  }, [invoiceId, t]);
}
