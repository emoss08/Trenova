import {
  FACTORING_INVOICE_STATE_KEY,
  fetchFactoringAgreementOwner,
  fetchFactoringCompany,
  fetchFactoringNoaOwner,
  fetchFactoringSummary,
  fetchInvoiceFactoring,
  listFactoringAgreements,
  listFactoringNoas,
  type FactoringNoaFilter,
} from "@/lib/graphql/factoring";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

type QueryContext = { signal?: AbortSignal };

export const factoring = createQueryKeys("factoring", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringSummary({ signal }),
  }),
  invoice: (invoiceId: string) => ({
    queryKey: ["invoice", invoiceId],
    queryFn: async ({ signal }: QueryContext) => fetchInvoiceFactoring(invoiceId, { signal }),
  }),
  agreements: (factoringCompanyId: string) => ({
    queryKey: ["agreements", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      listFactoringAgreements(factoringCompanyId, { signal }),
  }),
  agreementOwner: (agreementId: string) => ({
    queryKey: ["agreement-owner", agreementId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringAgreementOwner(agreementId, { signal }),
  }),
  noas: (filter: FactoringNoaFilter) => ({
    queryKey: ["noas", filter],
    queryFn: async ({ signal }: QueryContext) => listFactoringNoas(filter, { signal }),
  }),
  noaOwner: (noaId: string) => ({
    queryKey: ["noa-owner", noaId],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringNoaOwner(noaId, { signal }),
  }),
  company: (factoringCompanyId: string) => ({
    queryKey: ["company", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringCompany(factoringCompanyId, { signal }),
  }),
});

/**
 * Every cache a factoring change can stale: the readiness queue and its
 * summary, an invoice's factoring section, a factor's agreements and the
 * notices listed on a factor or a customer.
 */
export function invalidateFactoringQueries(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: factoring._def });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_INVOICE_STATE_KEY] });
}
