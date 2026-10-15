import {
  fetchFactoringAgreementOwner,
  fetchFactoringCompany,
  fetchFactoringFundingPlan,
  fetchFactoringReserveBalance,
  fetchFactoringReserveMovementPlan,
  fetchFactoringSubmission,
  fetchFactoringSubmissionPlan,
  fetchFactoringSummary,
  fetchFactoringTransport,
  fetchInvoiceFactoring,
  listFactoringAgreements,
  listFactoringReserveEntries,
  type FactoringReserveAction,
  type FactoringReserveEntryFilter,
} from "@/lib/graphql/factoring";
import { FACTORING_QUERY_SCOPE } from "@/lib/queries/factoring-roots";
import type {
  FactoringReserveMovementInput,
  RecordFactoringFundingInput,
} from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

export const factoring = createQueryKeys(FACTORING_QUERY_SCOPE, {
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
  company: (factoringCompanyId: string) => ({
    queryKey: ["company", factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      fetchFactoringCompany(factoringCompanyId, { signal }),
  }),
});
