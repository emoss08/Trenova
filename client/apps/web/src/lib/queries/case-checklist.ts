import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { apiService } from "@/services/api";
import type { ChecklistKind } from "@/types/case-checklist";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const caseChecklist = createQueryKeys("caseChecklist", {
  list: (kind: ChecklistKind) => ({
    queryKey: [kind],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.caseChecklistService.list(kind, { signal }),
  }),
  // Customers to give a checklist of their own, from the same source the
  // customer field searches.
  customers: (query: string) => ({
    queryKey: [query],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchGraphQLSelectOptions({ resource: "CUSTOMER", query, initialLimit: 20 }, { signal }),
  }),
  // The names of the document types added steps wait on.
  documentTypes: (ids: readonly string[]) => ({
    queryKey: [[...ids].sort().join(",")],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchGraphQLSelectOptions(
        { resource: "DOCUMENT_TYPE", ids: [...ids], initialLimit: Math.max(ids.length, 1) },
        { signal },
      ),
  }),
});
