import { fetchSupportLogs } from "@/lib/graphql/support-logs";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const supportLogs = createQueryKeys("supportLogs", {
  recent: () => ({
    queryKey: ["recent"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchSupportLogs({ signal }),
  }),
});
