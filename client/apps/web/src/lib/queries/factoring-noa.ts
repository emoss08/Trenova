import {
  fetchFactoringNoaOwner,
  listFactoringNoas,
  type FactoringNoaFilter,
} from "@/lib/graphql/factoring-noa";
import { FACTORING_NOA_QUERY_SCOPE } from "@/lib/queries/factoring-roots";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

export const factoringNoa = createQueryKeys(FACTORING_NOA_QUERY_SCOPE, {
  list: (filter: FactoringNoaFilter) => ({
    queryKey: [filter],
    queryFn: async ({ signal }: QueryContext) => listFactoringNoas(filter, { signal }),
  }),
  owner: (noaId: string) => ({
    queryKey: [noaId],
    queryFn: async ({ signal }: QueryContext) => fetchFactoringNoaOwner(noaId, { signal }),
  }),
});
