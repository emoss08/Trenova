import {
  getCostingControlGraphQL,
  getResolvedCostProfileGraphQL,
} from "@/lib/graphql/cost-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const costControl = createQueryKeys("costControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async ({ signal }) => getCostingControlGraphQL({ signal }),
  }),
  resolvedProfile: (asOfDate?: string) => ({
    queryKey: ["resolved-profile", asOfDate ?? "today"],
    queryFn: async ({ signal }) => getResolvedCostProfileGraphQL(asOfDate, { signal }),
  }),
});
