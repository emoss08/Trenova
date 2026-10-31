import { useApiMutation } from "@/hooks/use-api-mutation";
import { billingQueue } from "@/lib/queries/billing-queue";
import { apiService } from "@/services/api";
import type {
  BillingQueueInvoiceDetailsInput,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";
import { useQueryClient } from "@tanstack/react-query";
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
  };
}

/** Items whose invoice can still be shaped: not approved, posted or canceled. */
export function invoiceDetailsEditable(item: BillingQueueItem): boolean {
  return !["Approved", "Posted", "Canceled"].includes(item.status);
}

/**
 * Saves what the invoice will say (PO, memo, terms, date, packet) on the item, against
 * the version on screen. The saved item replaces the cached one, so the review, the
 * tabs and the list read the same record.
 */
export function useInvoiceDetails(item: BillingQueueItem) {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateBillingQueue();

  const mutation = useApiMutation({
    mutationFn: (changes: Partial<Omit<BillingQueueInvoiceDetailsInput, "version">>) =>
      apiService.billingQueueService.updateInvoiceDetails(item.id, {
        ...invoiceDetailsOf(item),
        ...changes,
      }),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => {
      queryClient.setQueryData<BillingQueueItem>(billingQueue.get(item.id).queryKey, (current) =>
        current ? { ...current, ...next, shipment: current.shipment } : next,
      );
      invalidate();
    },
  });

  return {
    save: mutation.mutateAsync,
    isSaving: mutation.isPending,
    editable: invoiceDetailsEditable(item),
  };
}
