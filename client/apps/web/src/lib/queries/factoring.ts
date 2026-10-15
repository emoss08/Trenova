import {
  FACTORING_INVOICE_STATE_KEY,
  FACTORING_REMITTANCE_KEY,
  FACTORING_SUBMISSION_KEY,
  fetchFactoringAgreementOwner,
  fetchFactoringCompany,
  fetchFactoringFundingPlan,
  fetchFactoringNoaOwner,
  fetchFactoringReserveBalance,
  fetchFactoringReserveMovementPlan,
  fetchFactoringSubmission,
  fetchFactoringSubmissionPlan,
  fetchFactoringSummary,
  fetchFactoringTransport,
  fetchInvoiceFactoring,
  listFactoringAgreements,
  listFactoringNoas,
  listFactoringReserveEntries,
  type FactoringNoaFilter,
  type FactoringReserveAction,
  type FactoringReserveEntryFilter,
} from "@/lib/graphql/factoring";
import {
  fetchFactoringRemittance,
  fetchFactoringRemittancePlan,
  listFactoringRemittanceLines,
  listOpenFactoringSchedules,
  listFactoringRemittances,
  type FactoringRemittanceFilter,
} from "@/lib/graphql/factoring-remittance";
import type {
  FactoringRemittanceMatchStatus,
  FactoringRemittanceVersionInput,
  FactoringReserveMovementInput,
  RecordFactoringFundingInput,
} from "@trenova/graphql/generated/graphql";
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
  fundingPlan: (input: RecordFactoringFundingInput) => ({
    queryKey: ["funding-plan", input],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringFundingPlan(input, { signal }),
  }),
  movementPlan: (action: FactoringReserveAction, input: FactoringReserveMovementInput) => ({
    queryKey: ["movement-plan", action, input],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringReserveMovementPlan(action, input, { signal }),
  }),
  reserveBalance: (factoringCompanyId: string) => ({
    queryKey: ["reserve-balance", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringReserveBalance(factoringCompanyId, { signal }),
  }),
  reserveEntries: (filter: FactoringReserveEntryFilter) => ({
    queryKey: ["reserve-entries", filter],
    queryFn: async ({ signal }: QueryContext) => listFactoringReserveEntries(filter, { signal }),
  }),
  remittances: (filter: FactoringRemittanceFilter) => ({
    queryKey: ["remittances", filter],
    queryFn: async ({ signal }: QueryContext) => listFactoringRemittances(filter, { signal }),
  }),
  remittance: (remittanceId: string) => ({
    queryKey: ["remittance", remittanceId],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringRemittance(remittanceId, { signal }),
  }),
  remittanceLines: (
    remittanceId: string,
    matchStatuses: FactoringRemittanceMatchStatus[] | null,
  ) => ({
    queryKey: ["remittance-lines", remittanceId, matchStatuses ?? "all"],
    queryFn: async ({ signal }: QueryContext) =>
      listFactoringRemittanceLines(remittanceId, matchStatuses, { signal }),
  }),
  openSchedules: (factoringCompanyId: string) => ({
    queryKey: ["open-schedules", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      listOpenFactoringSchedules(factoringCompanyId, { signal }),
  }),
  remittancePlan: (input: FactoringRemittanceVersionInput) => ({
    queryKey: ["remittance-plan", input],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringRemittancePlan(input, { signal }),
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
 * delivery settings, the notices listed on a factor or a customer, the
 * schedules sent to factors, and what the factors hold and have paid.
 */
export function invalidateFactoringQueries(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: factoring._def });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_INVOICE_STATE_KEY] });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_SUBMISSION_KEY] });
  void queryClient.invalidateQueries({ queryKey: [FACTORING_REMITTANCE_KEY] });
}
