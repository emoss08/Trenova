import { fetchDeskIftaReturnCard } from "@/lib/graphql/ifta-return-desk-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const iftaReturn = createQueryKeys("iftaReturn", {
  card: (id: string) => ({
    queryKey: ["card", id],
    queryFn: async ({ signal }) => fetchDeskIftaReturnCard(id, { signal }),
  }),
});
