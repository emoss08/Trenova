import { fetchAgentRunDetail } from "@/lib/graphql/agent-activity-tables";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentRun = createQueryKeys("agentRun", {
  detail: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentRunDetail(id, { signal }),
  }),
});
