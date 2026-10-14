import {
  FACTORING_INVOICE_STATE_KEY,
  FACTORING_SUBMISSION_KEY,
  fetchFactoringAgreementOwner,
  fetchFactoringCompany,
  fetchFactoringNoaOwner,
  fetchFactoringSubmission,
  fetchFactoringSubmissionPlan,
  fetchFactoringSummary,
  fetchFactoringTransport,
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
  submission: (submissionId: string) => ({
    queryKey: ["submission", submissionId],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringSubmission(submissionId, { signal }),
  }),
  plan: (factoringCompanyId: string, invoiceIds: readonly string[]) => ({
    queryKey: ["plan", factoringCompanyId, [...invoiceIds].sort()],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringSubmissionPlan({ factoringCompanyId, invoiceIds: [...invoiceIds] }, { signal }),
  }),
  transport: (factoringCompanyId: string) => ({
    queryKey: ["transport", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringTransport(factoringCompanyId, { signal }),
  }),
  company: (factoringCompanyId: string) => ({
    queryKey: ["company", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringCompany(factoringCompanyId, { signal }),
  }),
});

/**
 * Every cache a factoring change can stale: the readiness queue and its
 * summary, an invoice's factoring section, a factor's agreements, its
 * delivery settings, the notices listed on a factor or a customer, and the
 * schedules sent to factors.
 */
export function invalidateFactoringQueries(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: factoring._def });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_INVOICE_STATE_KEY] });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_SUBMISSION_KEY] });
}
