import {
  fetchFactoringRemittance,
  fetchFactoringRemittancePlan,
  listFactoringRemittanceLines,
  listFactoringRemittances,
  listOpenFactoringSchedules,
  type FactoringRemittanceFilter,
} from "@/lib/graphql/factoring-remittance";
import { FACTORING_REMITTANCE_QUERY_SCOPE } from "@/lib/queries/factoring-roots";
import type {
  FactoringRemittanceMatchStatus,
  FactoringRemittanceVersionInput,
} from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

export const factoringRemittance = createQueryKeys(FACTORING_REMITTANCE_QUERY_SCOPE, {
  list: (filter: FactoringRemittanceFilter) => ({
    queryKey: [filter],
    queryFn: async ({ signal }: QueryContext) => listFactoringRemittances(filter, { signal }),
  }),
  detail: (remittanceId: string) => ({
    queryKey: [remittanceId],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringRemittance(remittanceId, { signal }),
  }),
  lines: (remittanceId: string, matchStatuses: FactoringRemittanceMatchStatus[] | null) => ({
    queryKey: [remittanceId, matchStatuses ?? "all"],
    queryFn: async ({ signal }: QueryContext) =>
      listFactoringRemittanceLines(remittanceId, matchStatuses, { signal }),
  }),
  openSchedules: (factoringCompanyId: string) => ({
    queryKey: [factoringCompanyId],
    queryFn: async ({ signal }: QueryContext) =>
      listOpenFactoringSchedules(factoringCompanyId, { signal }),
  }),
  plan: (input: FactoringRemittanceVersionInput) => ({
    queryKey: [input],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringRemittancePlan(input, { signal }),
  }),
});
