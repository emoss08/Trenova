import { fetchDeskTrailer } from "@/lib/graphql/trailer-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const trailer = createQueryKeys("trailer", {
  card: (trailerId: string) => ({
    queryKey: ["card", trailerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskTrailer(trailerId, { signal }),
  }),
});
