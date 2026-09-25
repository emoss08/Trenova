import { fetchAIAuditChainStatus, fetchAIAuditEvent } from "@/lib/graphql/ai-audit";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const aiAudit = createQueryKeys("aiAudit", {
  chainStatus: () => ({
    queryKey: ["status"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIAuditChainStatus({ signal }),
  }),
  event: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAIAuditEvent(id, { signal }),
  }),
});
