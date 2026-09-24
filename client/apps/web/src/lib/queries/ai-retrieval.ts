import {
  fetchAIRetrievalReindexEstimate,
  fetchAIRetrievalStatus,
  type AIRetrievalSourceType,
} from "@/lib/graphql/ai-retrieval";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type Signal = { signal?: AbortSignal };

export const aiRetrieval = createQueryKeys("aiRetrieval", {
  status: () => ({
    queryKey: ["status"],
    queryFn: ({ signal }: Signal) => fetchAIRetrievalStatus({ signal }),
  }),
  reindexEstimate: (sourceType: AIRetrievalSourceType) => ({
    queryKey: [sourceType],
    queryFn: ({ signal }: Signal) => fetchAIRetrievalReindexEstimate(sourceType, { signal }),
  }),
});
