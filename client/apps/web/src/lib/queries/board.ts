import { fetchDeskBoardCard } from "@/lib/graphql/desk-board-card";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const board = createQueryKeys("board", {
  deskCard: (threadId: string, artifactId: string) => ({
    queryKey: [threadId, artifactId],
    queryFn: async ({ signal }) => fetchDeskBoardCard(threadId, artifactId, { signal }),
  }),
});
