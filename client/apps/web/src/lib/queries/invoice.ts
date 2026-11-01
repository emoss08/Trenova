import { fetchInvoiceArContext, fetchInvoicesByShipment } from "@/lib/graphql/invoice";
import { fetchDeskInvoice } from "@/lib/graphql/invoice-card";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";
import { ar } from "./ar";

export const invoice = createQueryKeys("invoice", {
  get: (invoiceId: string) => ({
    queryKey: ["get", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.invoiceService.getById(invoiceId, { signal }),
  }),
  arContext: (invoiceId: string) => ({
    queryKey: ["ar-context", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchInvoiceArContext(invoiceId, { signal }),
  }),
  byShipment: (shipmentId: string) => ({
    queryKey: ["by-shipment", shipmentId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchInvoicesByShipment(shipmentId, { signal }),
  }),
  sendPlan: (invoiceId: string) => ({
    queryKey: ["send-plan", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.invoiceService.getSendPlan(invoiceId, { signal }),
  }),
  emailAttempts: (invoiceId: string) => ({
    queryKey: ["email-attempts", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.invoiceService.listEmailAttempts(invoiceId, {}, { signal }),
  }),
  card: (invoiceId: string) => ({
    queryKey: ["card", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskInvoice(invoiceId, { signal }),
  }),
  lastEmailAttempt: (invoiceId: string) => ({
    queryKey: ["last-email-attempt", invoiceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => {
      const page = await apiService.invoiceService.listEmailAttempts(
        invoiceId,
        { limit: 1 },
        { signal },
      );
      return page.items[0] ?? null;
    },
  }),
});

/**
 * Every cache an invoice change can stale: the invoice itself, the workspace
 * list, the register, the billing queue whose items it released or posted,
 * and the AR read models built from its balance.
 */
export function invalidateInvoiceQueries(queryClient: QueryClient): void {
  for (const queryKey of [
    ["invoice"],
    ["invoice-list"],
    ["invoice-register-list"],
    ["invoice-register"],
    ["billingQueue"],
    ["billing-queue-list"],
    ar._def,
  ]) {
    void queryClient.invalidateQueries({ queryKey });
  }
}
