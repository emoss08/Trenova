import { fetchMyPageTourViews } from "@/lib/graphql/page-tour";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * The tours a person has been shown change only when they finish or skip one, and that write
 * updates this cache itself, so the read is made once a session.
 */
export const pageTour = createQueryKeys("pageTour", {
  views: () => ({
    queryKey: ["views"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchMyPageTourViews({ signal }),
  }),
});
