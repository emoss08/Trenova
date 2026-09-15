import { fetchInvoiceArContext, fetchInvoicesByShipment } from "@/lib/graphql/invoice";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";
import { ar } from "./ar";

export const invoice = createQueryKeys("invoice", {
  get: (invoiceId: string) => ({
    queryKey: ["get", invoiceId],
    queryFn: async () => apiService.invoiceService.getById(invoiceId),
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
    queryFn: async () => apiService.invoiceService.getSendPlan(invoiceId),
  }),
  emailAttempts: (invoiceId: string) => ({
    queryKey: ["email-attempts", invoiceId],
    queryFn: async () => apiService.invoiceService.listEmailAttempts(invoiceId),
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
