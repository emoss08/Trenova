import { fetchManagedPlan } from "@/lib/graphql/managed-plan";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const managedPlan = createQueryKeys("managedPlan", {
  current: () => ({
    queryKey: ["current"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchManagedPlan({ signal }),
  }),
});
