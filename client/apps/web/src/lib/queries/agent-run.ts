import { fetchAgentRunTranscript } from "@/lib/graphql/agent-activity-tables";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const agentRun = createQueryKeys("agentRun", {
  transcript: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentRunTranscript(id, { signal }),
  }),
});
