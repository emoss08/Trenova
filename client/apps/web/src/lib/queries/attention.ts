import { AttentionSummaryDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const attention = createQueryKeys("attention", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () =>
      requestGraphQL({
        document: AttentionSummaryDocument,
        operationName: "AttentionSummary",
      }),
  }),
  recentActivity: null,
});
