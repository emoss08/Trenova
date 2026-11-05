import {
  RESOURCE_EVENT_NAME,
  parseInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { useT } from "@trenova/shared/i18n/use-t";
import { realtimeClient } from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useRef } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  invoiceDeliveryOutcomes,
  invoicePostFailure,
  isInvoiceDeliveryEvent,
  type DeliveryInvoice,
  type InvoiceDeliveryState,
} from "@/lib/invoice-delivery-realtime";

/**
 * Announces, on the invoice someone has open, the background work they are
 * waiting on: the PDF finishing and an email send settling. The page itself
 * refreshes through the realtime connection; this compares what it refetched
 * with what it last showed, and only adds the toast.
 */
export function useInvoiceDeliveryToasts(invoice: DeliveryInvoice) {
  const t = useT();
  const currentUserId = useAuthStore((s) => s.user?.id ?? null);
  const seenRef = useRef<InvoiceDeliveryState & { invoiceId: string }>({
    invoiceId: invoice.id,
    sendStatus: invoice.sendStatus,
    pdfDocumentId: invoice.pdfDocumentId ?? null,
  });
  const numberRef = useRef(invoice.number);
  const awaitingRef = useRef(false);

  useEffect(() => {
    numberRef.current = invoice.number;

    if (seenRef.current.invoiceId !== invoice.id) {
      awaitingRef.current = false;
      seenRef.current = {
        invoiceId: invoice.id,
        sendStatus: invoice.sendStatus,
        pdfDocumentId: invoice.pdfDocumentId ?? null,
      };
      return;
    }

    const { outcomes, seen } = invoiceDeliveryOutcomes(invoice, seenRef.current);
    seenRef.current = { invoiceId: invoice.id, ...seen };
    if (!awaitingRef.current || outcomes.length === 0) {
      return;
    }
    awaitingRef.current = false;

    for (const outcome of outcomes) {
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
    }
  }, [invoice, t]);

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

      if (isInvoiceDeliveryEvent(event, invoiceId)) {
        awaitingRef.current = true;
        return;
      }

      const failure = invoicePostFailure(event, invoiceId);
      if (!failure || (failure.notifiedUserId && failure.notifiedUserId === currentUserId)) {
        return;
      }

      toast.error(t("{0} could not be posted automatically", numberRef.current), {
        description: t("It is still a draft. Fix the problem and post it."),
      });
    });
  }, [invoiceId, currentUserId, t]);
}
