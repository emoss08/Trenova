import { useApiMutation } from "@/hooks/use-api-mutation";
import { billingQueue } from "@/lib/queries/billing-queue";
import { apiService } from "@/services/api";
import type {
  BillingQueueInvoiceDetailsInput,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

/** The invoice details an item carries now, as the request that would keep them. */
export function invoiceDetailsOf(item: BillingQueueItem): BillingQueueInvoiceDetailsInput {
  return {
    version: item.version,
    poNumber: item.poNumber ?? "",
    invoiceMemo: item.invoiceMemo ?? "",
    paymentTerm: item.paymentTerm ?? "",
    invoiceDate: item.invoiceDate ?? null,
    packetDocumentIds: item.packetDocumentIds,
    packetPicked: item.packetPicked,
    packetPageRotations: item.packetPageRotations ?? [],
  };
}

/** Items whose invoice can still be shaped: not approved, posted or canceled. */
export function invoiceDetailsEditable(item: BillingQueueItem): boolean {
  return !["Approved", "Posted", "Canceled"].includes(item.status);
}

type InvoiceDetailsChanges = Partial<Omit<BillingQueueInvoiceDetailsInput, "version">>;

/**
 * Saves what the invoice will say (PO, memo, terms, date, packet) on the item.
 *
 * A change shows at once: it is written into the cached item before the request
 * goes. Saves for one item run one at a time, and each sends the item as it stands
 * on screen with the version the last save came back with, so a second click
 * while the first is still saving waits for it instead of racing it into a version
 * conflict. A failed save reloads the item, so the screen shows what was kept.
 */
export function useInvoiceDetails(item: BillingQueueItem) {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateBillingQueue();
  const itemKey = billingQueue.get(item.id).queryKey;
  const inFlight = useRef(0);
  const unsaved = useRef<InvoiceDetailsChanges>({});

  const latest = (): BillingQueueItem => ({
    ...(queryClient.getQueryData<BillingQueueItem>(itemKey) ?? item),
    ...unsaved.current,
  });

  const mutation = useApiMutation({
    scope: { id: `billing-queue-invoice-details:${item.id}` },
    mutationFn: () =>
      apiService.billingQueueService.updateInvoiceDetails(item.id, invoiceDetailsOf(latest())),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => {
      queryClient.setQueryData<BillingQueueItem>(itemKey, (current) => {
        if (!current) return next;
        // Later changes are still waiting their turn: keep them on screen and take
        // only the version this save moved the item to.
        if (inFlight.current > 1) return { ...current, version: next.version };
        return { ...current, ...next, shipment: current.shipment };
      });
    },
    onError: () => {
      unsaved.current = {};
      void queryClient.invalidateQueries({ queryKey: itemKey });
    },
    onSettled: () => {
      inFlight.current -= 1;
      if (inFlight.current === 0) {
        unsaved.current = {};
        invalidate();
      }
    },
  });

  // Written into the cache before the save is queued, not in onMutate: a scoped
  // mutation's onMutate waits for the save ahead of it, and the click would lag.
  const save = (changes: InvoiceDetailsChanges) => {
    inFlight.current += 1;
    unsaved.current = { ...unsaved.current, ...changes };
    void queryClient.cancelQueries({ queryKey: itemKey });
    queryClient.setQueryData<BillingQueueItem>(itemKey, (current) =>
      current ? { ...current, ...changes } : current,
    );
    return mutation.mutateAsync();
  };

  return {
    save,
    isSaving: mutation.isPending,
    editable: invoiceDetailsEditable(item),
  };
}
